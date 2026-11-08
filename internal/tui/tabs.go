package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// ActiveTab identifies which tab is currently visible.
type ActiveTab int

const (
	TabContainers ActiveTab = 0
	TabDisk       ActiveTab = 1
)

// RenderTabs renders the tab navigation bar.
func RenderTabs(current ActiveTab, containersCount int, safeReclaimableBytes int64) string {
	var tab1, tab2 string

	t1Text := fmt.Sprintf(" 1. КОНТЕЙНЕРЫ (%d) ", containersCount)
	t2Text := " 2. ДИСК И ОЧИСТКА "

	if current == TabContainers {
		tab1 = TabActiveStyle.Render(t1Text)
		tab2 = TabInactiveStyle.Render(t2Text)
	} else {
		tab1 = TabInactiveStyle.Render(t1Text)
		tab2 = TabActiveStyle.Render(t2Text)
	}

	title := TitleStyle.Render("⚡ DOCKERTRIM")
	return lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", tab1, " ", tab2)
}

