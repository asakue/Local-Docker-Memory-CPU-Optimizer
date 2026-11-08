package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ModalState represents the state of a confirmation dialog or viewer.
type ModalState struct {
	Active      bool
	Title       string
	Message     string
	ActionType  string   // "stop", "pause", "restart", "remove", "prune", "logs", "inspect"
	TargetIDs   []string // container IDs or category IDs
	TargetNames []string // container names or category names
	IsDangerous bool
	IsViewer    bool
	ViewText    string
}

// RenderModal renders a centered popup dialog.
func RenderModal(modal ModalState, screenWidth int) string {
	if !modal.Active {
		return ""
	}

	if modal.IsViewer {
		title := lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Render("📋  " + modal.Title)

		closeBtn := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorSecondary).
			Padding(0, 2).
			Render("[ Esc / Enter / q ] Закрыть")

		content := lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0")).Render(modal.ViewText),
			"",
			closeBtn,
		)

		dialog := ModalBoxStyle.Width(76).Render(content)
		return lipgloss.PlaceHorizontal(screenWidth, lipgloss.Center, dialog)
	}

	titleColor := ColorPrimary
	if modal.IsDangerous {
		titleColor = ColorDanger
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(titleColor).
		Render("⚠️  " + modal.Title)

	var targetsList string
	if len(modal.TargetNames) > 0 {
		maxShow := 6
		var shown []string
		for i, name := range modal.TargetNames {
			if i >= maxShow {
				shown = append(shown, fmt.Sprintf("...и ещё %d элементов", len(modal.TargetNames)-maxShow))
				break
			}
			shown = append(shown, "  • "+name)
		}
		targetsList = "\n" + strings.Join(shown, "\n") + "\n"
	}

	btnYes := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(titleColor).
		Padding(0, 2).
		Render("[ Y ] Подтвердить")

	btnNo := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(ColorMuted).
		Padding(0, 2).
		Render("[ Esc / N ] Отмена")

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, btnYes, "   ", btnNo)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		modal.Message,
		targetsList,
		"",
		buttons,
	)

	dialog := ModalBoxStyle.Width(60).Render(content)
	return lipgloss.PlaceHorizontal(screenWidth, lipgloss.Center, dialog)
}

// RenderProcessingModal renders an animated progress dialog during long-running tasks.
func RenderProcessingModal(spView, message string, screenWidth int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render("🧹  ВЫПОЛНЯЕТСЯ ОЧИСТКА DOCKER...")

	spinnerLine := fmt.Sprintf("  %s  %s",
		spView,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF87")).Render(message),
	)

	subtext := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render("  Пожалуйста, подождите: Docker удаляет выбранные данные и освобождает диск.\n  В зависимости от объема кэша операция может занять от 2 до 10 секунд...")

	box := ModalBoxStyle.
		BorderForeground(ColorSafe).
		Width(72).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			spinnerLine,
			"",
			subtext,
		))

	return lipgloss.PlaceHorizontal(screenWidth, lipgloss.Center, box)
}
