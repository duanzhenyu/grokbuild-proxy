//go:build ignore

// live-thinking-probe performs credentialed, quota-consuming checks against a
// running grokbuild-proxy. It is excluded from normal builds and must be run
// explicitly with: go run ./scripts/live-thinking-probe.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const maxProbeResponse = 64 << 20

type probeClient struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

type messageResponse struct {
	Content    []json.RawMessage `json:"content"`
	StopReason string            `json:"stop_reason"`
}

type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	Signature string          `json:"signature"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
}

type streamResult struct {
	thinkingStart  bool
	thinkingDelta  bool
	signatureDelta bool
	textDelta      bool
	messageStop    bool
	sequence       []string
}

func main() {
	if os.Getenv("GROKBUILD_LIVE_SMOKE") != "1" {
		fail("refusing live probe; set GROKBUILD_LIVE_SMOKE=1")
	}
	apiKey := strings.TrimSpace(os.Getenv("GROKBUILD_API_KEY"))
	if apiKey == "" {
		fail("GROKBUILD_API_KEY is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("GROKBUILD_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	model := strings.TrimSpace(os.Getenv("GROKBUILD_LIVE_MODEL"))
	if model == "" {
		model = "claude-opus-4-6"
	}
	probe := &probeClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 15 * time.Minute},
	}

	session := fmt.Sprintf("live-thinking-%d", time.Now().UnixNano())
	if err := probe.nonStreamToolRoundTrip(session); err != nil {
		fail("non-stream thinking tool round trip: %v", err)
	}
	if err := probe.streamThinking(session+"-summary", "summarized"); err != nil {
		fail("summarized thinking stream: %v", err)
	}
	if err := probe.streamThinking(session+"-omitted", "omitted"); err != nil {
		fail("omitted thinking stream: %v", err)
	}
	fmt.Println("live thinking probe passed: nonstream_roundtrip summarized_stream omitted_stream")
}

func (p *probeClient) nonStreamToolRoundTrip(session string) error {
	const prompt = "Use get_weather exactly once for Paris. Do not answer directly."
	tools := []any{map[string]any{
		"name":        "get_weather",
		"description": "Get the current weather for a city.",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{"type": "string"},
			},
			"required": []string{"location"},
		},
	}}
	firstRequest := map[string]any{
		"model":      p.model,
		"max_tokens": 1_024,
		"messages": []any{
			map[string]any{"role": "user", "content": prompt},
		},
		"tools":       tools,
		"tool_choice": map[string]any{"type": "any"},
		"thinking": map[string]any{
			"type":    "adaptive",
			"display": "summarized",
		},
		"output_config": map[string]any{"effort": "high"},
	}
	firstRaw, err := p.postJSON(session, firstRequest, false)
	if err != nil {
		return err
	}
	var first messageResponse
	if err := json.Unmarshal(firstRaw, &first); err != nil {
		return fmt.Errorf("decode first response: %w", err)
	}
	blocks, err := decodeBlocks(first.Content)
	if err != nil {
		return err
	}
	var thinking *contentBlock
	var toolUse *contentBlock
	for i := range blocks {
		switch blocks[i].Type {
		case "thinking":
			thinking = &blocks[i]
		case "tool_use":
			toolUse = &blocks[i]
		}
	}
	if thinking == nil || thinking.Thinking == "" || thinking.Signature == "" {
		return errors.New("first response did not contain summarized thinking plus signature")
	}
	if toolUse == nil || toolUse.ID == "" || toolUse.Name != "get_weather" {
		return errors.New("first response did not contain the expected tool_use")
	}
	if first.StopReason != "tool_use" {
		return fmt.Errorf("first stop_reason=%q, want tool_use", first.StopReason)
	}

	secondRequest := map[string]any{
		"model":      p.model,
		"max_tokens": 1_024,
		"messages": []any{
			map[string]any{"role": "user", "content": prompt},
			map[string]any{"role": "assistant", "content": first.Content},
			map[string]any{
				"role": "user",
				"content": []any{map[string]any{
					"type":        "tool_result",
					"tool_use_id": toolUse.ID,
					"content":     "Paris is sunny and 24 C.",
				}},
			},
		},
		"tools":       tools,
		"tool_choice": map[string]any{"type": "none"},
		"thinking": map[string]any{
			"type":    "adaptive",
			"display": "summarized",
		},
		"output_config": map[string]any{"effort": "high"},
	}
	secondRaw, err := p.postJSON(session, secondRequest, false)
	if err != nil {
		return err
	}
	var second messageResponse
	if err := json.Unmarshal(secondRaw, &second); err != nil {
		return fmt.Errorf("decode second response: %w", err)
	}
	secondBlocks, err := decodeBlocks(second.Content)
	if err != nil {
		return err
	}
	var hasText, hasThinkingSignature bool
	for _, block := range secondBlocks {
		if block.Type == "text" && block.Text != "" {
			hasText = true
		}
		if block.Type == "thinking" && block.Signature != "" {
			hasThinkingSignature = true
		}
	}
	if !hasText {
		return errors.New("second response did not contain final text")
	}
	if !hasThinkingSignature {
		return errors.New("second response did not contain a replayable thinking signature")
	}
	fmt.Println("live nonstream: summary=true signature=true tool_roundtrip=true final_text=true")
	return nil
}

func (p *probeClient) streamThinking(session, display string) error {
	request := map[string]any{
		"model":      p.model,
		"max_tokens": 512,
		"stream":     true,
		"messages": []any{map[string]any{
			"role":    "user",
			"content": "Determine whether 17 multiplied by 19 is greater than 300, then answer briefly.",
		}},
		"thinking": map[string]any{
			"type":    "adaptive",
			"display": display,
		},
		"output_config": map[string]any{"effort": "high"},
	}
	result, err := p.postStream(session, request)
	if err != nil {
		return err
	}
	if !result.thinkingStart || !result.signatureDelta || !result.textDelta || !result.messageStop {
		return fmt.Errorf(
			"incomplete stream events: thinking_start=%t signature=%t text=%t stop=%t",
			result.thinkingStart,
			result.signatureDelta,
			result.textDelta,
			result.messageStop,
		)
	}
	if display == "summarized" && !result.thinkingDelta {
		return errors.New("summarized stream did not include thinking_delta")
	}
	if display == "omitted" && result.thinkingDelta {
		return errors.New("omitted stream exposed thinking_delta")
	}
	if !ordered(result.sequence, "thinking_start", "signature_delta", "text_delta", "message_stop") {
		return fmt.Errorf("invalid event order: %s", strings.Join(result.sequence, ","))
	}
	fmt.Printf(
		"live stream %s: thinking_start=true thinking_delta=%t signature=true text=true stop=true\n",
		display,
		result.thinkingDelta,
	)
	return nil
}

func (p *probeClient) postJSON(session string, body any, stream bool) ([]byte, error) {
	response, err := p.do(session, body, stream)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxProbeResponse))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, statusError(response.StatusCode, raw)
	}
	return raw, nil
}

func (p *probeClient) postStream(session string, body any) (streamResult, error) {
	response, err := p.do(session, body, true)
	if err != nil {
		return streamResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return streamResult{}, statusError(response.StatusCode, raw)
	}
	var result streamResult
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 64*1024), 32*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		data := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		var event struct {
			Type         string `json:"type"`
			ContentBlock struct {
				Type string `json:"type"`
			} `json:"content_block"`
			Delta struct {
				Type string `json:"type"`
			} `json:"delta"`
		}
		if json.Unmarshal(data, &event) != nil {
			continue
		}
		switch {
		case event.Type == "content_block_start" && event.ContentBlock.Type == "thinking":
			result.thinkingStart = true
			result.sequence = append(result.sequence, "thinking_start")
		case event.Type == "content_block_delta" && event.Delta.Type == "thinking_delta":
			result.thinkingDelta = true
			result.sequence = append(result.sequence, "thinking_delta")
		case event.Type == "content_block_delta" && event.Delta.Type == "signature_delta":
			result.signatureDelta = true
			result.sequence = append(result.sequence, "signature_delta")
		case event.Type == "content_block_delta" && event.Delta.Type == "text_delta":
			result.textDelta = true
			if !contains(result.sequence, "text_delta") {
				result.sequence = append(result.sequence, "text_delta")
			}
		case event.Type == "message_stop":
			result.messageStop = true
			result.sequence = append(result.sequence, "message_stop")
		}
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("read stream: %w", err)
	}
	return result, nil
}

func (p *probeClient) do(session string, body any, stream bool) (*http.Response, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	request, err := http.NewRequest(
		http.MethodPost,
		p.baseURL+"/v1/messages",
		bytes.NewReader(raw),
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+p.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("anthropic-version", "2023-06-01")
	request.Header.Set("x-claude-code-session-id", session)
	if stream {
		request.Header.Set("Accept", "text/event-stream")
	}
	return p.client.Do(request)
}

func decodeBlocks(rawBlocks []json.RawMessage) ([]contentBlock, error) {
	blocks := make([]contentBlock, 0, len(rawBlocks))
	for _, raw := range rawBlocks {
		var block contentBlock
		if err := json.Unmarshal(raw, &block); err != nil {
			return nil, fmt.Errorf("decode content block: %w", err)
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func statusError(status int, raw []byte) error {
	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &envelope)
	if envelope.Error.Type != "" || envelope.Error.Message != "" {
		return fmt.Errorf(
			"status %d: %s: %s",
			status,
			envelope.Error.Type,
			envelope.Error.Message,
		)
	}
	return fmt.Errorf("status %d", status)
}

func ordered(sequence []string, values ...string) bool {
	next := 0
	for _, value := range sequence {
		if next < len(values) && value == values[next] {
			next++
		}
	}
	return next == len(values)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "live thinking probe failed: "+format+"\n", args...)
	os.Exit(1)
}
