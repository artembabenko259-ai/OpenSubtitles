package tui

import (
	"fmt"
	"io/fs"
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
	StateConfirmProcess
	StateProcessing
	StateComplete
)

type FileItem struct {
	Name string
	Path string
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

	// Default scan root to User Home directory (e.g. C:\Users\User)
	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		dir, _ = os.Getwd()
	}

	ti := textinput.New()
	ti.Placeholder = "sk-or-v1-..."
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

func isMediaFile(ext string) bool {
	ext = strings.ToLower(ext)
	mediaExts := map[string]bool{
		// Video
		".mp4": true, ".mkv": true, ".mov": true, ".avi": true,
		".webm": true, ".flv": true, ".wmv": true, ".m4v": true,
		".3gp": true, ".ts": true,
		// Audio
		".mp3": true, ".wav": true, ".m4a": true, ".aac": true,
		".flac": true, ".ogg": true, ".opus": true, ".wma": true,
	}
	return mediaExts[ext]
}

func normalizeKey(k string) string {
	k = strings.ToLower(k)
	switch k {
	case "k", "л":
		return "k"
	case "l", "д":
		return "l"
	case "m", "ь":
		return "m"
	case "q", "й":
		return "q"
	case "y", "н":
		return "y"
	case "n", "т":
		return "n"
	case "r", "к":
		return "r"
	}
	return k
}

func (m *Model) loadFiles() {
	var items []FileItem

	// Recursively scan entire User disk space (Desktop, Downloads, Videos, Documents, etc.)
	_ = filepath.WalkDir(m.CurrentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			// Skip system / build directories for high speed scanning
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "AppData" || name == "$RECYCLE.BIN" || name == "System Volume Information" || name == "Windows" || name == "Program Files" || name == "Program Files (x86)" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(d.Name())
		if isMediaFile(ext) {
			rel, relErr := filepath.Rel(m.CurrentDir, path)
			displayName := rel
			if relErr != nil {
				displayName = d.Name()
			}
			items = append(items, FileItem{Name: displayName, Path: path})
		}
		return nil
	})

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
		rawKey := msg.String()
		key := normalizeKey(rawKey)

		if m.State == StateConnect {
			switch rawKey {
			case "enter":
				apiKey := strings.TrimSpace(m.TextInput.Value())
				m.Config.OpenRouterAPIKey = apiKey
				_ = engine.SaveConfig(m.Config)
				m.StatusMsg = "API Key saved successfully!"
				m.State = StateFilePicker
				return m, nil
			case "esc":
				m.State = StateFilePicker
				return m, nil
			}
			m.TextInput, cmd = m.TextInput.Update(msg)
			return m, cmd
		}

		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "k":
			m.State = StateConnect
			m.TextInput.Focus()
			m.TextInput.SetValue(m.Config.OpenRouterAPIKey)
			return m, textinput.Blink

		case "m":
			m.State = StateModelSelect
			return m, nil

		case "l":
			m.State = StateLangSelect
			return m, nil

		case "r":
			m.loadFiles()
			m.StatusMsg = "Rescanned disk for media files"
			return m, nil
		}

		switch m.State {
		case StateFilePicker:
			switch rawKey {
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
				if m.Config.OpenRouterAPIKey == "" {
					m.Err = fmt.Errorf("API Key is not set! Press 'K' to enter API Key first")
					break
				}
				m.SelectedPath = item.Path
				m.OutputDir = filepath.Dir(item.Path)
				m.State = StateConfirmProcess
			}

		case StateConfirmProcess:
			switch key {
			case "y":
				m.State = StateProcessing
				return m, m.startProcessing()
			case "n":
				m.State = StateFilePicker
			}
			if rawKey == "enter" {
				m.State = StateProcessing
				return m, m.startProcessing()
			} else if rawKey == "esc" {
				m.State = StateFilePicker
			}

		case StateModelSelect:
			switch rawKey {
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
				m.StatusMsg = "Models saved"
				m.State = StateFilePicker
			}

		case StateLangSelect:
			switch rawKey {
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
				m.StatusMsg = "Target Language set to " + m.Config.TargetLanguage
				m.State = StateFilePicker
			}

		case StateComplete:
			switch rawKey {
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
			// Progress
		})

		return ProcessDoneMsg{ResultPath: resPath, Err: err}
	}
}

func (m Model) View() string {
	var s strings.Builder

	s.WriteString(TitleStyle.Render("OPENSUBTITLES CONTROL PANEL") + "\n")

	keyStatus := "[NOT SET - PRESS K]"
	if m.Config.OpenRouterAPIKey != "" {
		keyStatus = "[CONFIGURED]"
	}
	s.WriteString(fmt.Sprintf("API Key: %s  |  Language: %s  |  Model: %s\n",
		keyStatus, m.Config.TargetLanguage, filepath.Base(m.Config.WhisperModel)))
	s.WriteString(strings.Repeat("-", 65) + "\n\n")

	if m.StatusMsg != "" {
		s.WriteString(SuccessStyle.Render("[OK] "+m.StatusMsg) + "\n\n")
	}
	if m.Err != nil {
		s.WriteString(ErrorStyle.Render("[ERROR] "+m.Err.Error()) + "\n\n")
	}

	switch m.State {
	case StateFilePicker:
		s.WriteString(SubtitleStyle.Render(fmt.Sprintf("Discovered System Media Files (%d found):", len(m.Files))) + "\n")
		s.WriteString(MutedStyle.Render("Scanning User Disk: "+m.CurrentDir) + "\n\n")

		if len(m.Files) == 0 {
			s.WriteString(ItemStyle.Render("No media files (.mp4, .mkv, .mov, .mp3, .wav, etc.) found."))
		} else {
			maxVisible := 10
			startIdx := 0
			if m.SelectedIdx >= maxVisible {
				startIdx = m.SelectedIdx - maxVisible + 1
			}
			endIdx := startIdx + maxVisible
			if endIdx > len(m.Files) {
				endIdx = len(m.Files)
			}

			if startIdx > 0 {
				s.WriteString(MutedStyle.Render("  ▲ ...") + "\n")
			}

			for i := startIdx; i < endIdx; i++ {
				file := m.Files[i]
				cursor := "  "
				style := ItemStyle
				if i == m.SelectedIdx {
					cursor = "> "
					style = SelectedItemStyle
				}
				s.WriteString(style.Render(cursor+file.Name) + "\n")
			}

			if endIdx < len(m.Files) {
				s.WriteString(MutedStyle.Render("  ▼ ...") + "\n")
			}
		}

		s.WriteString(HelpStyle.Render("\n[Enter] Process    [K] API Key    [L] Language    [M] Models    [R] Rescan    [Q] Quit"))

	case StateConfirmProcess:
		s.WriteString(SubtitleStyle.Render("Confirm Subtitle Processing:") + "\n\n")
		s.WriteString(fmt.Sprintf("File:     %s\n", filepath.Base(m.SelectedPath)))
		s.WriteString(fmt.Sprintf("Language: %s\n", m.Config.TargetLanguage))
		s.WriteString(fmt.Sprintf("Whisper:  %s\n\n", m.Config.WhisperModel))
		s.WriteString(SuccessStyle.Render("Start processing now? [Y/n]") + "\n\n")
		s.WriteString(HelpStyle.Render("[Y / Enter] Start  [N / Esc] Cancel"))

	case StateConnect:
		s.WriteString(SubtitleStyle.Render("Enter OpenRouter API Key:") + "\n\n")
		s.WriteString(m.TextInput.View() + "\n\n")
		s.WriteString(HelpStyle.Render("[Enter] Save API Key  [Esc] Cancel"))

	case StateModelSelect:
		s.WriteString(SubtitleStyle.Render("Select Models:") + "\n\n")
		s.WriteString("STT Whisper Model:\n")
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
		s.WriteString(SubtitleStyle.Render("Select Target Language:") + "\n\n")
		for i, lang := range m.Languages {
			prefix := "  "
			style := ItemStyle
			if i == m.LangIdx {
				prefix = "> "
				style = SelectedItemStyle
			}
			s.WriteString(style.Render(prefix+lang) + "\n")
		}
		s.WriteString(HelpStyle.Render("\n[↑/↓] Select Language  [Enter] Save"))

	case StateProcessing:
		s.WriteString(SubtitleStyle.Render("Processing Subtitles...") + "\n\n")
		s.WriteString(fmt.Sprintf("File: %s\n", filepath.Base(m.SelectedPath)))
		s.WriteString(fmt.Sprintf("Language: %s\n\n", m.Config.TargetLanguage))
		s.WriteString(fmt.Sprintf("Progress: %.0f%%\n", m.ProgressPercent))
		s.WriteString(m.ProgressStep + "\n")

	case StateComplete:
		s.WriteString(SuccessStyle.Render("[SUCCESS] Subtitle Processing Complete!") + "\n\n")
		s.WriteString("Saved Output Video: " + m.ResultPath + "\n\n")
		s.WriteString(HelpStyle.Render("[Enter] Return to File Picker"))
	}

	return HeaderBoxStyle.Render(s.String())
}
