package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SocialImageService struct {
	AI        *AIService
	AssetsDir string
	BaseURL   string
}

func NewSocialImageService(ai *AIService, assetsDir, baseURL string) *SocialImageService {
	return &SocialImageService{
		AI:        ai,
		AssetsDir: assetsDir,
		BaseURL:   strings.TrimRight(baseURL, "/"),
	}
}

type openAIImageReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Size   string `json:"size,omitempty"`
}

type openAIImageResp struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *SocialImageService) GenerateImage(ctx context.Context, prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("image prompt vacío")
	}

	if strings.TrimSpace(s.AI.APIKey) == "" {
		return "", fmt.Errorf("OPENAI_API_KEY no configurada")
	}

	// crear carpeta
	if err := os.MkdirAll(s.AssetsDir, 0o755); err != nil {
		return "", err
	}

	payload := openAIImageReq{
		Model:  "gpt-image-1",
		Prompt: prompt,
		Size:   "1024x1024",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// ⚡ contexto con timeout alto
	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.openai.com/v1/images/generations",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+s.AI.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// 🔥 CLIENTE NUEVO (ESTA ES LA CLAVE)
	client := &http.Client{
		Timeout: 180 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error conexión OpenAI: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var parsed openAIImageResp
	_ = json.Unmarshal(raw, &parsed)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return "", fmt.Errorf(parsed.Error.Message)
		}
		return "", fmt.Errorf("openai image error: %s", string(raw))
	}

	if len(parsed.Data) == 0 {
		return "", fmt.Errorf("openai no devolvió imagen")
	}

	// 🔥 CASO URL DIRECTA
	if strings.TrimSpace(parsed.Data[0].URL) != "" {
		return strings.TrimSpace(parsed.Data[0].URL), nil
	}

	// 🔥 CASO BASE64
	if strings.TrimSpace(parsed.Data[0].B64JSON) == "" {
		return "", fmt.Errorf("respuesta de imagen vacía")
	}

	bin, err := base64.StdEncoding.DecodeString(parsed.Data[0].B64JSON)
	if err != nil {
		return "", fmt.Errorf("decode base64 image: %w", err)
	}

	filename := uuid.NewString() + ".png"
	absPath := filepath.Join(s.AssetsDir, filename)

	if err := os.WriteFile(absPath, bin, 0o644); err != nil {
		return "", err
	}

	relURL := "/social-assets/" + filename
	if s.BaseURL != "" {
		return s.BaseURL + relURL, nil
	}

	return relURL, nil
}