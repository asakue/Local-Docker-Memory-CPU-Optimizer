package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"dockertrim/internal/analyzer"

	"github.com/charmbracelet/lipgloss"
)

// FormatBytes formats bytes into human-readable string (B, KB, MB, GB, TB).
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// PrintJSON outputs the system report in JSON format.
func PrintJSON(report analyzer.SystemReport) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// PrintSummary outputs a rich, formatted terminal overview of Docker resource health.
func PrintSummary(report analyzer.SystemReport) {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00D7D7")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#5F00D7")).
		Padding(0, 1)

	badgeIdle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#FFAF00")).
		Padding(0, 1)

	badgeOOM := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#FF0055")).
		Padding(0, 1)

	badgeSafe := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF87"))

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#767676"))

	fmt.Println(headerStyle.Render("🐳 DOCKERTRIM — Ресурсная гигиена Docker"))
	fmt.Printf("Хост RAM: %s / %s | Контейнеров: %d (активных: %d)\n\n",
		FormatBytes(report.HostUsedRAM),
		FormatBytes(report.HostTotalRAM),
		report.ContainersTotal,
		report.ContainersActive,
	)

	// 1. Idle Containers & Leaks
	fmt.Println(sectionStyle.Render(" 1. ЗАБЫТЫЕ И ПРОСТАИВАЮЩИЕ КОНТЕЙНЕРЫ (IDLE / STALE) "))
	if report.IdleContainers == 0 {
		fmt.Println("  ✅ Нет простаивающих контейнеров без нагрузки.")
	} else {
		fmt.Printf("  Найдено %s контейнеров, удерживающих %s памяти:\n\n",
			badgeIdle.Render(fmt.Sprintf("%d IDLE", report.IdleContainers)),
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFAF00")).Render(FormatBytes(report.IdleRAMWasted)),
		)

		for _, c := range report.Containers {
			if c.IsIdle {
				idleHours := c.IdleDuration.Hours()
				fmt.Printf("  • %-24s [%s]  CPU: %.1f%%  RAM: %s  (в простое: %.1f ч)\n",
					lipgloss.NewStyle().Bold(true).Render(c.Name),
					dimStyle.Render(c.Project),
					c.CPUPercent,
					FormatBytes(c.MemoryUsage),
					idleHours,
				)
			}
		}
	}
	fmt.Println()

	// 2. OOM Alerts
	if report.OOMAlertCount > 0 {
		fmt.Println(sectionStyle.Render(" 2. ПРЕДУПРЕЖДЕНИЯ О ПАМЯТИ (OOM ALERTS) "))
		for _, c := range report.Containers {
			if c.IsOOMCritical {
				fmt.Printf("  • %s %-24s Использование: %.1f%% от лимита (%s / %s)\n",
					badgeOOM.Render("CRITICAL OOM"),
					c.Name,
					c.MemoryPercent,
					FormatBytes(c.MemoryUsage),
					FormatBytes(c.MemoryLimit),
				)
			} else if c.IsOOMWarning {
				fmt.Printf("  • %s %-24s Использование: %.1f%% от лимита (%s / %s)\n",
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFAF00")).Render("[WARN 80%+]"),
					c.Name,
					c.MemoryPercent,
					FormatBytes(c.MemoryUsage),
					FormatBytes(c.MemoryLimit),
				)
			} else if c.IsHostRAMAlert {
				fmt.Printf("  • %s %-24s Без лимита, занимает %.1f%% RAM хоста (%s)\n",
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D78700")).Render("[HOST RAM >30%]"),
					c.Name,
					c.MemoryPercent,
					FormatBytes(c.MemoryUsage),
				)
			}
		}
		fmt.Println()
	}

	// 3. Disk Reclaimable Audit
	fmt.Println(sectionStyle.Render(" 3. АУДИТ ДИСКОВОГО ПРОСТРАНСТВА (RECLAIMABLE DISK) "))
	fmt.Printf("  Общий объем Docker на диске: %s\n", FormatBytes(uint64(report.Disk.TotalDockerSpace)))
	fmt.Printf("  Доступно для БЕЗОПАСНОГО освобождения: %s\n\n",
		badgeSafe.Render(FormatBytes(uint64(report.Disk.SafeReclaimable))),
	)

	for _, item := range report.Disk.Items {
		riskTag := ""
		switch item.Risk {
		case analyzer.RiskSafe:
			riskTag = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87")).Render("[БЕЗОПАСНО]")
		case analyzer.RiskCaution:
			riskTag = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAF00")).Render("[ОСТОРОЖНО]")
		case analyzer.RiskDangerous:
			riskTag = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("[ОПАСНО: ТОМА]")
		}

		fmt.Printf("  %-16s %-35s %10s  %s\n",
			riskTag,
			item.Name,
			FormatBytes(uint64(item.Size)),
			dimStyle.Render(item.Description),
		)
	}

	fmt.Println()
	fmt.Println(dimStyle.Render(strings.Repeat("─", 75)))
	fmt.Println("Запустите " + lipgloss.NewStyle().Bold(true).Render("dockertrim") + " без флагов для интерактивного TUI управления.")
}

