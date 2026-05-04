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

type SocialService struct {
	DB        *sql.DB
	AI        *AIService
	Publisher *SocialPublisher
	Image     *SocialImageService
}

func NewSocialService(db *sql.DB, ai *AIService, assetsDir, baseURL string) *SocialService {
	return &SocialService{
		DB:        db,
		AI:        ai,
		Publisher: NewSocialPublisher(db),
		Image:     NewSocialImageService(ai, assetsDir, baseURL),
	}
}

func (s *SocialService) GenerateContent(ctx context.Context, campaign models.SocialCampaign) (string, error) {
	prompt := strings.TrimSpace(campaign.Prompt)
	if prompt == "" {
		prompt = "Genera una publicación profesional para Facebook e Instagram."
	}

	system := fmt.Sprintf(`
Eres experto en marketing digital y copywriting para Facebook e Instagram.

Objetivo: %s
CTA: %s

Devuelve:
1. Headline corto
2. Copy principal persuasivo
3. CTA
4. 5 a 8 hashtags

Reglas:
- escribe en español
- no uses markdown
- no expliques nada
- entrega texto listo para publicar
- estilo comercial, humano y profesional
`, campaign.Objective, campaign.CallToAction)

	return s.AI.GenerateHTML(ctx, system+"\n\n"+prompt, "")
}

func (s *SocialService) GenerateImage(ctx context.Context, imagePrompt string) (string, error) {
	imagePrompt = strings.TrimSpace(imagePrompt)
	if imagePrompt == "" {
		return "", fmt.Errorf("prompt de imagen vacío")
	}
	return s.Image.GenerateImage(ctx, imagePrompt)
}

func (s *SocialService) CreateCampaign(c models.SocialCampaign) (models.SocialCampaign, error) {
	now := time.Now()

	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.Status == "" {
		c.Status = "draft"
	}
	if strings.TrimSpace(c.ImageMode) == "" {
		c.ImageMode = "none"
	}
	if strings.TrimSpace(c.PublishMode) == "" {
		c.PublishMode = "now"
	}

	c.CreatedAt = now
	c.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO social_campaigns (
			id, client_id, name, objective, bot_id, landing_id, prompt, status,
			image_mode, image_prompt, manual_image_url, manual_link_url, call_to_action,
			publish_mode, recurring_minutes, days_of_week, scheduled_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		c.ID,
		c.ClientID,
		c.Name,
		c.Objective,
		c.BotID,
		c.LandingID,
		c.Prompt,
		c.Status,
		c.ImageMode,
		c.ImagePrompt,
		c.ManualImageURL,
		c.ManualLinkURL,
		c.CallToAction,
		c.PublishMode,
		c.RecurringMinutes,
		c.DaysOfWeek,
		c.ScheduledAt,
		c.CreatedAt,
		c.UpdatedAt,
	)

	return c, err
}

func (s *SocialService) CreatePost(
	clientID string,
	campaignID string,
	platform string,
	content string,
	imageURL string,
	targetURL string,
	publishMode string,
	imageMode string,
	imagePrompt string,
	scheduledAt *time.Time,
) (models.SocialPost, error) {
	now := time.Now()

	if strings.TrimSpace(platform) == "" {
		platform = "facebook"
	}
	if strings.TrimSpace(publishMode) == "" {
		publishMode = "now"
	}
	if strings.TrimSpace(imageMode) == "" {
		imageMode = "none"
	}

	post := models.SocialPost{
		ID:          uuid.NewString(),
		ClientID:    clientID,
		CampaignID:  campaignID,
		Platform:    strings.TrimSpace(platform),
		Content:     strings.TrimSpace(content),
		ImageURL:    strings.TrimSpace(imageURL),
		TargetURL:   strings.TrimSpace(targetURL),
		PublishMode: strings.TrimSpace(publishMode),
		ImageMode:   strings.TrimSpace(imageMode),
		ImagePrompt: strings.TrimSpace(imagePrompt),
		Status:      "draft",
		Error:       "",
		ScheduledAt: scheduledAt,
		CreatedAt:   now,
	}

	_, err := s.DB.Exec(`
		INSERT INTO social_posts (
			id, client_id, campaign_id, platform, content, image_url, target_url,
			publish_mode, image_mode, image_prompt, status, error, facebook_post_id,
			scheduled_at, published_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		post.ID,
		post.ClientID,
		post.CampaignID,
		post.Platform,
		post.Content,
		post.ImageURL,
		post.TargetURL,
		post.PublishMode,
		post.ImageMode,
		post.ImagePrompt,
		post.Status,
		post.Error,
		"",
		post.ScheduledAt,
		post.PublishedAt,
		post.CreatedAt,
	)
	if err != nil {
		return models.SocialPost{}, err
	}

	return post, nil
}

func (s *SocialService) ResolveTargetURL(clientID, objective, botID, landingID, manualLink string) string {
	switch strings.TrimSpace(objective) {
	case "manual_link":
		return strings.TrimSpace(manualLink)

	case "landing":
		var target string
		_ = s.DB.QueryRow(`
			SELECT whatsapp_url
			FROM landing_pages
			WHERE id=? AND client_id=?
			LIMIT 1
		`, landingID, clientID).Scan(&target)
		return strings.TrimSpace(target)

	case "whatsapp":
		var phone string
		_ = s.DB.QueryRow(`
			SELECT phone
			FROM bots
			WHERE id=? AND client_id=?
			LIMIT 1
		`, botID, clientID).Scan(&phone)

		phone = strings.TrimSpace(strings.TrimPrefix(phone, "+"))
		if phone == "" {
			return ""
		}
		return "https://wa.me/" + phone

	default:
		return ""
	}
}

func (s *SocialService) GetInstagramFromPage(accessToken, pageID string) (string, string, error) {
	if strings.TrimSpace(accessToken) == "" || strings.TrimSpace(pageID) == "" {
		return "", "", fmt.Errorf("token o page_id vacío")
	}

	graphURL := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/%s?fields=instagram_business_account{id,username}&access_token=%s",
		pageID,
		url.QueryEscape(accessToken),
	)

	resp, err := http.Get(graphURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var data struct {
		Instagram struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"instagram_business_account"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    int    `json:"code"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}

	if data.Error != nil {
		return "", "", fmt.Errorf("Meta error: %s", data.Error.Message)
	}

	return data.Instagram.ID, data.Instagram.Username, nil
}

func (s *SocialService) PublishNow(ctx context.Context, postID string) error {
	var p models.SocialPost

	err := s.DB.QueryRow(`
		SELECT id, client_id, campaign_id, platform, content, image_url, target_url,
		       publish_mode, image_mode, image_prompt, status, error,
		       facebook_post_id, scheduled_at, published_at, created_at
		FROM social_posts
		WHERE id=?
	`, postID).Scan(
		&p.ID,
		&p.ClientID,
		&p.CampaignID,
		&p.Platform,
		&p.Content,
		&p.ImageURL,
		&p.TargetURL,
		&p.PublishMode,
		&p.ImageMode,
		&p.ImagePrompt,
		&p.Status,
		&p.Error,
		&p.FacebookPostID,
		&p.ScheduledAt,
		&p.PublishedAt,
		&p.CreatedAt,
	)
	if err != nil {
		return err
	}

	platform := strings.ToLower(strings.TrimSpace(p.Platform))

	switch platform {
	case "facebook":
		return s.publishFacebook(ctx, p)
	case "instagram":
		return s.publishInstagram(ctx, p)
	default:
		return fmt.Errorf("plataforma no soportada: %s", p.Platform)
	}
}

func (s *SocialService) publishFacebook(ctx context.Context, p models.SocialPost) error {
	postIDFB, err := s.Publisher.PublishFacebookPost(
		ctx,
		p.ClientID,
		p.Content,
		p.ImageURL,
		p.TargetURL,
	)
	if err != nil {
		_, _ = s.DB.Exec(`UPDATE social_posts SET status='error', error=? WHERE id=?`, err.Error(), p.ID)
		s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "error", err.Error())
		return err
	}

	now := time.Now()

	_, _ = s.DB.Exec(`
		UPDATE social_posts
		SET status='published', error='', facebook_post_id=?, published_at=?
		WHERE id=?
	`, postIDFB, now, p.ID)

	s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "info", "publicación enviada a Facebook: "+postIDFB)
	return nil
}

func (s *SocialService) publishInstagram(ctx context.Context, p models.SocialPost) error {
	imageURL := strings.TrimSpace(p.ImageURL)
	if imageURL == "" {
		err := fmt.Errorf("Instagram requiere una imagen pública HTTPS")
		_, _ = s.DB.Exec(`UPDATE social_posts SET status='error', error=? WHERE id=?`, err.Error(), p.ID)
		s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "error", err.Error())
		return err
	}

	if !strings.HasPrefix(imageURL, "https://") {
		err := fmt.Errorf("la imagen para Instagram debe ser una URL pública HTTPS")
		_, _ = s.DB.Exec(`UPDATE social_posts SET status='error', error=? WHERE id=?`, err.Error(), p.ID)
		s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "error", err.Error())
		return err
	}

	cred, err := s.GetCredentialByClient(p.ClientID)
	if err != nil {
		_, _ = s.DB.Exec(`UPDATE social_posts SET status='error', error=? WHERE id=?`, err.Error(), p.ID)
		s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "error", err.Error())
		return err
	}

	igID := strings.TrimSpace(cred.InstagramAccountID)
	if igID == "" {
		igID, _, err = s.GetInstagramFromPage(cred.AccessToken, cred.PageID)
		if err != nil || igID == "" {
			if err == nil {
				err = fmt.Errorf("Instagram no conectado")
			}
			_, _ = s.DB.Exec(`UPDATE social_posts SET status='error', error=? WHERE id=?`, err.Error(), p.ID)
			s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "error", err.Error())
			return err
		}
	}

	createForm := url.Values{}
	createForm.Set("image_url", imageURL)
	createForm.Set("caption", strings.TrimSpace(p.Content))
	createForm.Set("access_token", cred.AccessToken)

	createURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/media", igID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, strings.NewReader(createForm.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var container struct {
		ID    string `json:"id"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    int    `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(raw, &container); err != nil {
		err = fmt.Errorf("respuesta inválida creando media IG: %s", string(raw))
		_, _ = s.DB.Exec(`UPDATE social_posts SET status='error', error=? WHERE id=?`, err.Error(), p.ID)
		s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "error", err.Error())
		return err
	}

	if resp.StatusCode >= 300 || container.Error != nil || container.ID == "" {
		msg := string(raw)
		if container.Error != nil {
			msg = container.Error.Message
		}
		err := fmt.Errorf("error creando media IG: %s", msg)
		_, _ = s.DB.Exec(`UPDATE social_posts SET status='error', error=? WHERE id=?`, err.Error(), p.ID)
		s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "error", err.Error())
		return err
	}

	time.Sleep(3 * time.Second)

	publishForm := url.Values{}
	publishForm.Set("creation_id", container.ID)
	publishForm.Set("access_token", cred.AccessToken)

	publishURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/media_publish", igID)

	req2, err := http.NewRequestWithContext(ctx, http.MethodPost, publishURL, strings.NewReader(publishForm.Encode()))
	if err != nil {
		return err
	}
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	raw2, _ := io.ReadAll(resp2.Body)

	var published struct {
		ID    string `json:"id"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    int    `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(raw2, &published); err != nil {
		err = fmt.Errorf("respuesta inválida publicando IG: %s", string(raw2))
		_, _ = s.DB.Exec(`UPDATE social_posts SET status='error', error=? WHERE id=?`, err.Error(), p.ID)
		s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "error", err.Error())
		return err
	}

	if resp2.StatusCode >= 300 || published.Error != nil || published.ID == "" {
		msg := string(raw2)
		if published.Error != nil {
			msg = published.Error.Message
		}
		err := fmt.Errorf("error publicando IG: %s", msg)
		_, _ = s.DB.Exec(`UPDATE social_posts SET status='error', error=? WHERE id=?`, err.Error(), p.ID)
		s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "error", err.Error())
		return err
	}

	now := time.Now()

	_, _ = s.DB.Exec(`
		UPDATE social_posts
		SET status='published', error='', facebook_post_id=?, published_at=?
		WHERE id=?
	`, published.ID, now, p.ID)

	s.Publisher.Log(p.ClientID, p.CampaignID, p.ID, "info", "publicación enviada a Instagram: "+published.ID)
	return nil
}

func (s *SocialService) SaveCredential(c models.SocialCredential) (models.SocialCredential, error) {
	now := time.Now()

	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if strings.TrimSpace(c.Platform) == "" {
		c.Platform = "facebook"
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO social_credentials (
			id, client_id, platform, access_token, page_id, page_name,
			enabled, ad_account_id,
			instagram_account_id, instagram_username,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			client_id=excluded.client_id,
			platform=excluded.platform,
			access_token=excluded.access_token,
			page_id=excluded.page_id,
			page_name=excluded.page_name,
			enabled=excluded.enabled,
			ad_account_id=excluded.ad_account_id,
			instagram_account_id=excluded.instagram_account_id,
			instagram_username=excluded.instagram_username,
			updated_at=excluded.updated_at
	`,
		c.ID,
		c.ClientID,
		c.Platform,
		c.AccessToken,
		c.PageID,
		c.PageName,
		c.Enabled,
		c.AdAccountID,
		c.InstagramAccountID,
		c.InstagramUsername,
		c.CreatedAt,
		c.UpdatedAt,
	)

	return c, err
}

func (s *SocialService) GetCredentialByClient(clientID string) (*models.SocialCredential, error) {
	var c models.SocialCredential

	err := s.DB.QueryRow(`
		SELECT 
			id, client_id, platform, access_token, page_id, page_name,
			enabled, ad_account_id,
			instagram_account_id, instagram_username,
			created_at, updated_at
		FROM social_credentials
		WHERE client_id=? AND platform='facebook' AND enabled=1
		ORDER BY updated_at DESC
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
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("credenciales no encontradas")
	}

	return &c, nil
}