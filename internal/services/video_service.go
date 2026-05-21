package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type VideoService struct {
	DB       *sql.DB
	APIKey   string
	Model    string
	BaseURL  string
}

func NewVideoService(db *sql.DB, baseURL string) *VideoService {
	model := os.Getenv("REPLICATE_VIDEO_MODEL")
	if model == "" {
		model = "bytedance/seedance-1-lite"
	}

	return &VideoService{
		DB:      db,
		APIKey:  os.Getenv("REPLICATE_API_TOKEN"),
		Model:   model,
		BaseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (v *VideoService) CreateJob(ctx context.Context, clientID, prompt, imageURL string, duration int) (models.AIVideoJob, error) {
	if strings.TrimSpace(v.APIKey) == "" {
		return models.AIVideoJob{}, fmt.Errorf("REPLICATE_API_TOKEN no configurado")
	}

	if duration <= 0 {
		duration = 5
	}

	now := time.Now()
	job := models.AIVideoJob{
		ID:        uuid.NewString(),
		ClientID:  clientID,
		Prompt:    strings.TrimSpace(prompt),
		ImageURL:  strings.TrimSpace(imageURL),
		Provider:  "replicate",
		Model:     v.Model,
		Status:    "processing",
		CostCredits: duration,
		CreatedAt: now,
		UpdatedAt: now,
	}

	input := map[string]any{
		"prompt": job.Prompt,
		"duration": duration,
		"aspect_ratio": "9:16",
	}

	if job.ImageURL != "" {
		input["image"] = job.ImageURL
	}

	payload := map[string]any{
		"input": input,
	}

	body, _ := json.Marshal(payload)

	apiURL := fmt.Sprintf("https://api.replicate.com/v1/models/%s/predictions", v.Model)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return models.AIVideoJob{}, err
	}

	req.Header.Set("Authorization", "Bearer "+v.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "wait=5")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.AIVideoJob{}, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var parsed struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Output any `json:"output"`
		Error any `json:"error"`
		Urls struct {
			Get string `json:"get"`
		} `json:"urls"`
	}

	_ = json.Unmarshal(raw, &parsed)

	if resp.StatusCode >= 300 {
		return models.AIVideoJob{}, fmt.Errorf("replicate error: %s", string(raw))
	}

	job.ProviderJobID = parsed.ID
	job.ProviderGetURL = parsed.Urls.Get

	_, err = v.DB.Exec(`
		INSERT INTO ai_video_jobs (
			id, client_id, prompt, image_url, video_url, provider, model,
			provider_job_id, provider_get_url, status, error,
			cost_credits, created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		job.ID,
		job.ClientID,
		job.Prompt,
		job.ImageURL,
		job.VideoURL,
		job.Provider,
		job.Model,
		job.ProviderJobID,
		job.ProviderGetURL,
		job.Status,
		job.Error,
		job.CostCredits,
		job.CreatedAt,
		job.UpdatedAt,
		job.CompletedAt,
	)

	if err != nil {
		return models.AIVideoJob{}, err
	}

	return job, nil
}

func (v *VideoService) RefreshJob(ctx context.Context, jobID string) (models.AIVideoJob, error) {
	job, err := v.GetJob(jobID)
	if err != nil {
		return models.AIVideoJob{}, err
	}

	if job.ProviderGetURL == "" || job.Status == "completed" || job.Status == "error" {
		return job, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.ProviderGetURL, nil)
	if err != nil {
		return job, err
	}

	req.Header.Set("Authorization", "Bearer "+v.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return job, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var parsed struct {
		Status string `json:"status"`
		Output any `json:"output"`
		Error any `json:"error"`
	}

	_ = json.Unmarshal(raw, &parsed)

	now := time.Now()

	if parsed.Status == "succeeded" {
		job.Status = "completed"
		job.VideoURL = extractReplicateOutputURL(parsed.Output)
		job.CompletedAt = &now
	} else if parsed.Status == "failed" || parsed.Status == "canceled" {
		job.Status = "error"
		job.Error = fmt.Sprint(parsed.Error)
	} else {
		job.Status = "processing"
	}

	job.UpdatedAt = now

	_, _ = v.DB.Exec(`
		UPDATE ai_video_jobs
		SET video_url=?, status=?, error=?, updated_at=?, completed_at=?
		WHERE id=?
	`, job.VideoURL, job.Status, job.Error, job.UpdatedAt, job.CompletedAt, job.ID)

	return job, nil
}

func extractReplicateOutputURL(output any) string {
	switch v := output.(type) {
	case string:
		return v
	case []any:
		if len(v) > 0 {
			return fmt.Sprint(v[0])
		}
	case map[string]any:
		for _, key := range []string{"video", "url", "output"} {
			if val, ok := v[key]; ok {
				return fmt.Sprint(val)
			}
		}
	}
	return ""
}

func (v *VideoService) GetJob(id string) (models.AIVideoJob, error) {
	var job models.AIVideoJob

	err := v.DB.QueryRow(`
		SELECT id, client_id, prompt, image_url, video_url, provider, model,
		       provider_job_id, provider_get_url, status, error,
		       cost_credits, created_at, updated_at, completed_at
		FROM ai_video_jobs
		WHERE id=?
	`, id).Scan(
		&job.ID,
		&job.ClientID,
		&job.Prompt,
		&job.ImageURL,
		&job.VideoURL,
		&job.Provider,
		&job.Model,
		&job.ProviderJobID,
		&job.ProviderGetURL,
		&job.Status,
		&job.Error,
		&job.CostCredits,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.CompletedAt,
	)

	return job, err
}

func (v *VideoService) ListJobs(clientID string) ([]models.AIVideoJob, error) {
	rows, err := v.DB.Query(`
		SELECT id, client_id, prompt, image_url, video_url, provider, model,
		       provider_job_id, provider_get_url, status, error,
		       cost_credits, created_at, updated_at, completed_at
		FROM ai_video_jobs
		WHERE client_id=?
		ORDER BY created_at DESC
		LIMIT 50
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.AIVideoJob{}

	for rows.Next() {
		var job models.AIVideoJob
		if err := rows.Scan(
			&job.ID,
			&job.ClientID,
			&job.Prompt,
			&job.ImageURL,
			&job.VideoURL,
			&job.Provider,
			&job.Model,
			&job.ProviderJobID,
			&job.ProviderGetURL,
			&job.Status,
			&job.Error,
			&job.CostCredits,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.CompletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, job)
	}

	return out, nil
}