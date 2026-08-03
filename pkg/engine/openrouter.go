package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type WordTiming struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type Segment struct {
	ID    int          `json:"id"`
	Start float64      `json:"start"`
	End   float64      `json:"end"`
	Text  string       `json:"text"`
	Words []WordTiming `json:"words"`
}

type TranscriptionResult struct {
	Text     string    `json:"text"`
	Segments []Segment `json:"segments"`
}

type OpenRouterClient struct {
	APIKey string
	HTTP   *http.Client
}

func NewOpenRouterClient(apiKey string) *OpenRouterClient {
	return &OpenRouterClient{
		APIKey: apiKey,
		HTTP: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *OpenRouterClient) TranscribeAudio(audioPath, model string) (*TranscriptionResult, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is not set. Use /connect in TUI to set it")
	}

	file, err := os.Open(audioPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	_ = writer.WriteField("model", model)
	_ = writer.WriteField("response_format", "verbose_json")

	if err := writer.Close(); err != nil {
		return nil, err
	}

	url := "https://openrouter.ai/api/v1/audio/transcriptions"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/artembabenko259-ai/OpenSubtitles")
	req.Header.Set("X-Title", "OpenSubtitles")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transcription API returned error status %d: %s", resp.StatusCode, string(respBody))
	}

	var result TranscriptionResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse transcription response: %v, body: %s", err, string(respBody))
	}

	return &result, nil
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *OpenRouterClient) TranslateSegments(result *TranscriptionResult, targetLang, model string) (*TranscriptionResult, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is not set")
	}

	if targetLang == "" || targetLang == "Original" {
		return result, nil
	}

	inputJSON, err := json.Marshal(result.Segments)
	if err != nil {
		return nil, err
	}

	prompt := "You are a professional subtitle translator. Translate the following JSON array of subtitle segments into " + targetLang + ".\n" +
		"CRITICAL RULES:\n" +
		"1. Maintain the EXACT SAME JSON array structure with 'id', 'start', 'end', 'text', and 'words' fields.\n" +
		"2. Translate the 'text' field into natural " + targetLang + ".\n" +
		"3. In the 'words' array, translate the 'word' fields to " + targetLang + ", but KEEP 'start' and 'end' numbers EXACTLY as they are.\n" +
		"4. Return ONLY valid raw JSON array. Do not include markdown code block formatting.\n\n" +
		"Input JSON:\n" + string(inputJSON)

	reqPayload := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
	}

	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	url := "https://openrouter.ai/api/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("translation API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, err
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("OpenRouter error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("empty translation response")
	}

	translatedText := chatResp.Choices[0].Message.Content

	// Clean code block ticks if LLM returned them
	if bytes.HasPrefix([]byte(translatedText), []byte("```")) {
		lines := bytes.Split([]byte(translatedText), []byte("\n"))
		if len(lines) > 2 {
			translatedText = string(bytes.Join(lines[1:len(lines)-1], []byte("\n")))
		}
	}

	var translatedSegments []Segment
	if err := json.Unmarshal([]byte(translatedText), &translatedSegments); err != nil {
		// Fallback to original if translation JSON parse failed
		return result, nil
	}

	translatedResult := &TranscriptionResult{
		Text:     result.Text,
		Segments: translatedSegments,
	}

	return translatedResult, nil
}
