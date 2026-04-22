package services

import (
	"context"
	"database/sql"
	"fmt"
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

// ================= GENERAR CONTENIDO =================

func (s *SocialService) GenerateContent(ctx context.Context, campaign models.SocialCampaign) (string, error) {
	prompt := strings.TrimSpace(campaign.Prompt)
	if prompt == "" {
		prompt = "Genera una publicación profesional para Facebook."
	}

	system := fmt.Sprintf(`
Eres experto en marketing digital y copywriting para Facebook.

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

// ================= GENERAR IMAGEN =================

func (s *SocialService) GenerateImage(ctx context.Context, imagePrompt string) (string, error) {
	imagePrompt = strings.TrimSpace(imagePrompt)
	if imagePrompt == "" {
		return "", fmt.Errorf("prompt de imagen vacío")
	}
	return s.Image.GenerateImage(ctx, imagePrompt)
}

// ================= CREAR CAMPAÑA =================

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

// ================= CREAR POST =================

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

// ================= RESOLVER LINK =================

func (s *SocialService) ResolveTargetURL(clientID, objective, botID, landingID, manualLink string) string {
	switch strings.TrimSpace(objective) {
	case "manual_link":
		return strings.TrimSpace(manualLink)

	case "landing":
		// Aquí luego podrás cambiarlo por public_url cuando la guardes en DB
		var url string
		_ = s.DB.QueryRow(`
			SELECT whatsapp_url
			FROM landing_pages
			WHERE id=? AND client_id=?
			LIMIT 1
		`, landingID, clientID).Scan(&url)
		return strings.TrimSpace(url)

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

// ================= PUBLICAR =================

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

	if p.Platform != "facebook" {
		return fmt.Errorf("solo Facebook soportado")
	}

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

// ================= GUARDAR CREDENCIALES =================

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
			id, client_id, platform, access_token, page_id, page_name, enabled, ad_account_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			client_id=excluded.client_id,
			platform=excluded.platform,
			access_token=excluded.access_token,
			page_id=excluded.page_id,
			page_name=excluded.page_name,
			enabled=excluded.enabled,
			ad_account_id=excluded.ad_account_id,
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
		c.CreatedAt,
		c.UpdatedAt,
	)

	return c, err
}