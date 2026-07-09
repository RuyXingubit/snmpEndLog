// Package ai provides the Gemini AI client for log analysis.
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-flash-latest:generateContent"
	maxTokens      = 8192
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SystemPrompt is the fixed prompt for network log analysis.
const SystemPrompt = `Você é um analista de redes e infraestrutura de telecomunicações especializado.
Analise os logs de rede fornecidos e forneça:

1. **Problemas Críticos**: Identifique erros, falhas ou situações que precisam de atenção imediata.
2. **Padrões Suspeitos**: Destaque atividades incomuns, tentativas de acesso não autorizado, ou comportamentos anômalos.
3. **Recomendações**: Sugira ações para resolver os problemas e melhorar a segurança/estabilidade.

Seja conciso e objetivo. Use português brasileiro. Foque em informações acionáveis.
Os logs vêm de equipamentos MikroTik e Huawei em uma rede ISP.`

// geminiRequest is the Gemini API request body.
type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  *genConfig      `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type genConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

// geminiResponse is the Gemini API response body.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// Analyze sends messages to Gemini and returns the response text.
func Analyze(messages []Message) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not configured")
	}

	// Build contents, merging consecutive same-role messages
	// (Gemini requires strict user/model alternation)
	var contents []geminiContent
	for _, m := range messages {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}

		// Merge with previous if same role
		if len(contents) > 0 && contents[len(contents)-1].Role == role {
			prev := &contents[len(contents)-1]
			prev.Parts[0].Text += "\n\n" + m.Content
		} else {
			contents = append(contents, geminiContent{
				Role:  role,
				Parts: []geminiPart{{Text: m.Content}},
			})
		}
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: SystemPrompt}},
		},
		Contents: contents,
		GenerationConfig: &genConfig{
			MaxOutputTokens: maxTokens,
			Temperature:     0.3,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", geminiEndpoint, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if geminiResp.Error != nil {
		return "", fmt.Errorf("gemini error (%d): %s", geminiResp.Error.Code, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty gemini response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}
