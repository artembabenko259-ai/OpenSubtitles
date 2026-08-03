package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ProgressCallback func(percent float64, step string)

type ProcessOptions struct {
	VideoPath      string
	OutputDir      string
	TargetLanguage string
	WhisperModel   string
	TranslateModel string
	OpenRouterKey  string
}

type Processor struct {
	ffmpeg *FFmpeg
	subGen *SubtitleGenerator
}

func NewProcessor() *Processor {
	return &Processor{
		ffmpeg: NewFFmpeg(),
		subGen: NewSubtitleGenerator(),
	}
}

// Detect mounted vdisk RAM/VRAM drive (e.g. R:\ or Z:\) for ultra-fast 30GB/s I/O
func getRAMDiskTempDir() string {
	drives := []string{"R:", "Z:", "Y:", "X:", "V:"}
	for _, drive := range drives {
		if fi, err := os.Stat(drive + "\\"); err == nil && fi.IsDir() {
			ramTemp := drive + "\\opensubtitles_ramtemp"
			if err := os.MkdirAll(ramTemp, 0755); err == nil {
				return ramTemp
			}
		}
	}
	return ""
}

func (p *Processor) Process(opts ProcessOptions, callback ProgressCallback) (string, error) {
	if !p.ffmpeg.IsInstalled() {
		return "", fmt.Errorf("FFmpeg is not installed or not in PATH")
	}

	if opts.OpenRouterKey == "" {
		return "", fmt.Errorf("OpenRouter API key is missing. Set it using 'K' key")
	}

	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Dir(opts.VideoPath)
	}

	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return "", err
	}

	baseName := strings.TrimSuffix(filepath.Base(opts.VideoPath), filepath.Ext(opts.VideoPath))

	// Use vdisk RAM disk if available, otherwise use output dir
	workDir := getRAMDiskTempDir()
	usingRAMDisk := false
	if workDir != "" {
		usingRAMDisk = true
	} else {
		workDir = opts.OutputDir
	}

	tempAudio := filepath.Join(workDir, baseName+"_temp.mp3")
	tempASS := filepath.Join(workDir, baseName+"_sub.ass")
	outVideo := filepath.Join(opts.OutputDir, baseName+"_subtitled.mp4")

	defer func() {
		_ = os.Remove(tempAudio)
		_ = os.Remove(tempASS)
		if usingRAMDisk {
			_ = os.RemoveAll(workDir)
		}
	}()

	// Step 1: Extract Audio
	stepMsg := "Extracting audio with FFmpeg..."
	if usingRAMDisk {
		stepMsg = "Extracting audio to vdisk RAM Disk (30GB/s Speed)..."
	}
	if callback != nil {
		callback(10.0, stepMsg)
	}
	if err := p.ffmpeg.ExtractAudio(opts.VideoPath, tempAudio); err != nil {
		return "", err
	}

	// Step 2: Speech-to-Text via OpenRouter Whisper
	if callback != nil {
		callback(35.0, fmt.Sprintf("Transcribing with %s...", opts.WhisperModel))
	}
	client := NewOpenRouterClient(opts.OpenRouterKey)
	transcription, err := client.TranscribeAudio(tempAudio, opts.WhisperModel)
	if err != nil {
		return "", err
	}

	// Step 3: Translation via OpenRouter LLM
	if callback != nil {
		callback(65.0, fmt.Sprintf("Translating to %s with %s...", opts.TargetLanguage, opts.TranslateModel))
	}
	finalTranscription := transcription
	if opts.TargetLanguage != "" && opts.TargetLanguage != "Original" {
		translated, err := client.TranslateSegments(transcription, opts.TargetLanguage, opts.TranslateModel)
		if err == nil {
			finalTranscription = translated
		}
	}

	// Step 4: Generate ASS Subtitles
	if callback != nil {
		callback(85.0, "Generating ASS karaoke subtitles...")
	}
	if err := p.subGen.GenerateASS(finalTranscription, tempASS, "CapCut"); err != nil {
		return "", err
	}

	// Step 5: Burn Subtitles into Video
	if callback != nil {
		callback(95.0, "Burning subtitles into final video...")
	}
	if err := p.ffmpeg.BurnSubtitles(opts.VideoPath, tempASS, outVideo); err != nil {
		return "", err
	}

	if callback != nil {
		callback(100.0, fmt.Sprintf("DONE! Saved to %s", outVideo))
	}

	return outVideo, nil
}
