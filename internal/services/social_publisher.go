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
			Timeout: 45 * time.Second,
		},
	}
}

func (p *SocialPublisher) getFacebookCredential(clientID string) (models.SocialCredential, error) {
	var c models.SocialCredential

	err := p.DB.QueryRow(`
		SELECT id, client_id, platform, access_token, page_id, page_name, enabled, ad_account_id, created_at, updated_at
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
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	return c, err
}

func (p *SocialPublisher) PublishFacebookPost(
	ctx context.Context,
	clientID string,
	content string,
	imageURL string,
	targetURL string,
) (string, error) {
	cred, err := p.getFacebookCredential(clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no hay credenciales de Facebook configuradas para este cliente")
		}
		return "", err
	}

	if !cred.Enabled {
		return "", fmt.Errorf("facebook está desactivado para este cliente")
	}

	if strings.TrimSpace(cred.AccessToken) == "" || strings.TrimSpace(cred.PageID) == "" {
		return "", fmt.Errorf("faltan access_token o page_id de Facebook")
	}

	message := strings.TrimSpace(content)
	if strings.TrimSpace(targetURL) != "" {
		message = strings.TrimSpace(message + "\n\n" + targetURL)
	}

	var endpoint string
	form := url.Values{}
	form.Set("access_token", cred.AccessToken)
	form.Set("message", message)

	if strings.TrimSpace(imageURL) != "" {
		endpoint = fmt.Sprintf("https://graph.facebook.com/v19.0/%s/photos", cred.PageID)
		form.Set("url", strings.TrimSpace(imageURL))
	} else {
		endpoint = fmt.Sprintf("https://graph.facebook.com/v19.0/%s/feed", cred.PageID)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		strings.NewReader(form.Encode()),
	)
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

	return "", fmt.Errorf("facebook no devolvió id de publicación")
}

func (p *SocialPublisher) Log(clientID, campaignID, postID, level, message string) {
	_, _ = p.DB.Exec(`
		INSERT INTO social_logs (
			id, client_id, campaign_id, post_id, level, message, created_at
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