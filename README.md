# OpenSubtitles

**OpenSubtitles** is an AI-powered automated video subtitle generator & translator built with **Go** and **Bubble Tea** (Charm TUI).

It automatically transcribes audio using OpenRouter's **Whisper Large-v3-Turbo**, translates subtitles into any target language via OpenRouter LLMs (Gemini, Claude, GPT), generates stylized **CapCut/TikTok-style karaoke subtitles** (`.ass`), and hardcodes them onto the final video file using **FFmpeg**.

---

## Architecture

The project is cleanly split into two modular parts:

1. **`pkg/engine` (Headless Core Engine & Library)**:
   - Standalone Go library decoupled from TUI/GUI.
   - Handles FFmpeg audio extraction and subtitle burn-in.
   - Manages OpenRouter API requests for Whisper speech-to-text and LLM translation.
   - Generates Advanced SubStation Alpha (`.ass`) karaoke subtitles.
   - Can be imported into other projects or compiled as a native shared library (`.dll`/`.so`).

2. **`cmd/opensubtitles` & `pkg/tui` (Bubble Tea Interactive TUI)**:
   - Built with `charmbracelet/bubbletea` and `charmbracelet/lipgloss`.
   - File explorer for finding `.mp4`, `.mkv`, `.mov` files.
   - Interactive commands:
     - `/connect` — Prompt for OpenRouter API Key (`sk-or-v1-...`).
     - `/model` — Select Whisper model and LLM Translation model.
     - `/lang` — Select target translation language.

---

## Features

- **Bubble Tea TUI**: Beautiful, responsive terminal UI.
- **OpenRouter Integration**: Access to Whisper Large-v3-Turbo and state-of-the-art LLMs.
- **Word-Level Karaoke Highlighting**: Generates `.ass` subtitles with active word highlighting.
- **FFmpeg Integration**: Automatic audio extraction and hardcoded video rendering.

---

## Getting Started

### Prerequisites
- [Go 1.22+](https://go.dev/)
- [FFmpeg](https://ffmpeg.org/) in system `PATH`.

### Build & Run
```bash
go build -o opensubtitles.exe ./cmd/opensubtitles
./opensubtitles.exe
```

---

## License

MIT License.
