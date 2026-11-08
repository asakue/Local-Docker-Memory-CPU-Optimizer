package tui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary   = lipgloss.Color("#00D7D7")
	ColorSecondary = lipgloss.Color("#7D56F4")
	ColorSafe      = lipgloss.Color("#00FF87")
	ColorWarning   = lipgloss.Color("#FFAF00")
	ColorDanger    = lipgloss.Color("#FF4136")
	ColorMuted     = lipgloss.Color("#626262")
	ColorBackground= lipgloss.Color("#1A1B26")

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Padding(0, 1)

	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorSecondary).
			Padding(0, 2)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A0A0A0")).
				Background(lipgloss.Color("#2E2E3E")).
				Padding(0, 2)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#353545")).
			Padding(0, 1)

	RowCursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3A3F58"))

	RowSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorSafe).
				Background(lipgloss.Color("#252836"))

	BadgeIdle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(ColorWarning).
			Padding(0, 1)

	BadgeOOM = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorDanger).
			Padding(0, 1)

	BadgeNoLimit = lipgloss.NewStyle().
			Foreground(ColorMuted)

	ModalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2).
			Background(lipgloss.Color("#16161E"))

	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	KeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	SuccessToast = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorSafe).
			Padding(0, 2)

	ErrorToast = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorDanger).
			Padding(0, 2)
)

