package engine

import (
	"fmt"
	"math"
	"os"
	"strings"
)

type SubtitleGenerator struct{}

func NewSubtitleGenerator() *SubtitleGenerator {
	return &SubtitleGenerator{}
}

func formatASSTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	centisecs := int(math.Round((seconds - math.Floor(seconds)) * 100))

	if centisecs >= 100 {
		centisecs = 99
	}

	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, secs, centisecs)
}

func (s *SubtitleGenerator) GenerateASS(result *TranscriptionResult, outputPath string, styleName string) error {
	var builder strings.Builder

	// Write ASS Header
	builder.WriteString("[Script Info]\n")
	builder.WriteString("Title: OpenSubtitles Dynamic Subtitles\n")
	builder.WriteString("ScriptType: v4.00+\n")
	builder.WriteString("WrapStyle: 0\n")
	builder.WriteString("ScaledBorderAndShadow: yes\n\n")

	builder.WriteString("[V4+ Styles]\n")
	builder.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")

	// CapCut / TikTok Style: Yellow/White text, thick black outline, bottom-center alignment
	builder.WriteString("Style: Default,Arial,20,&H00FFFFFF,&H0000FFFF,&H00000000,&H80000000,1,0,0,0,100,100,0,0,1,3,2,2,10,10,30,1\n\n")

	builder.WriteString("[Events]\n")
	builder.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	for _, seg := range result.Segments {
		if seg.Text == "" {
			continue
		}

		startStr := formatASSTime(seg.Start)
		endStr := formatASSTime(seg.End)

		var textBuilder strings.Builder
		if len(seg.Words) > 0 {
			// Generate Word-level Karaoke highlighting
			for _, w := range seg.Words {
				durationCenti := int(math.Round((w.End - w.Start) * 100))
				if durationCenti <= 0 {
					durationCenti = 10
				}
				textBuilder.WriteString(fmt.Sprintf("{\\k%d}%s ", durationCenti, strings.TrimSpace(w.Word)))
			}
		} else {
			textBuilder.WriteString(strings.TrimSpace(seg.Text))
		}

		line := fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", startStr, endStr, strings.TrimSpace(textBuilder.String()))
		builder.WriteString(line)
	}

	return os.WriteFile(outputPath, []byte(builder.String()), 0644)
}
