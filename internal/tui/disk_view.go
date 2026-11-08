package tui

import (
	"fmt"
	"strings"

	"dockertrim/internal/analyzer"
	"dockertrim/internal/cli"

	"github.com/charmbracelet/lipgloss"
)

// RenderDiskView renders the Disk & Reclaimable Space tab.
func RenderDiskView(
	disk analyzer.DiskAuditSummary,
	cursorIndex int,
	width int,
) string {
	if len(disk.Items) == 0 {
		return lipgloss.NewStyle().Padding(2, 4).Render("Аудит диска завершен: объекты для очистки не обнаружены.\nНажмите 'R' для повторного сканирования.")
	}

	// 1. Top Summary Banner
	var selectedBytes int64
	for _, it := range disk.Items {
		if it.Selected {
			selectedBytes += it.Size
		}
	}

	totalDockerStr := cli.FormatBytes(uint64(disk.TotalDockerSpace))
	safeReclaimStr := cli.FormatBytes(uint64(disk.SafeReclaimable))
	selectedStr := cli.FormatBytes(uint64(selectedBytes))

	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(width - 4).
		Render(fmt.Sprintf(
			"💾 ВСЕГО ЗАНИМАЕТ DOCKER: %s\n"+
				"🟢 ДОСТУПНО ДЛЯ БЕЗОПАСНОГО ОСВОБОЖДЕНИЯ: %s\n"+
				"✨ ВЫБРАНО К ОЧИСТКЕ СЕЙЧАС: %s",
			lipgloss.NewStyle().Bold(true).Render(totalDockerStr),
			lipgloss.NewStyle().Bold(true).Foreground(ColorSafe).Render(safeReclaimStr),
			lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(selectedStr),
		))

	// 2. Interactive Category List
	var rows []string
	rows = append(rows, lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#252535")).
		Padding(0, 1).
		Width(width-4).
		Render(" КАТЕГОРИИ ДЛЯ ОЧИСТКИ (Пробел — отметить / снять, Enter — очистить выбранное)"),
	)

	for i, item := range disk.Items {
		chk := "[ ]"
		if item.Selected {
			chk = "[✓]"
		}

		var riskBadge string
		switch item.Risk {
		case analyzer.RiskSafe:
			riskBadge = lipgloss.NewStyle().Foreground(ColorSafe).Render("[Безопасно]")
		case analyzer.RiskCaution:
			riskBadge = lipgloss.NewStyle().Foreground(ColorWarning).Render("[Осторожно]")
		case analyzer.RiskDangerous:
			riskBadge = lipgloss.NewStyle().Foreground(ColorDanger).Render("[ОПАСНО: ТОМА]")
		}

		sizeStr := cli.FormatBytes(uint64(item.Size))

		rowContent := fmt.Sprintf("%-4s %-16s %-32s %10s   %s",
			chk,
			riskBadge,
			item.Name,
			sizeStr,
			lipgloss.NewStyle().Foreground(ColorMuted).Render(item.Description),
		)

		if i == cursorIndex {
			rows = append(rows, RowCursorStyle.Width(width-4).Render("> "+rowContent))
		} else if item.Selected {
			rows = append(rows, RowSelectedStyle.Width(width-4).Render("  "+rowContent))
		} else {
			rows = append(rows, "  "+rowContent)
		}
	}

	// 3. Safety Notice Banner
	safetyNote := lipgloss.NewStyle().
		Foreground(ColorMuted).
		MarginTop(1).
		Render("🛡️  Защита данных: Непривязанные тома (Unused Volumes) исключены из автоочистки,\n   чтобы предотвратить случайное удаление локальных баз данных (PostgreSQL, MySQL и др.).")

	itemsList := strings.Join(rows, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, headerBox, "", itemsList, safetyNote)
}

