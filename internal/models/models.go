package models

import "time"

type User struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id,omitempty"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}

type Client struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	Plan      string    `json:"plan"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Bot struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone,omitempty"`
	Status    string    `json:"status"`
	LastQR    string    `json:"last_qr,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BotConfig struct {
	BotID               string    `json:"bot_id"`
	SystemPrompt        string    `json:"system_prompt"`
	BusinessName        string    `json:"business_name"`
	BusinessDescription string    `json:"business_description"`
	Offer               string    `json:"offer"`
	TargetAudience      string    `json:"target_audience"`
	Tone                string    `json:"tone"`
	CTAButtonText       string    `json:"cta_button_text"`
	CTALink             string    `json:"cta_link"`
	FallbackMessage     string    `json:"fallback_message"`
	HumanHandoffPhone   string    `json:"human_handoff_phone"`
	Temperature         float64   `json:"temperature"`
	Model               string    `json:"model"`
	FollowupEnabled     bool      `json:"followup_enabled"`
	FollowupDelayMins   int       `json:"followup_delay_mins"`
	UpdatedAt           time.Time `json:"updated_at"`
	ReplyMode           string    `json:"reply_mode"`  // manual | template_only | template_ai
	TemplateID          string    `json:"template_id"` // plantilla fija opcional
}

type Lead struct {
	ID              int64      `json:"id"`
	BotID           string     `json:"bot_id"`
	ClientID        string     `json:"client_id,omitempty"`
	BotName         string     `json:"bot_name,omitempty"`
	ClientName      string     `json:"client_name,omitempty"`
	ChatJID         string     `json:"chat_jid"`
	DisplayName     string     `json:"display_name"`
	Phone           string     `json:"phone"`
	Stage           string     `json:"stage"`
	LastIntent      string     `json:"last_intent"`
	Summary         string     `json:"summary"`
	Tags            string     `json:"tags"`
	LastInboundText string     `json:"last_inbound_text"`
	LastReplyText   string     `json:"last_reply_text"`
	FollowupCount   int        `json:"followup_count"`
	NextFollowupAt  *time.Time `json:"next_followup_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LastMessageAt   *time.Time `json:"last_message_at,omitempty"`
}

type Message struct {
	ID        int64     `json:"id"`
	BotID     string    `json:"bot_id"`
	ChatJID   string    `json:"chat_jid"`
	Direction string    `json:"direction"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Template struct {
	ID              string    `json:"id"`
	ClientID        string    `json:"client_id,omitempty"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	BusinessType    string    `json:"business_type"`
	Stage           string    `json:"stage"`
	PromptSnippet   string    `json:"prompt_snippet"`
	MessageTemplate string    `json:"message_template"`
	IsDefault       bool      `json:"is_default"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Metrics struct {
	Clients     int `json:"clients"`
	Bots        int `json:"bots"`
	Leads       int `json:"leads"`
	HotLeads    int `json:"hot_leads"`
	ClosedLeads int `json:"closed_leads"`
	Messages24h int `json:"messages_24h"`
}

type LandingPage struct {
	ID              string    `json:"id"`
	ClientID        string    `json:"client_id"`
	BotID           string    `json:"bot_id"`
	Name            string    `json:"name"`
	Prompt          string    `json:"prompt"`
	Status          string    `json:"status"` // draft | generated | published
	StylePreset     string    `json:"style_preset"`
	LogoURL         string    `json:"logo_url"`
	FaviconURL      string    `json:"favicon_url"`
	HeroImageURL    string    `json:"hero_image_url"`
	YoutubeURL      string    `json:"youtube_url"`
	FacebookPixelID string    `json:"facebook_pixel_id"`
	GoogleAnalytics string    `json:"google_analytics"`
	PrimaryColor    string    `json:"primary_color"`
	SecondaryColor  string    `json:"secondary_color"`
	ShowVideo       bool      `json:"show_video"`
	ShowImage       bool      `json:"show_image"`
	Html            string    `json:"html"`
	Css             string    `json:"css"`
	Js              string    `json:"js"`
	PreviewHTML     string    `json:"preview_html"`
	WhatsappURL     string    `json:"whatsapp_url"`
	TrackingMode    string    `json:"tracking_mode"`     // auto | external
	TrackingBaseURL string    `json:"tracking_base_url"` // ej: http://localhost:8080 o https://panel.midominio.com
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ================= SOCIAL IA =================

type SocialCredential struct {
	ID          string    `json:"id"`
	ClientID    string    `json:"client_id"`
	Platform    string    `json:"platform"` // facebook
	AccessToken string    `json:"access_token"`
	PageID      string    `json:"page_id"`
	PageName    string    `json:"page_name"`
	Enabled     bool      `json:"enabled"`
	AdAccountID string    `json:"ad_account_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SocialCampaign struct {
	ID               string     `json:"id"`
	ClientID         string     `json:"client_id"`
	Name             string     `json:"name"`
	Objective        string     `json:"objective"` // whatsapp | landing | manual_link | no_link
	BotID            string     `json:"bot_id"`
	LandingID        string     `json:"landing_id"`
	Prompt           string     `json:"prompt"`
	ImageMode        string     `json:"image_mode"` // ai | manual | none
	ImagePrompt      string     `json:"image_prompt"`
	ManualImageURL   string     `json:"manual_image_url"`
	ManualLinkURL    string     `json:"manual_link_url"`
	CallToAction     string     `json:"call_to_action"`
	PublishMode      string     `json:"publish_mode"` // now | scheduled | recurring
	RecurringMinutes int        `json:"recurring_minutes"`
	DaysOfWeek       string     `json:"days_of_week"` // mon,tue,wed
	ScheduledAt      *time.Time `json:"scheduled_at,omitempty"`
	Status           string     `json:"status"` // draft | active | paused
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type SocialPost struct {
	ID             string     `json:"id"`
	ClientID       string     `json:"client_id"`
	CampaignID     string     `json:"campaign_id"`
	Platform       string     `json:"platform"` // facebook
	Content        string     `json:"content"`
	ImageURL       string     `json:"image_url"`
	TargetURL      string     `json:"target_url"`
	PublishMode    string     `json:"publish_mode"`
	ImageMode      string     `json:"image_mode"`   // ai | manual | none
	ImagePrompt    string     `json:"image_prompt"` // prompt usado para la imagen IA
	Status         string     `json:"status"`       // draft | scheduled | published | error
	Error          string     `json:"error"`
	FacebookPostID string     `json:"facebook_post_id"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type SocialJob struct {
	ID               string     `json:"id"`
	ClientID         string     `json:"client_id"`
	CampaignID       string     `json:"campaign_id"`
	PostID           string     `json:"post_id"`
	JobType          string     `json:"job_type"` // publish_once | recurring_publish
	RunAt            time.Time  `json:"run_at"`
	RecurringMinutes int        `json:"recurring_minutes"`
	DaysOfWeek       string     `json:"days_of_week"`
	Status           string     `json:"status"` // pending | running | done | error | cancelled
	LastError        string     `json:"last_error"`
	LastRunAt        *time.Time `json:"last_run_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type SocialLog struct {
	ID         string    `json:"id"`
	ClientID   string    `json:"client_id"`
	CampaignID string    `json:"campaign_id"`
	PostID     string    `json:"post_id"`
	Level      string    `json:"level"` // info | error
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}