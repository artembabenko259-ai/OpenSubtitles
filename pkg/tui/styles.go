package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7D56F4") // Purple
	secondaryColor = lipgloss.Color("#04B5D5") // Cyan
	accentColor    = lipgloss.Color("#FFD700") // Gold / Yellow
	successColor   = lipgloss.Color("#00FF7F") // Spring Green
	errorColor     = lipgloss.Color("#FF4500") // Orange Red
	mutedColor     = lipgloss.Color("#626262") // Dark Gray

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	HeaderBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			MarginBottom(1)

	ItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	SelectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(accentColor).
				Bold(true)

	StatusBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(secondaryColor).
				Padding(0, 1).
				Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)
)
