package engine

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type FFmpeg struct{}

func NewFFmpeg() *FFmpeg {
	return &FFmpeg{}
}

func (f *FFmpeg) IsInstalled() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func (f *FFmpeg) ExtractAudio(videoPath, audioPath string) error {
	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", videoPath,
		"-vn",
		"-acodec", "libmp3lame",
		"-ar", "16000",
		"-ac", "1",
		"-q:a", "2",
		audioPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg extract audio failed: %v, output: %s", err, string(output))
	}
	return nil
}

func (f *FFmpeg) BurnSubtitles(videoPath, assPath, outputPath string) error {
	escapedAssPath := assPath
	if runtime.GOOS == "windows" {
		escapedAssPath = strings.ReplaceAll(assPath, "\\", "/")
		escapedAssPath = strings.ReplaceAll(escapedAssPath, ":", "\\:")
	}

	subFilter := fmt.Sprintf("subtitles='%s'", escapedAssPath)

	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", videoPath,
		"-vf", subFilter,
		"-c:a", "copy",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg burn subtitles failed: %v, output: %s", err, string(output))
	}
	return nil
}
