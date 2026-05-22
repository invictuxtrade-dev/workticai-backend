package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type VideoService struct {
	DB      *sql.DB
	APIKey string
	Model  string
	BaseURL string
}

type VoiceSubtitleOptions struct {
	Text            string `json:"text"`
	Language        string `json:"language"` // es | en
	Gender          string `json:"gender"`   // female | male
	EnableVoice     bool   `json:"enable_voice"`
	EnableSubtitles bool   `json:"enable_subtitles"`
}

func NewVideoService(db *sql.DB, baseURL string) *VideoService {
	model := os.Getenv("REPLICATE_VIDEO_MODEL")
	if model == "" {
		model = "bytedance/seedance-1-lite"
	}

	return &VideoService{
		DB:      db,
		APIKey: os.Getenv("REPLICATE_API_TOKEN"),
		Model:  model,
		BaseURL: strings.TrimRight(baseURL, "/"),
	}
}

func musicLibraryDir() string {
	dir := os.Getenv("MUSIC_LIBRARY_DIR")
	if strings.TrimSpace(dir) == "" {
		dir = filepath.Join("data", "music")
	}
	return dir
}

func videoAssetsDir() string {
	dir := os.Getenv("VIDEO_ASSETS_DIR")
	if strings.TrimSpace(dir) == "" {
		dir = filepath.Join("data", "social_assets")
	}
	return dir
}

func (v *VideoService) publicAssetURL(filename string) string {
	rel := "/social-assets/" + filename
	if strings.TrimSpace(v.BaseURL) != "" {
		return strings.TrimRight(v.BaseURL, "/") + rel
	}
	return rel
}

func detectMusicCategory(text string) string {
	t := strings.ToLower(text)

	switch {
	case strings.Contains(t, "trading"), strings.Contains(t, "forex"), strings.Contains(t, "crypto"), strings.Contains(t, "copytrading"), strings.Contains(t, "invers"):
		return "trading"
	case strings.Contains(t, "viral"), strings.Contains(t, "tiktok"), strings.Contains(t, "reel"), strings.Contains(t, "short"):
		return "viral"
	case strings.Contains(t, "cinematic"), strings.Contains(t, "cinematográfico"), strings.Contains(t, "película"):
		return "cinematic"
	case strings.Contains(t, "luxury"), strings.Contains(t, "lujo"), strings.Contains(t, "premium"):
		return "luxury"
	case strings.Contains(t, "dark"), strings.Contains(t, "oscuro"), strings.Contains(t, "hacker"):
		return "dark"
	case strings.Contains(t, "tech"), strings.Contains(t, "tecnología"), strings.Contains(t, "ia"), strings.Contains(t, "ai"), strings.Contains(t, "automatización"):
		return "tech"
	case strings.Contains(t, "motivacional"), strings.Contains(t, "éxito"), strings.Contains(t, "crecimiento"):
		return "motivational"
	default:
		return "corporate"
	}
}

func pickRandomMusic(category string) (string, error) {
	category = strings.TrimSpace(strings.ToLower(category))
	if category == "" || category == "auto" {
		category = "corporate"
	}

	dir := filepath.Join(musicLibraryDir(), category)

	files, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("no se pudo leer carpeta de música %s: %w", dir, err)
	}

	candidates := []string{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if ext == ".mp3" || ext == ".wav" || ext == ".m4a" {
			candidates = append(candidates, filepath.Join(dir, f.Name()))
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no hay músicas en la categoría %s", category)
	}

	return candidates[rand.Intn(len(candidates))], nil
}

func downloadFile(ctx context.Context, fileURL, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
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
		ID:          uuid.NewString(),
		ClientID:    clientID,
		Prompt:      strings.TrimSpace(prompt),
		ImageURL:    strings.TrimSpace(imageURL),
		Provider:    "replicate",
		Model:       v.Model,
		Status:      "processing",
		CostCredits: duration,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	input := map[string]any{
		"prompt":       job.Prompt,
		"duration":     duration,
		"aspect_ratio": "9:16",
	}

	if job.ImageURL != "" {
		input["image"] = job.ImageURL
	}

	body, _ := json.Marshal(map[string]any{"input": input})
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
		Output any    `json:"output"`
		Error  any    `json:"error"`
		Urls   struct {
			Get string `json:"get"`
		} `json:"urls"`
	}

	_ = json.Unmarshal(raw, &parsed)

	if resp.StatusCode >= 300 {
		if strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "{}" {
			return models.AIVideoJob{}, fmt.Errorf("replicate no aceptó la solicitud. Revisa modelo, saldo, parámetros o intenta con prompt más corto")
		}
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
		job.ID, job.ClientID, job.Prompt, job.ImageURL, job.VideoURL, job.Provider, job.Model,
		job.ProviderJobID, job.ProviderGetURL, job.Status, job.Error,
		job.CostCredits, job.CreatedAt, job.UpdatedAt, job.CompletedAt,
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
		Output any    `json:"output"`
		Error  any    `json:"error"`
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

func (v *VideoService) AddMusicToJob(ctx context.Context, jobID, category string) (models.AIVideoJob, error) {
	job, err := v.GetJob(jobID)
	if err != nil {
		return models.AIVideoJob{}, err
	}

	if strings.TrimSpace(job.VideoURL) == "" {
		return models.AIVideoJob{}, fmt.Errorf("video no disponible")
	}

	if strings.TrimSpace(category) == "" || strings.ToLower(category) == "auto" {
		category = detectMusicCategory(job.Prompt)
	}

	musicPath, err := pickRandomMusic(category)
	if err != nil {
		return models.AIVideoJob{}, err
	}

	dir := videoAssetsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return models.AIVideoJob{}, err
	}

	tmpInput := filepath.Join(os.TempDir(), uuid.NewString()+"_input.mp4")
	outputName := uuid.NewString() + "_music.mp4"
	outputPath := filepath.Join(dir, outputName)

	if err := downloadFile(ctx, job.VideoURL, tmpInput); err != nil {
		return models.AIVideoJob{}, fmt.Errorf("descargando video: %w", err)
	}
	defer os.Remove(tmpInput)

	ffCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ffCtx,
		"ffmpeg",
		"-y",
		"-i", tmpInput,
		"-stream_loop", "-1",
		"-i", musicPath,
		"-filter_complex", "[1:a]volume=0.18[a]",
		"-map", "0:v:0",
		"-map", "[a]",
		"-shortest",
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return models.AIVideoJob{}, fmt.Errorf("ffmpeg error: %s", string(out))
	}

	now := time.Now()
	job.VideoURL = v.publicAssetURL(outputName)
	job.Status = "completed"
	job.Error = ""
	job.UpdatedAt = now
	job.CompletedAt = &now

	_, err = v.DB.Exec(`
		UPDATE ai_video_jobs
		SET video_url=?, status=?, error=?, updated_at=?, completed_at=?
		WHERE id=?
	`, job.VideoURL, job.Status, job.Error, job.UpdatedAt, job.CompletedAt, job.ID)

	if err != nil {
		return models.AIVideoJob{}, err
	}

	return job, nil
}

func normalizeVoiceText(text, fallback, language string) string {
	text = cleanSubtitleText(text)
	if text != "" {
		return text
	}

	fallback = cleanSubtitleText(fallback)
	if fallback != "" {
		return fallback
	}

	if language == "en" {
		return "Discover how artificial intelligence can help your business grow."
	}

	return "Descubre cómo la inteligencia artificial puede ayudar a tu negocio a crecer."
}

func pickOpenAIVoice(gender string) string {
	gender = strings.ToLower(strings.TrimSpace(gender))
	if gender == "male" {
		return "onyx"
	}
	return "nova"
}

func (v *VideoService) generateVoice(text, gender string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if strings.TrimSpace(apiKey) == "" {
		return "", fmt.Errorf("OPENAI_API_KEY no configurada")
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("texto de voz vacío")
	}

	if len(text) > 450 {
		text = text[:450]
	}

	payload := map[string]any{
		"model": "tts-1",
		"voice": pickOpenAIVoice(gender),
		"input": text,
	}

	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.openai.com/v1/audio/speech",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			ForceAttemptHTTP2: false,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("tts error: %s", string(raw))
	}

	if err := os.MkdirAll(videoAssetsDir(), 0o755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("voice_%d.mp3", time.Now().UnixNano())
	outPath := filepath.Join(videoAssetsDir(), filename)

	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return outPath, err
}

func cleanSubtitleText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, `"`, `'`)
	if len(s) > 220 {
		s = s[:220] + "..."
	}
	return s
}

func createSRT(text string, duration int, path string) error {
	if duration <= 0 {
		duration = 10
	}
	if duration > 20 {
		duration = 20
	}

	text = cleanSubtitleText(text)
	content := fmt.Sprintf(`1
00:00:00,000 --> 00:00:%02d,000
%s
`, duration, text)

	return os.WriteFile(path, []byte(content), 0o644)
}

func escapeSubtitlePath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.ReplaceAll(path, `\`, `/`)
	path = strings.ReplaceAll(path, ":", `\:`)
	path = strings.ReplaceAll(path, "'", `\'`)
	return path
}

func (v *VideoService) AddVoiceAndSubtitles(jobID string, opts VoiceSubtitleOptions) error {
	_, _ = v.DB.Exec(`
		UPDATE ai_video_jobs
		SET status='processing', error='', updated_at=?
		WHERE id=?
	`, time.Now(), jobID)

	job, err := v.GetJob(jobID)
	if err != nil {
		return err
	}

	if strings.TrimSpace(job.VideoURL) == "" {
		return fmt.Errorf("video vacío")
	}

	if !opts.EnableVoice && !opts.EnableSubtitles {
		err := fmt.Errorf("debes activar voz IA o subtítulos")
		_ = v.markVideoJobError(jobID, err)
		return err
	}

	opts.Language = strings.ToLower(strings.TrimSpace(opts.Language))
	if opts.Language != "en" {
		opts.Language = "es"
	}

	opts.Gender = strings.ToLower(strings.TrimSpace(opts.Gender))
	if opts.Gender != "male" {
		opts.Gender = "female"
	}

	voiceText := normalizeVoiceText(opts.Text, job.Prompt, opts.Language)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	tmpInput := filepath.Join(os.TempDir(), jobID+"_input.mp4")
	subPath := filepath.Join(os.TempDir(), jobID+".srt")

	if err := downloadFile(ctx, job.VideoURL, tmpInput); err != nil {
		_ = v.markVideoJobError(jobID, err)
		return err
	}
	defer os.Remove(tmpInput)

	var voicePath string

	if opts.EnableVoice {
		voicePath, err = v.generateVoice(voiceText, opts.Gender)
		if err != nil {
			_ = v.markVideoJobError(jobID, err)
			return err
		}
		defer os.Remove(voicePath)
	}

	if opts.EnableSubtitles {
		if err := createSRT(voiceText, job.CostCredits, subPath); err != nil {
			_ = v.markVideoJobError(jobID, err)
			return err
		}
		defer os.Remove(subPath)
	}

	if err := os.MkdirAll(videoAssetsDir(), 0o755); err != nil {
		_ = v.markVideoJobError(jobID, err)
		return err
	}

	outputName := fmt.Sprintf("video_final_%d.mp4", time.Now().UnixNano())
	outputPath := filepath.Join(videoAssetsDir(), outputName)

	args := []string{
		"-y",
		"-i", tmpInput,
	}

	if opts.EnableVoice {
		args = append(args, "-i", voicePath)
	}

	if opts.EnableSubtitles {
		subFilter := fmt.Sprintf(
			"subtitles='%s':force_style='FontName=Arial,FontSize=18,PrimaryColour=&H00FFFFFF,OutlineColour=&H00000000,BorderStyle=1,Outline=2,Shadow=1,Alignment=2,MarginV=70'",
			escapeSubtitlePath(subPath),
		)
		args = append(args, "-vf", subFilter)
	}

	args = append(args, "-map", "0:v:0")

	if opts.EnableVoice {
		args = append(args, "-map", "1:a:0")
	} else {
		args = append(args, "-an")
	}

	args = append(args,
		"-preset", "ultrafast",
		"-crf", "30",
		"-c:v", "libx264",
	)

	if opts.EnableVoice {
		args = append(args, "-c:a", "aac", "-b:a", "96k")
	}

	args = append(args,
		"-shortest",
		"-movflags", "+faststart",
		outputPath,
	)

	ffCtx, ffCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer ffCancel()

	cmd := exec.CommandContext(ffCtx, "ffmpeg", args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		ffErr := fmt.Errorf("ffmpeg: %s", string(out))
		_ = v.markVideoJobError(jobID, ffErr)
		return ffErr
	}

	finalURL := v.publicAssetURL(outputName)
	now := time.Now()

	_, err = v.DB.Exec(`
		UPDATE ai_video_jobs
		SET video_url=?, status='completed', error='', updated_at=?, completed_at=?
		WHERE id=?
	`, finalURL, now, now, jobID)

	return err
}

func (v *VideoService) markVideoJobError(jobID string, err error) error {
	_, dbErr := v.DB.Exec(`
		UPDATE ai_video_jobs
		SET status='error', error=?, updated_at=?
		WHERE id=?
	`, err.Error(), time.Now(), jobID)
	return dbErr
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