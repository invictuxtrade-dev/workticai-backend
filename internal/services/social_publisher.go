package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type SocialPublisher struct {
	DB   *sql.DB
	HTTP *http.Client
}

func NewSocialPublisher(db *sql.DB) *SocialPublisher {
	return &SocialPublisher{
		DB: db,
		HTTP: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (p *SocialPublisher) getFacebookCredential(clientID string) (models.SocialCredential, error) {
	var c models.SocialCredential

	err := p.DB.QueryRow(`
		SELECT
			id,
			client_id,
			platform,
			access_token,
			page_id,
			page_name,
			enabled,
			ad_account_id,
			instagram_account_id,
			instagram_username,
			instagram_connected,
			tiktok_access_token,
			tiktok_open_id,
			tiktok_connected,
			created_at,
			updated_at
		FROM social_credentials
		WHERE client_id=? AND platform='facebook'
		LIMIT 1
	`, clientID).Scan(
		&c.ID,
		&c.ClientID,
		&c.Platform,
		&c.AccessToken,
		&c.PageID,
		&c.PageName,
		&c.Enabled,
		&c.AdAccountID,
		&c.InstagramAccountID,
		&c.InstagramUsername,
		&c.InstagramConnected,
		&c.TikTokAccessToken,
		&c.TikTokOpenID,
		&c.TikTokConnected,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	return c, err
}

func (p *SocialPublisher) validateFacebookCredential(clientID string) (models.SocialCredential, error) {
	cred, err := p.getFacebookCredential(clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.SocialCredential{}, fmt.Errorf("no hay credenciales de Facebook configuradas")
		}
		return models.SocialCredential{}, err
	}

	if !cred.Enabled {
		return models.SocialCredential{}, fmt.Errorf("facebook está desactivado")
	}

	if strings.TrimSpace(cred.AccessToken) == "" || strings.TrimSpace(cred.PageID) == "" {
		return models.SocialCredential{}, fmt.Errorf("faltan access_token o page_id de Facebook")
	}

	return cred, nil
}

func (p *SocialPublisher) PublishFacebookPost(
	ctx context.Context,
	clientID string,
	content string,
	imageURL string,
	targetURL string,
) (string, error) {
	cred, err := p.validateFacebookCredential(clientID)
	if err != nil {
		return "", err
	}

	message := strings.TrimSpace(content)
	if strings.TrimSpace(targetURL) != "" {
		message = strings.TrimSpace(message + "\n\n" + strings.TrimSpace(targetURL))
	}

	form := url.Values{}
	form.Set("access_token", cred.AccessToken)
	form.Set("message", message)

	var endpoint string

	if strings.TrimSpace(imageURL) != "" {
		endpoint = fmt.Sprintf("https://graph.facebook.com/v19.0/%s/photos", cred.PageID)
		form.Set("url", strings.TrimSpace(imageURL))
	} else {
		endpoint = fmt.Sprintf("https://graph.facebook.com/v19.0/%s/feed", cred.PageID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("facebook publish error: %s", string(b))
	}

	var out struct {
		ID     string `json:"id"`
		PostID string `json:"post_id"`
	}

	if err := json.Unmarshal(b, &out); err != nil {
		return "", fmt.Errorf("facebook parse error: %w", err)
	}

	if strings.TrimSpace(out.PostID) != "" {
		return out.PostID, nil
	}
	if strings.TrimSpace(out.ID) != "" {
		return out.ID, nil
	}

	return "", fmt.Errorf("facebook no devolvió id: %s", string(b))
}

func (p *SocialPublisher) PublishFacebookVideo(
	ctx context.Context,
	clientID string,
	content string,
	videoURL string,
) (string, error) {
	cred, err := p.validateFacebookCredential(clientID)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(videoURL) == "" {
		return "", fmt.Errorf("video_url vacío")
	}

	endpoint := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/videos", cred.PageID)

	form := url.Values{}
	form.Set("access_token", cred.AccessToken)
	form.Set("description", strings.TrimSpace(content))
	form.Set("file_url", strings.TrimSpace(videoURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("facebook video error: %s", string(b))
	}

	var out struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(b, &out); err != nil {
		return "", fmt.Errorf("facebook video parse error: %w", err)
	}

	if strings.TrimSpace(out.ID) == "" {
		return "", fmt.Errorf("facebook no devolvió id del video: %s", string(b))
	}

	return out.ID, nil
}

func (p *SocialPublisher) PublishInstagramImage(
	ctx context.Context,
	clientID string,
	content string,
	imageURL string,
) (string, error) {
	cred, err := p.validateFacebookCredential(clientID)
	if err != nil {
		return "", err
	}

	if !cred.InstagramConnected {
		return "", fmt.Errorf("instagram no conectado")
	}

	if strings.TrimSpace(cred.InstagramAccountID) == "" {
		return "", fmt.Errorf("instagram_account_id vacío")
	}

	if strings.TrimSpace(imageURL) == "" {
		return "", fmt.Errorf("image_url vacío")
	}

	createURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/media", cred.InstagramAccountID)

	form := url.Values{}
	form.Set("image_url", strings.TrimSpace(imageURL))
	form.Set("caption", strings.TrimSpace(content))
	form.Set("access_token", cred.AccessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("instagram media error: %s", string(b))
	}

	var creation struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(b, &creation); err != nil {
		return "", fmt.Errorf("instagram media parse error: %w", err)
	}

	if strings.TrimSpace(creation.ID) == "" {
		return "", fmt.Errorf("instagram no devolvió creation_id: %s", string(b))
	}

	return p.publishInstagramCreation(ctx, cred, creation.ID)
}

func (p *SocialPublisher) PublishInstagramReel(
	ctx context.Context,
	clientID string,
	content string,
	videoURL string,
) (string, error) {
	cred, err := p.validateFacebookCredential(clientID)
	if err != nil {
		return "", err
	}

	if !cred.InstagramConnected {
		return "", fmt.Errorf("instagram no conectado")
	}

	if strings.TrimSpace(cred.InstagramAccountID) == "" {
		return "", fmt.Errorf("instagram_account_id vacío")
	}

	if strings.TrimSpace(videoURL) == "" {
		return "", fmt.Errorf("video_url vacío")
	}

	createURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/media", cred.InstagramAccountID)

	form := url.Values{}
	form.Set("media_type", "REELS")
	form.Set("video_url", strings.TrimSpace(videoURL))
	form.Set("caption", strings.TrimSpace(content))
	form.Set("share_to_feed", "true")
	form.Set("access_token", cred.AccessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("instagram reel create error: %s", string(b))
	}

	var creation struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(b, &creation); err != nil {
		return "", fmt.Errorf("instagram reel parse error: %w", err)
	}

	if strings.TrimSpace(creation.ID) == "" {
		return "", fmt.Errorf("instagram no devolvió creation_id: %s", string(b))
	}

	time.Sleep(8 * time.Second)

	return p.publishInstagramCreation(ctx, cred, creation.ID)
}

func (p *SocialPublisher) publishInstagramCreation(
	ctx context.Context,
	cred models.SocialCredential,
	creationID string,
) (string, error) {
	publishURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/media_publish", cred.InstagramAccountID)

	pubForm := url.Values{}
	pubForm.Set("creation_id", creationID)
	pubForm.Set("access_token", cred.AccessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, publishURL, strings.NewReader(pubForm.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	pubResp, err := p.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer pubResp.Body.Close()

	pubBytes, _ := io.ReadAll(pubResp.Body)

	if pubResp.StatusCode < 200 || pubResp.StatusCode >= 300 {
		return "", fmt.Errorf("instagram publish error: %s", string(pubBytes))
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(pubBytes, &result); err != nil {
		return "", fmt.Errorf("instagram publish parse error: %w", err)
	}

	if strings.TrimSpace(result.ID) == "" {
		return "", fmt.Errorf("instagram publish sin id: %s", string(pubBytes))
	}

	return result.ID, nil
}

func (p *SocialPublisher) PublishTikTokVideo(
	ctx context.Context,
	clientID string,
	content string,
	videoURL string,
) (string, error) {
	cred, err := p.validateFacebookCredential(clientID)
	if err != nil {
		return "", err
	}

	if !cred.TikTokConnected {
		return "", fmt.Errorf("tiktok no conectado")
	}

	if strings.TrimSpace(cred.TikTokAccessToken) == "" {
		return "", fmt.Errorf("tiktok access token vacío")
	}

	if strings.TrimSpace(videoURL) == "" {
		return "", fmt.Errorf("video_url vacío")
	}

	payload := map[string]any{
		"post_info": map[string]any{
			"title":                    strings.TrimSpace(content),
			"privacy_level":            "PUBLIC_TO_EVERYONE",
			"disable_comment":          false,
			"disable_duet":             false,
			"disable_stitch":           false,
			"video_cover_timestamp_ms": 1000,
		},
		"source_info": map[string]any{
			"source":    "PULL_FROM_URL",
			"video_url": strings.TrimSpace(videoURL),
		},
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://open.tiktokapis.com/v2/post/publish/video/init/",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+cred.TikTokAccessToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("tiktok publish error: %s", string(b))
	}

	var out struct {
		Data struct {
			PublishID string `json:"publish_id"`
			UploadURL string `json:"upload_url"`
		} `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			LogID   string `json:"log_id"`
		} `json:"error"`
	}

	if err := json.Unmarshal(b, &out); err != nil {
		return "", fmt.Errorf("tiktok parse error: %w", err)
	}

	if strings.TrimSpace(out.Error.Code) != "" && strings.TrimSpace(out.Error.Code) != "ok" {
		return "", fmt.Errorf("tiktok error: %s - %s", out.Error.Code, out.Error.Message)
	}

	if strings.TrimSpace(out.Data.PublishID) != "" {
		return out.Data.PublishID, nil
	}

	return "tiktok-video-published", nil
}

func (p *SocialPublisher) Log(
	clientID,
	campaignID,
	postID,
	level,
	message string,
) {
	_, _ = p.DB.Exec(`
		INSERT INTO social_logs (
			id,
			client_id,
			campaign_id,
			post_id,
			level,
			message,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		uuid.NewString(),
		clientID,
		campaignID,
		postID,
		level,
		message,
		time.Now(),
	)
}