package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"opensubtitles/pkg/tui"
)

func main() {
	p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running OpenSubtitles TUI: %v\n", err)
		os.Exit(1)
	}
}
