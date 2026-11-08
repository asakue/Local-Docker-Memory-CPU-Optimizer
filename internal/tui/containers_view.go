package tui

import (
	"fmt"
	"strings"

	"dockertrim/internal/analyzer"
	"dockertrim/internal/cli"

	"github.com/charmbracelet/lipgloss"
)

// RenderContainersView renders the Containers tab.
func RenderContainersView(
	containers []analyzer.ContainerMetrics,
	selectedMap map[string]bool,
	cursorIndex int,
	hostTotalRAM uint64,
	hostUsedRAM uint64,
	sortBy string,
	width int,
) string {
	if len(containers) == 0 {
		return lipgloss.NewStyle().Padding(2, 4).Render("Контейнеры не найдены или Docker Daemon не отвечает.\nНажмите 'R' для повторной попытки.")
	}

	// 1. Stats Bar
	activeCount := 0
	idleCount := 0
	var idleRAMWasted uint64
	oomCount := 0

	for _, c := range containers {
		if c.Status == "running" {
			activeCount++
		}
		if c.IsIdle {
			idleCount++
			idleRAMWasted += c.MemoryUsage
		}
		if c.IsOOMCritical || c.IsOOMWarning || c.IsHostRAMAlert {
			oomCount++
		}
	}

	statsText := fmt.Sprintf("Активных: %d/%d  │  Хост RAM: %s / %s  │  Сортировка: %s",
		activeCount, len(containers),
		cli.FormatBytes(hostUsedRAM), cli.FormatBytes(hostTotalRAM),
		sortBy,
	)

	var idleBadge string
	if idleCount > 0 {
		idleBadge = "  " + BadgeIdle.Render(fmt.Sprintf("%d IDLE (удерживают %s RAM)", idleCount, cli.FormatBytes(idleRAMWasted)))
	}

	var oomBadge string
	if oomCount > 0 {
		oomBadge = "  " + BadgeOOM.Render(fmt.Sprintf("%d OOM ALERTS", oomCount))
	}

	topBar := HeaderStyle.Width(width - 2).Render(statsText + idleBadge + oomBadge)

	// 2. Table Header
	tblHeader := fmt.Sprintf(" %-4s %-22s %-10s %-8s %-24s %-16s %-16s",
		"SEL", "КОНТЕЙНЕР", "СТАТУС", "CPU%", "RAM (ИСПОЛЬЗОВАНО/ЛИМИТ)", "NET I/O", "ИНДИКАТОРЫ",
	)
	tblHeaderStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#A0A0B0")).
		Background(lipgloss.Color("#252535")).
		Width(width - 2).
		Render(tblHeader)

	// 3. Rows Grouped by Docker-Compose Project
	var rows []string
	currentProject := ""

	for i, c := range containers {
		// Project Separator Header
		if c.Project != currentProject {
			currentProject = c.Project
			projectTitle := fmt.Sprintf(" 📦 Проект: %s ", strings.ToUpper(currentProject))
			projHeader := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00E5FF")).
				Background(lipgloss.Color("#1E1E2E")).
				MarginTop(1).
				Render(projectTitle)
			rows = append(rows, projHeader)
		}

		// Checkbox
		chk := "[ ]"
		if selectedMap[c.ID] {
			chk = "[✓]"
		}

		// Status styling
		statusStr := c.Status
		if c.Status == "running" {
			statusStr = lipgloss.NewStyle().Foreground(ColorSafe).Render("● running")
		} else if c.Status == "paused" {
			statusStr = lipgloss.NewStyle().Foreground(ColorWarning).Render("⏸ paused")
		} else {
			statusStr = lipgloss.NewStyle().Foreground(ColorMuted).Render("○ " + c.Status)
		}

		// RAM display
		var ramStr string
		if c.HasCustomLimit && c.MemoryLimit > 0 {
			ramStr = fmt.Sprintf("%s / %s (%.0f%%)", cli.FormatBytes(c.MemoryUsage), cli.FormatBytes(c.MemoryLimit), c.MemoryPercent)
		} else {
			ramStr = fmt.Sprintf("%s (No limit)", cli.FormatBytes(c.MemoryUsage))
		}

		// Net I/O
		netStr := fmt.Sprintf("↓%s ↑%s", cli.FormatBytes(c.NetRxBytes), cli.FormatBytes(c.NetTxBytes))

		// Flags & Badges
		var flags []string
		if c.IsIdle {
			flags = append(flags, BadgeIdle.Render("IDLE"))
		}
		if c.IsOOMCritical {
			flags = append(flags, BadgeOOM.Render("OOM-CRIT!"))
		} else if c.IsOOMWarning {
			flags = append(flags, lipgloss.NewStyle().Bold(true).Foreground(ColorWarning).Render("[WARN >80%]"))
		} else if c.IsHostRAMAlert {
			flags = append(flags, lipgloss.NewStyle().Foreground(ColorWarning).Render("[RAM >30%]"))
		}

		flagsStr := strings.Join(flags, " ")

		// Format row
		rowText := fmt.Sprintf(" %-4s %-22s %-16s %-8.1f%% %-24s %-16s %s",
			chk,
			truncate(c.Name, 21),
			statusStr,
			c.CPUPercent,
			ramStr,
			netStr,
			flagsStr,
		)

		// Cursor and Selection highlighting
		if i == cursorIndex {
			rows = append(rows, RowCursorStyle.Width(width-2).Render("> "+rowText))
		} else if selectedMap[c.ID] {
			rows = append(rows, RowSelectedStyle.Width(width-2).Render("  "+rowText))
		} else {
			rows = append(rows, "  "+rowText)
		}
	}

	tableBody := strings.Join(rows, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, topBar, tblHeaderStyled, tableBody)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}

