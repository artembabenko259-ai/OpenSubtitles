package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"opensubtitles/pkg/engine"
)

type ViewState int

const (
	StateFilePicker ViewState = iota
	StateConnect
	StateModelSelect
	StateLangSelect
	StateProcessing
	StateComplete
)

type FileItem struct {
	Name  string
	Path  string
	IsDir bool
}

type Model struct {
	State        ViewState
	Config       engine.Config
	Processor    *engine.Processor
	CurrentDir   string
	Files        []FileItem
	SelectedIdx  int
	SelectedPath string
	OutputDir    string

	// Inputs & Modals
	TextInput textinput.Model
	Err       error
	StatusMsg string

	// Model choices
	WhisperModels   []string
	WhisperIdx      int
	TranslateModels []string
	TranslateIdx    int
	Languages       []string
	LangIdx         int

	// Progress
	ProgressPercent float64
	ProgressStep    string
	ResultPath      string

	Width  int
	Height int
}

type ProgressMsg struct {
	Percent float64
	Step    string
}

type ProcessDoneMsg struct {
	ResultPath string
	Err        error
}

func NewModel() Model {
	cfg, _ := engine.LoadConfig()

	dir, _ := os.Getwd()

	ti := textinput.New()
	ti.Placeholder = "API Key..."
	ti.Focus()
	ti.CharLimit = 200

	m := Model{
		State:        StateFilePicker,
		Config:       cfg,
		Processor:    engine.NewProcessor(),
		CurrentDir:   dir,
		TextInput:    ti,
		WhisperModels: []string{
			"openai/whisper-large-v3-turbo",
			"openai/whisper-large-v3",
			"openai/whisper-medium",
		},
		TranslateModels: []string{
			"google/gemini-2.5-flash",
			"anthropic/claude-3.5-sonnet",
			"openai/gpt-4o-mini",
			"meta-llama/llama-3.3-70b-instruct",
		},
		Languages: []string{
			"Ukrainian",
			"English",
			"Russian",
			"Spanish",
			"German",
			"French",
			"Original",
		},
	}

	m.loadFiles()
	return m
}

func (m *Model) loadFiles() {
	entries, err := os.ReadDir(m.CurrentDir)
	if err != nil {
		m.Err = err
		return
	}

	var items []FileItem

	// Parent dir entry
	parent := filepath.Dir(m.CurrentDir)
	if parent != m.CurrentDir {
		items = append(items, FileItem{Name: "..", Path: parent, IsDir: true})
	}

	for _, entry := range entries {
		path := filepath.Join(m.CurrentDir, entry.Name())
		if entry.IsDir() {
			items = append(items, FileItem{Name: entry.Name() + "/", Path: path, IsDir: true})
		} else {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".mp4" || ext == ".mkv" || ext == ".mov" || ext == ".avi" {
				items = append(items, FileItem{Name: entry.Name(), Path: path, IsDir: false})
			}
		}
	}

	m.Files = items
	m.SelectedIdx = 0
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case ProgressMsg:
		m.ProgressPercent = msg.Percent
		m.ProgressStep = msg.Step

	case ProcessDoneMsg:
		if msg.Err != nil {
			m.Err = msg.Err
			m.State = StateFilePicker
		} else {
			m.ResultPath = msg.ResultPath
			m.State = StateComplete
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.State != StateConnect {
				return m, tea.Quit
			}

		case "/connect":
			m.State = StateConnect
			m.TextInput.Focus()
			m.TextInput.SetValue(m.Config.OpenRouterAPIKey)
			return m, textinput.Blink

		case "/model":
			m.State = StateModelSelect
			return m, nil

		case "/lang":
			m.State = StateLangSelect
			return m, nil
		}

		switch m.State {
		case StateFilePicker:
			switch msg.String() {
			case "up", "k":
				if m.SelectedIdx > 0 {
					m.SelectedIdx--
				}
			case "down", "j":
				if m.SelectedIdx < len(m.Files)-1 {
					m.SelectedIdx++
				}
			case "enter":
				if len(m.Files) == 0 {
					break
				}
				item := m.Files[m.SelectedIdx]
				if item.IsDir {
					m.CurrentDir = item.Path
					m.loadFiles()
				} else {
					m.SelectedPath = item.Path
					m.OutputDir = filepath.Dir(item.Path)
					return m, m.startProcessing()
				}
			}

		case StateConnect:
			switch msg.String() {
			case "enter":
				key := strings.TrimSpace(m.TextInput.Value())
				m.Config.OpenRouterAPIKey = key
				_ = engine.SaveConfig(m.Config)
				m.StatusMsg = "API Key saved"
				m.State = StateFilePicker
				return m, nil
			case "esc":
				m.State = StateFilePicker
			}
			m.TextInput, cmd = m.TextInput.Update(msg)
			return m, cmd

		case StateModelSelect:
			switch msg.String() {
			case "up", "k":
				if m.WhisperIdx > 0 {
					m.WhisperIdx--
				}
			case "down", "j":
				if m.WhisperIdx < len(m.WhisperModels)-1 {
					m.WhisperIdx++
				}
			case "right", "l":
				if m.TranslateIdx < len(m.TranslateModels)-1 {
					m.TranslateIdx++
				}
			case "left", "h":
				if m.TranslateIdx > 0 {
					m.TranslateIdx--
				}
			case "enter", "esc":
				m.Config.WhisperModel = m.WhisperModels[m.WhisperIdx]
				m.Config.TranslationModel = m.TranslateModels[m.TranslateIdx]
				_ = engine.SaveConfig(m.Config)
				m.StatusMsg = fmt.Sprintf("Models: %s / %s", m.Config.WhisperModel, m.Config.TranslationModel)
				m.State = StateFilePicker
			}

		case StateLangSelect:
			switch msg.String() {
			case "up", "k":
				if m.LangIdx > 0 {
					m.LangIdx--
				}
			case "down", "j":
				if m.LangIdx < len(m.Languages)-1 {
					m.LangIdx++
				}
			case "enter", "esc":
				m.Config.TargetLanguage = m.Languages[m.LangIdx]
				_ = engine.SaveConfig(m.Config)
				m.StatusMsg = "Language: " + m.Config.TargetLanguage
				m.State = StateFilePicker
			}

		case StateComplete:
			switch msg.String() {
			case "enter", "esc":
				m.State = StateFilePicker
			}
		}
	}

	return m, nil
}

func (m Model) startProcessing() tea.Cmd {
	return func() tea.Msg {
		opts := engine.ProcessOptions{
			VideoPath:      m.SelectedPath,
			OutputDir:      m.OutputDir,
			TargetLanguage: m.Config.TargetLanguage,
			WhisperModel:   m.Config.WhisperModel,
			TranslateModel: m.Config.TranslationModel,
			OpenRouterKey:  m.Config.OpenRouterAPIKey,
		}

		resPath, err := m.Processor.Process(opts, func(percent float64, step string) {
			// Callback
		})

		return ProcessDoneMsg{ResultPath: resPath, Err: err}
	}
}

func (m Model) View() string {
	var s strings.Builder

	s.WriteString(TitleStyle.Render("OPENSUBTITLES") + "\n\n")

	if m.StatusMsg != "" {
		s.WriteString(SuccessStyle.Render("[OK] "+m.StatusMsg) + "\n\n")
	}
	if m.Err != nil {
		s.WriteString(ErrorStyle.Render("[ERR] "+m.Err.Error()) + "\n\n")
	}

	switch m.State {
	case StateFilePicker:
		s.WriteString(SubtitleStyle.Render("Files:") + "\n")
		s.WriteString(MutedStyle.Render("Dir: "+m.CurrentDir) + "\n\n")

		if len(m.Files) == 0 {
			s.WriteString(ItemStyle.Render("No video files found."))
		} else {
			for i, file := range m.Files {
				cursor := "  "
				style := ItemStyle
				if i == m.SelectedIdx {
					cursor = "> "
					style = SelectedItemStyle
				}
				s.WriteString(style.Render(cursor+file.Name) + "\n")
			}
		}

		s.WriteString(HelpStyle.Render("\n[↑/↓] Navigate  [Enter] Select  [/connect] API Key  [/model] Models  [/lang] Language  [q] Quit"))

	case StateConnect:
		s.WriteString(SubtitleStyle.Render("API Key:") + "\n\n")
		s.WriteString(m.TextInput.View() + "\n\n")
		s.WriteString(HelpStyle.Render("[Enter] Save  [Esc] Back"))

	case StateModelSelect:
		s.WriteString(SubtitleStyle.Render("Models:") + "\n\n")
		s.WriteString("STT Model:\n")
		for i, wm := range m.WhisperModels {
			prefix := "  "
			if i == m.WhisperIdx {
				prefix = "> "
			}
			s.WriteString(prefix + wm + "\n")
		}
		s.WriteString("\nTranslation Model:\n")
		for i, tm := range m.TranslateModels {
			prefix := "  "
			if i == m.TranslateIdx {
				prefix = "> "
			}
			s.WriteString(prefix + tm + "\n")
		}
		s.WriteString(HelpStyle.Render("\n[↑/↓] STT  [←/→] Translation  [Enter] Save"))

	case StateLangSelect:
		s.WriteString(SubtitleStyle.Render("Language:") + "\n\n")
		for i, lang := range m.Languages {
			prefix := "  "
			style := ItemStyle
			if i == m.LangIdx {
				prefix = "> "
				style = SelectedItemStyle
			}
			s.WriteString(style.Render(prefix+lang) + "\n")
		}
		s.WriteString(HelpStyle.Render("\n[↑/↓] Select  [Enter] Save"))

	case StateProcessing:
		s.WriteString(SubtitleStyle.Render("Processing...") + "\n\n")
		s.WriteString(fmt.Sprintf("File: %s\n", filepath.Base(m.SelectedPath)))
		s.WriteString(fmt.Sprintf("Lang: %s\n\n", m.Config.TargetLanguage))
		s.WriteString(fmt.Sprintf("Progress: %.0f%%\n", m.ProgressPercent))
		s.WriteString(m.ProgressStep + "\n")

	case StateComplete:
		s.WriteString(SuccessStyle.Render("Done!") + "\n\n")
		s.WriteString("Saved: " + m.ResultPath + "\n\n")
		s.WriteString(HelpStyle.Render("[Enter] Back"))
	}

	return HeaderBoxStyle.Render(s.String())
}
