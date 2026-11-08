package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines keyboard shortcuts for the TUI.
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Tab       key.Binding
	Tab1      key.Binding
	Tab2      key.Binding
	Select    key.Binding
	SelectAll key.Binding
	Stop      key.Binding
	Pause     key.Binding
	Restart   key.Binding
	Remove    key.Binding
	CleanDisk key.Binding
	Logs      key.Binding
	Inspect   key.Binding
	SortMem   key.Binding
	SortCPU   key.Binding
	SortName  key.Binding
	Confirm   key.Binding
	Cancel    key.Binding
	Refresh   key.Binding
	Quit      key.Binding
}

// DefaultKeyMap sets up default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "вверх"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "вниз"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "сменить вкладку"),
		),
		Tab1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "контейнеры"),
		),
		Tab2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "диск"),
		),
		Select: key.NewBinding(
			key.WithKeys(" ", "v"),
			key.WithHelp("space/v", "выбрать"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "выбрать все"),
		),
		Stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "остановить"),
		),
		Pause: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "пауза/старт"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "перезапуск"),
		),
		Remove: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "удалить"),
		),
		CleanDisk: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "быстрая очистка"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "логи"),
		),
		Inspect: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "детали"),
		),
		SortMem: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "сорт RAM"),
		),
		SortCPU: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "сорт CPU"),
		),
		SortName: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "сорт Имя"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y", "enter"),
			key.WithHelp("y/enter", "подтвердить"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc", "n"),
			key.WithHelp("esc/n", "отмена"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "обновить"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "выход"),
		),
	}
}

