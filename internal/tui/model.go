package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"dockertrim/internal/analyzer"
	"dockertrim/internal/cli"
	"dockertrim/internal/docker"
	"dockertrim/internal/logger"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/mem"
)

// Messages
type tickMsg time.Time
type dataLoadedMsg struct {
	containers []analyzer.ContainerMetrics
	disk       analyzer.DiskAuditSummary
	hostTotal  uint64
	hostUsed   uint64
	err        error
}

type actionDoneMsg struct {
	message    string
	isError    bool
	showModal  bool
	modalTitle string
	modalText  string
}

// Model is the main Bubble Tea state.
type Model struct {
	dockerSvc          *docker.DockerService
	keys               KeyMap
	currentTab         ActiveTab
	containers         []analyzer.ContainerMetrics
	diskSummary        analyzer.DiskAuditSummary
	selectedContainers map[string]bool
	containersCursor   int
	diskCursor         int
	sortBy             string // "mem", "cpu", "name"
	hostTotalRAM       uint64
	hostUsedRAM        uint64
	modal              ModalState
	toastMessage       string
	toastIsError       bool
	idleCfg            analyzer.IdleConfig
	oomCfg             analyzer.OOMConfig
	width              int
	height             int
	quitting           bool
	isDemo             bool
	spinner            spinner.Model
	isProcessing       bool
	processingMsg      string
}

// NewModel creates a new TUI model instance.
func NewModel(dockerSvc *docker.DockerService, idleCfg analyzer.IdleConfig, oomCfg analyzer.OOMConfig, isDemo bool) Model {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	return Model{
		dockerSvc:          dockerSvc,
		keys:               DefaultKeyMap(),
		currentTab:         TabContainers,
		selectedContainers: make(map[string]bool),
		sortBy:             "mem",
		idleCfg:            idleCfg,
		oomCfg:             oomCfg,
		width:              100,
		height:             30,
		isDemo:             isDemo,
		spinner:            sp,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Init starts the periodic ticker and loads initial data.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadDataCmd(), tickCmd())
}

func (m Model) loadDataCmd() tea.Cmd {
	return func() tea.Msg {
		var hostTotal, hostUsed uint64
		v, err := mem.VirtualMemory()
		if err == nil {
			hostTotal = v.Total
			hostUsed = v.Used
		} else {
			hostTotal = 16 * 1024 * 1024 * 1024
			hostUsed = 8 * 1024 * 1024 * 1024
		}

		if m.isDemo || m.dockerSvc == nil {
			containers, disk := docker.GenerateMockData(m.idleCfg, m.oomCfg, hostTotal)
			return dataLoadedMsg{
				containers: containers,
				disk:       disk,
				hostTotal:  hostTotal,
				hostUsed:   hostUsed,
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		containers, err := m.dockerSvc.ListContainers(ctx)
		if err != nil {
			return dataLoadedMsg{err: err}
		}

		containers = m.dockerSvc.EnrichContainerMetrics(ctx, containers, hostTotal)
		now := time.Now()
		for i := range containers {
			analyzer.AnalyzeIdle(&containers[i], m.idleCfg, now)
			analyzer.AnalyzeMemory(&containers[i], hostTotal, m.oomCfg)
		}

		disk, err := m.dockerSvc.AuditDiskSpace(ctx)
		if err != nil {
			return dataLoadedMsg{containers: containers, hostTotal: hostTotal, hostUsed: hostUsed, err: err}
		}

		return dataLoadedMsg{
			containers: containers,
			disk:       disk,
			hostTotal:  hostTotal,
			hostUsed:   hostUsed,
		}
	}
}

// Update handles incoming messages and user input.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if !m.modal.Active && !m.isProcessing {
			return m, tea.Batch(m.loadDataCmd(), tickCmd())
		}
		return m, tickCmd()

	case spinner.TickMsg:
		if m.isProcessing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case dataLoadedMsg:
		if msg.err != nil {
			m.toastMessage = "Ошибка подключения к Docker: " + msg.err.Error()
			m.toastIsError = true
		} else {
			m.containers = msg.containers
			m.hostTotalRAM = msg.hostTotal
			m.hostUsedRAM = msg.hostUsed

			// Preserve existing selected state for disk items
			if len(m.diskSummary.Items) > 0 {
				oldSelected := make(map[string]bool)
				for _, it := range m.diskSummary.Items {
					oldSelected[it.ID] = it.Selected
				}
				for i := range msg.disk.Items {
					if sel, ok := oldSelected[msg.disk.Items[i].ID]; ok {
						msg.disk.Items[i].Selected = sel
					}
				}
			}
			m.diskSummary = msg.disk
			m.sortContainers()

			// Adjust cursors
			if m.containersCursor >= len(m.containers) && len(m.containers) > 0 {
				m.containersCursor = len(m.containers) - 1
			}
			if m.diskCursor >= len(m.diskSummary.Items) && len(m.diskSummary.Items) > 0 {
				m.diskCursor = len(m.diskSummary.Items) - 1
			}
		}
		return m, nil

	case actionDoneMsg:
		m.isProcessing = false
		m.toastMessage = msg.message
		m.toastIsError = msg.isError
		if msg.showModal {
			m.modal = ModalState{
				Active:   true,
				IsViewer: true,
				Title:    msg.modalTitle,
				ViewText: msg.modalText,
			}
		} else {
			m.modal.Active = false
		}
		// Clear selection if action succeeded
		if !msg.isError && m.modal.ActionType != "prune" {
			m.selectedContainers = make(map[string]bool)
		}
		return m, m.loadDataCmd()

	case tea.KeyMsg:
		// If an operation is currently processing, ignore all keys except quit
		if m.isProcessing {
			if key.Matches(msg, m.keys.Quit) {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		// Modal Key Handling
		if m.modal.Active {
			if m.modal.IsViewer {
				// Any cancel/confirm key closes the viewer
				switch {
				case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Confirm), key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Select):
					m.modal.Active = false
					return m, nil
				default:
					return m, nil
				}
			}

			switch {
			case key.Matches(msg, m.keys.Confirm):
				m.isProcessing = true
				if m.modal.ActionType == "prune" {
					m.processingMsg = "Очистка кэша сборок и неиспользуемых слоев Docker..."
				} else {
					m.processingMsg = "Выполняется действие над выбранными контейнерами..."
				}
				return m, tea.Batch(m.executeModalActionCmd(), m.spinner.Tick)
			case key.Matches(msg, m.keys.Cancel):
				m.modal.Active = false
				return m, nil
			default:
				return m, nil
			}
		}

		// Global Key Handling
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Tab):
			if m.currentTab == TabContainers {
				m.currentTab = TabDisk
			} else {
				m.currentTab = TabContainers
			}
			return m, nil

		case key.Matches(msg, m.keys.Tab1):
			m.currentTab = TabContainers
			return m, nil

		case key.Matches(msg, m.keys.Tab2):
			m.currentTab = TabDisk
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			m.toastMessage = "Обновление данных..."
			m.toastIsError = false
			return m, m.loadDataCmd()
		}

		// Tab-Specific Key Handling
		if m.currentTab == TabContainers {
			return m.handleContainersKeys(msg)
		} else {
			return m.handleDiskKeys(msg)
		}
	}

	return m, nil
}

func (m *Model) sortContainers() {
	switch m.sortBy {
	case "mem":
		sort.SliceStable(m.containers, func(i, j int) bool {
			// Sort by project first for grouping, then by RAM desc
			if m.containers[i].Project != m.containers[j].Project {
				return m.containers[i].Project < m.containers[j].Project
			}
			return m.containers[i].MemoryUsage > m.containers[j].MemoryUsage
		})
	case "cpu":
		sort.SliceStable(m.containers, func(i, j int) bool {
			if m.containers[i].Project != m.containers[j].Project {
				return m.containers[i].Project < m.containers[j].Project
			}
			return m.containers[i].CPUPercent > m.containers[j].CPUPercent
		})
	case "name":
		sort.SliceStable(m.containers, func(i, j int) bool {
			if m.containers[i].Project != m.containers[j].Project {
				return m.containers[i].Project < m.containers[j].Project
			}
			return m.containers[i].Name < m.containers[j].Name
		})
	}
}

func (m Model) handleContainersKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.containersCursor > 0 {
			m.containersCursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.containersCursor < len(m.containers)-1 {
			m.containersCursor++
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		if len(m.containers) > 0 {
			id := m.containers[m.containersCursor].ID
			m.selectedContainers[id] = !m.selectedContainers[id]
			if !m.selectedContainers[id] {
				delete(m.selectedContainers, id)
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.SelectAll):
		if len(m.selectedContainers) == len(m.containers) {
			m.selectedContainers = make(map[string]bool)
		} else {
			for _, c := range m.containers {
				m.selectedContainers[c.ID] = true
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.SortMem):
		m.sortBy = "mem"
		m.sortContainers()
		return m, nil

	case key.Matches(msg, m.keys.SortCPU):
		m.sortBy = "cpu"
		m.sortContainers()
		return m, nil

	case key.Matches(msg, m.keys.SortName):
		m.sortBy = "name"
		m.sortContainers()
		return m, nil

	case key.Matches(msg, m.keys.Stop):
		return m.promptContainerAction("stop", "Остановить контейнеры?", false)

	case key.Matches(msg, m.keys.Pause):
		return m.promptContainerAction("pause", "Поставить на паузу / снять с паузы?", false)

	case key.Matches(msg, m.keys.Restart):
		return m.promptContainerAction("restart", "Перезапустить контейнеры?", false)

	case key.Matches(msg, m.keys.Remove):
		return m.promptContainerAction("remove", "УДАЛИТЬ контейнеры навсегда?", true)

	case key.Matches(msg, m.keys.CleanDisk):
		m.currentTab = TabDisk
		return m, nil

	case key.Matches(msg, m.keys.Logs):
		if len(m.containers) == 0 || m.containersCursor >= len(m.containers) {
			return m, nil
		}
		c := m.containers[m.containersCursor]
		var logs string
		if m.isDemo || m.dockerSvc == nil {
			logs = docker.GenerateMockLogs(c.Name)
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var err error
			logs, err = m.dockerSvc.FetchContainerLogs(ctx, c.ID, 40)
			if err != nil {
				logs = "Ошибка чтения логов: " + err.Error()
			}
		}
		m.modal = ModalState{
			Active:   true,
			IsViewer: true,
			Title:    "Последние логи: " + c.Name,
			ViewText: logs,
		}
		return m, nil

	case key.Matches(msg, m.keys.Inspect):
		if len(m.containers) == 0 || m.containersCursor >= len(m.containers) {
			return m, nil
		}
		c := m.containers[m.containersCursor]
		var detailsText string
		if m.isDemo || m.dockerSvc == nil {
			d := docker.GenerateMockInspect(c)
			detailsText = formatInspectDetails(d)
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			d, err := m.dockerSvc.InspectContainerDetails(ctx, c.ID)
			if err != nil {
				detailsText = "Ошибка инспекции: " + err.Error()
			} else {
				detailsText = formatInspectDetails(d)
			}
		}
		m.modal = ModalState{
			Active:   true,
			IsViewer: true,
			Title:    "Инспекция контейнера: " + c.Name,
			ViewText: detailsText,
		}
		return m, nil
	}

	return m, nil
}

func (m Model) promptContainerAction(actionType, title string, dangerous bool) (tea.Model, tea.Cmd) {
	targetIDs, targetNames := m.getTargets()
	if len(targetIDs) == 0 {
		m.toastMessage = "Сначала выберите контейнер (пробел) или наведите курсор!"
		m.toastIsError = true
		return m, nil
	}

	m.modal = ModalState{
		Active:      true,
		Title:       title,
		Message:     fmt.Sprintf("Действие будет применено к %d контейнерам:", len(targetIDs)),
		ActionType:  actionType,
		TargetIDs:   targetIDs,
		TargetNames: targetNames,
		IsDangerous: dangerous,
	}
	return m, nil
}

func (m Model) getTargets() ([]string, []string) {
	var ids []string
	var names []string

	if len(m.selectedContainers) > 0 {
		for _, c := range m.containers {
			if m.selectedContainers[c.ID] {
				ids = append(ids, c.ID)
				names = append(names, c.Name)
			}
		}
	} else if len(m.containers) > 0 && m.containersCursor < len(m.containers) {
		c := m.containers[m.containersCursor]
		ids = append(ids, c.ID)
		names = append(names, c.Name)
	}

	return ids, names
}

func (m Model) handleDiskKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.diskCursor > 0 {
			m.diskCursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.diskCursor < len(m.diskSummary.Items)-1 {
			m.diskCursor++
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		if len(m.diskSummary.Items) > 0 {
			m.diskSummary.Items[m.diskCursor].Selected = !m.diskSummary.Items[m.diskCursor].Selected
		}
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		// Enter triggers prune confirmation modal
		var targetIDs, targetNames []string
		hasVolumes := false
		var totalBytes int64

		for _, it := range m.diskSummary.Items {
			if it.Selected {
				targetIDs = append(targetIDs, it.ID)
				targetNames = append(targetNames, fmt.Sprintf("%s (%s)", it.Name, cli.FormatBytes(uint64(it.Size))))
				totalBytes += it.Size
				if it.Risk == analyzer.RiskDangerous {
					hasVolumes = true
				}
			}
		}

		if len(targetIDs) == 0 {
			m.toastMessage = "Выберите хотя бы одну категорию (пробел) для очистки!"
			m.toastIsError = true
			return m, nil
		}

		title := fmt.Sprintf("Очистить выбранные категории и освободить ~%s?", cli.FormatBytes(uint64(totalBytes)))
		msgText := "Выбранные элементы будут удалены из кэша/хранилища Docker:"
		if hasVolumes {
			msgText += "\n⚠️ ВНИМАНИЕ: Выбран пункт с ТОМАМИ (Volumes). Убедитесь, что там нет нужных БД!"
		}

		m.modal = ModalState{
			Active:      true,
			Title:       title,
			Message:     msgText,
			ActionType:  "prune",
			TargetIDs:   targetIDs,
			TargetNames: targetNames,
			IsDangerous: hasVolumes,
		}
		return m, nil
	}

	return m, nil
}

func (m Model) executeModalActionCmd() tea.Cmd {
	modal := m.modal
	isDemo := m.isDemo
	return func() tea.Msg {
		if isDemo {
			time.Sleep(1200 * time.Millisecond) // realistic animated delay
			switch modal.ActionType {
			case "stop":
				return actionDoneMsg{message: fmt.Sprintf("[DEMO] Остановлено контейнеров: %d", len(modal.TargetIDs))}
			case "pause":
				return actionDoneMsg{message: fmt.Sprintf("[DEMO] Поставлено на паузу контейнеров: %d", len(modal.TargetIDs))}
			case "restart":
				return actionDoneMsg{message: fmt.Sprintf("[DEMO] Перезапущено контейнеров: %d", len(modal.TargetIDs))}
			case "remove":
				return actionDoneMsg{message: fmt.Sprintf("[DEMO] Удалено контейнеров: %d", len(modal.TargetIDs))}
			case "prune":
				return actionDoneMsg{
					message:    "[DEMO] Очистка успешно завершена: освобождено 11.7 GB",
					showModal:  true,
					modalTitle: "[ДЕМО] Отчет о результатах очистки диска",
					modalText:  "✅ Docker Build Cache: освобождено 8.7 GB (48 кэш-слоев)\n✅ Dangling Images: освобождено 2.3 GB (14 слоев)\n✅ Stopped Containers: освобождено 680.0 MB (5 контейнеров)\n\n🎉 Всего фактически освобождено: 11.7 GB\nЛог операций: dockertrim.log",
				}
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		switch modal.ActionType {
		case "stop":
			successCount := 0
			for _, id := range modal.TargetIDs {
				if err := m.dockerSvc.StopContainer(ctx, id); err == nil {
					successCount++
				}
			}
			return actionDoneMsg{
				message: fmt.Sprintf("Остановлено контейнеров: %d из %d", successCount, len(modal.TargetIDs)),
				isError: successCount == 0 && len(modal.TargetIDs) > 0,
			}

		case "pause":
			for _, id := range modal.TargetIDs {
				_ = m.dockerSvc.PauseContainer(ctx, id)
			}
			return actionDoneMsg{
				message: fmt.Sprintf("Команда паузы выполнена для %d контейнеров", len(modal.TargetIDs)),
			}

		case "restart":
			for _, id := range modal.TargetIDs {
				_ = m.dockerSvc.RestartContainer(ctx, id)
			}
			return actionDoneMsg{
				message: fmt.Sprintf("Перезапущено контейнеров: %d", len(modal.TargetIDs)),
			}

		case "remove":
			removed := 0
			for _, id := range modal.TargetIDs {
				if err := m.dockerSvc.RemoveContainer(ctx, id); err == nil {
					removed++
				}
			}
			return actionDoneMsg{
				message: fmt.Sprintf("Удалено контейнеров: %d", removed),
			}

		case "prune":
			var totalReclaimed uint64
			var details []string
			var errCount int

			for _, catID := range modal.TargetIDs {
				logger.Info("User confirmed prune for category: %s", catID)
				report, err := m.dockerSvc.PruneCategory(ctx, catID)
				if err == nil {
					totalReclaimed += report.SpaceReclaimed
					details = append(details, fmt.Sprintf("✅ %-25s Освобождено: %s (удалено объектов: %d)",
						report.Category, cli.FormatBytes(report.SpaceReclaimed), report.ItemsDeleted))
				} else {
					errCount++
					logger.Error("Prune category %s error: %v", catID, err)
					details = append(details, fmt.Sprintf("❌ %-25s Ошибка: %v", catID, err))
				}
			}

			summaryLine := fmt.Sprintf("\n🎉 Всего фактически освобождено: %s\nЖурнал операций сохранен в %s",
				cli.FormatBytes(totalReclaimed), logger.GetLogPath())

			modalBody := strings.Join(details, "\n") + summaryLine

			msgText := fmt.Sprintf("Очистка завершена: освобождено %s", cli.FormatBytes(totalReclaimed))
			if errCount > 0 {
				msgText = fmt.Sprintf("Очистка завершена с предупреждениями (ошибок: %d, см. dockertrim.log)", errCount)
			}

			return actionDoneMsg{
				message:    msgText,
				isError:    errCount > 0 && totalReclaimed == 0,
				showModal:  true,
				modalTitle: "Отчет о результатах очистки диска",
				modalText:  modalBody,
			}
		}

		return actionDoneMsg{message: "Действие выполнено"}
	}
}

// View renders the complete UI screen.
func (m Model) View() string {
	if m.quitting {
		return "До свидания! Ресурсная гигиена Docker сохранена. 🐳\n"
	}

	// 1. Navigation Tabs
	tabs := RenderTabs(m.currentTab, len(m.containers), m.diskSummary.SafeReclaimable)

	// 2. Active Tab Content
	var content string
	if m.currentTab == TabContainers {
		content = RenderContainersView(
			m.containers,
			m.selectedContainers,
			m.containersCursor,
			m.hostTotalRAM,
			m.hostUsedRAM,
			m.sortBy,
			m.width,
		)
	} else {
		content = RenderDiskView(
			m.diskSummary,
			m.diskCursor,
			m.width,
		)
	}

	// 3. Toast Notifications
	var toast string
	if m.toastMessage != "" {
		if m.toastIsError {
			toast = "\n" + ErrorToast.Render("❌ "+m.toastMessage)
		} else {
			toast = "\n" + SuccessToast.Render("✨ "+m.toastMessage)
		}
	}

	// 4. Modal or Processing Overlay
	if m.isProcessing {
		procOverlay := RenderProcessingModal(m.spinner.View(), m.processingMsg, m.width)
		content = lipgloss.JoinVertical(lipgloss.Center, content, "\n", procOverlay)
	} else if m.modal.Active {
		modalOverlay := RenderModal(m.modal, m.width)
		content = lipgloss.JoinVertical(lipgloss.Center, content, "\n", modalOverlay)
	}

	// 5. Contextual Footer
	var footerText string
	if m.currentTab == TabContainers {
		selCount := len(m.selectedContainers)
		selInfo := ""
		if selCount > 0 {
			selInfo = fmt.Sprintf(" | Выбрано: %d", selCount)
		}
		footerText = fmt.Sprintf(
			"Навигация: %s %s │ Выбор: %s %s%s │ Действия: %s %s %s %s │ Сортировка: %s %s %s │ %s",
			KeyStyle.Render("↑/↓/j/k"), KeyStyle.Render("Tab"),
			KeyStyle.Render("Space/v"), KeyStyle.Render("a(все)"), selInfo,
			KeyStyle.Render("s(stop)"), KeyStyle.Render("p(pause)"), KeyStyle.Render("r(restart)"), KeyStyle.Render("x(remove)"),
			KeyStyle.Render("m(RAM)"), KeyStyle.Render("u(CPU)"), KeyStyle.Render("n(Имя)"),
			KeyStyle.Render("q(выход)"),
		)
	} else {
		footerText = fmt.Sprintf(
			"Навигация: %s %s │ Чекбокс: %s │ Очистка: %s │ %s",
			KeyStyle.Render("↑/↓"), KeyStyle.Render("Tab"),
			KeyStyle.Render("Space"),
			KeyStyle.Render("Enter(начать prune)"),
			KeyStyle.Render("q(выход)"),
		)
	}
	footer := FooterStyle.Render(footerText)

	return lipgloss.JoinVertical(lipgloss.Left, tabs, "", content, toast, footer)
}

func formatInspectDetails(d docker.ContainerDetails) string {
	portsStr := "(нет привязанных портов)"
	if len(d.Ports) > 0 {
		portsStr = strings.Join(d.Ports, ", ")
	}

	mountsStr := "(нет смонтированных томов)"
	if len(d.Mounts) > 0 {
		mountsStr = strings.Join(d.Mounts, "\n  ")
	}

	envStr := "(переменные не заданы)"
	if len(d.Env) > 0 {
		maxEnv := 6
		var envList []string
		for i, e := range d.Env {
			if i >= maxEnv {
				envList = append(envList, fmt.Sprintf("...и ещё %d переменных", len(d.Env)-maxEnv))
				break
			}
			envList = append(envList, e)
		}
		envStr = strings.Join(envList, "\n  ")
	}

	return fmt.Sprintf(
		"ID: %s\n"+
			"Образ: %s\n"+
			"IP-адрес: %s\n"+
			"Команда: %s\n"+
			"Перезапусков: %d\n\n"+
			"Порты:\n  %s\n\n"+
			"Тома (Mounts):\n  %s\n\n"+
			"Переменные окружения (Env):\n  %s",
		d.ID[:12],
		d.Image,
		d.IPAddress,
		d.Command,
		d.RestartCount,
		portsStr,
		mountsStr,
		envStr,
	)
}
