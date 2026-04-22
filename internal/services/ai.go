package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type AIService struct {
	APIKey       string
	DefaultModel string
	HTTP         *http.Client
}

func NewAIService(apiKey, model string) *AIService {
	return &AIService{
		APIKey:       apiKey,
		DefaultModel: model,
		HTTP:         &http.Client{Timeout: 10 * time.Second}, // rápido para chat/WhatsApp
	}
}

type oaReq struct {
	Model       string              `json:"model"`
	Messages    []map[string]string `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type oaResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (a *AIService) resolveModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		model = strings.TrimSpace(a.DefaultModel)
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	return model
}

func (a *AIService) doChatCompletion(
	ctx context.Context,
	model string,
	temperature float64,
	maxTokens int,
	messages []map[string]string,
) (string, error) {
	if strings.TrimSpace(a.APIKey) == "" {
		return "", fmt.Errorf("OPENAI_API_KEY no configurada")
	}

	if temperature <= 0 {
		temperature = 0.7
	}

	payload := oaReq{
		Model:       a.resolveModel(model),
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.openai.com/v1/chat/completions",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	var parsed oaResp
	if err := json.Unmarshal(b, &parsed); err != nil {
		return "", fmt.Errorf("openai parse: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil {
			return "", fmt.Errorf("%s", parsed.Error.Message)
		}
		return "", fmt.Errorf("%s", string(b))
	}

	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty choices")
	}

	answer := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if answer == "" {
		return "", fmt.Errorf("empty content")
	}

	return answer, nil
}

func cleanCodeBlock(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "```html", "")
	s = strings.ReplaceAll(s, "```HTML", "")
	s = strings.ReplaceAll(s, "```", "")
	return strings.TrimSpace(s)
}

func (a *AIService) GenerateReply(
	ctx context.Context,
	lead models.Lead,
	incoming string,
	cfg models.BotConfig,
	promptSnippets ...string,
) (string, error) {
	if strings.TrimSpace(a.APIKey) == "" {
		return strings.TrimSpace(cfg.FallbackMessage), nil
	}

	model := a.resolveModel(cfg.Model)

	temperature := cfg.Temperature
	if temperature <= 0 {
		temperature = 0.7
	}

	extra := strings.TrimSpace(strings.Join(promptSnippets, "\n\n"))

	system := fmt.Sprintf(`Eres el asistente comercial de %s.
Negocio: %s
Oferta: %s
Público objetivo: %s
Tono: %s
CTA: %s %s
Etapa del lead: %s
Última intención: %s
Resumen del lead: %s

Reglas:
- responde en español
- usa máximo 5 líneas
- no prometas ganancias ni resultados garantizados
- busca mover la conversación al siguiente paso
- termina con una pregunta corta cuando convenga
- si te dan una plantilla o mensaje base, úsalo como guía principal
- conserva un tono natural, humano y vendedor

Instrucciones específicas:
%s

Prompt personalizado:
%s`,
		cfg.BusinessName,
		cfg.BusinessDescription,
		cfg.Offer,
		cfg.TargetAudience,
		cfg.Tone,
		cfg.CTAButtonText,
		cfg.CTALink,
		lead.Stage,
		lead.LastIntent,
		lead.Summary,
		extra,
		cfg.SystemPrompt,
	)

	answer, err := a.doChatCompletion(
		ctx,
		model,
		temperature,
		220,
		[]map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": incoming},
		},
	)
	if err != nil {
		return "", err
	}

	answer = strings.TrimSpace(answer)
	if answer == "" {
		return strings.TrimSpace(cfg.FallbackMessage), nil
	}

	return answer, nil
}

func (a *AIService) GenerateHTML(
	ctx context.Context,
	systemPrompt string,
	model string,
) (string, error) {
	if strings.TrimSpace(a.APIKey) == "" {
		return "", fmt.Errorf("OPENAI_API_KEY no configurada")
	}

	model = a.resolveModel(model)

	// cliente separado solo para generación pesada de landing pages
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	payload := oaReq{
		Model: model,
		Messages: []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": "Devuelve únicamente HTML completo, válido y renderizable. No uses markdown ni explicaciones."},
		},
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.openai.com/v1/chat/completions",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	var parsed oaResp
	if err := json.Unmarshal(b, &parsed); err != nil {
		return "", fmt.Errorf("openai parse: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil {
			return "", fmt.Errorf("%s", parsed.Error.Message)
		}
		return "", fmt.Errorf("%s", string(b))
	}

	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty choices")
	}

	answer := strings.TrimSpace(parsed.Choices[0].Message.Content)
	answer = cleanCodeBlock(answer)

	if answer == "" {
		return "", fmt.Errorf("html vacío")
	}

	lower := strings.ToLower(answer)
	if !strings.Contains(lower, "<html") && !strings.Contains(lower, "<!doctype html") {
		return "", fmt.Errorf("la IA no devolvió HTML válido")
	}

	return answer, nil
}