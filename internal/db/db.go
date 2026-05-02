package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	// Migraciones suaves para instalaciones existentes
	softMigrations := []string{
	// bot_configs
	`ALTER TABLE bot_configs ADD COLUMN reply_mode TEXT DEFAULT 'manual'`,
	`ALTER TABLE bot_configs ADD COLUMN template_id TEXT DEFAULT ''`,

	// landing_pages
	`ALTER TABLE landing_pages ADD COLUMN style_preset TEXT DEFAULT 'dark_premium'`,
	`ALTER TABLE landing_pages ADD COLUMN logo_url TEXT DEFAULT ''`,
	`ALTER TABLE landing_pages ADD COLUMN favicon_url TEXT DEFAULT ''`,
	`ALTER TABLE landing_pages ADD COLUMN hero_image_url TEXT DEFAULT ''`,
	`ALTER TABLE landing_pages ADD COLUMN youtube_url TEXT DEFAULT ''`,
	`ALTER TABLE landing_pages ADD COLUMN facebook_pixel_id TEXT DEFAULT ''`,
	`ALTER TABLE landing_pages ADD COLUMN google_analytics TEXT DEFAULT ''`,
	`ALTER TABLE landing_pages ADD COLUMN primary_color TEXT DEFAULT '#2563eb'`,
	`ALTER TABLE landing_pages ADD COLUMN secondary_color TEXT DEFAULT '#0f172a'`,
	`ALTER TABLE landing_pages ADD COLUMN show_video INTEGER DEFAULT 0`,
	`ALTER TABLE landing_pages ADD COLUMN show_image INTEGER DEFAULT 0`,
	`ALTER TABLE landing_pages ADD COLUMN tracking_mode TEXT DEFAULT 'auto'`,
	`ALTER TABLE landing_pages ADD COLUMN tracking_base_url TEXT DEFAULT ''`,

	// social_credentials
	`ALTER TABLE social_credentials ADD COLUMN page_name TEXT DEFAULT ''`,
	`ALTER TABLE social_credentials ADD COLUMN enabled INTEGER DEFAULT 1`,

	// social_campaigns
	`ALTER TABLE social_campaigns ADD COLUMN image_mode TEXT DEFAULT 'ai'`,
	`ALTER TABLE social_campaigns ADD COLUMN image_prompt TEXT DEFAULT ''`,
	`ALTER TABLE social_campaigns ADD COLUMN manual_image_url TEXT DEFAULT ''`,
	`ALTER TABLE social_campaigns ADD COLUMN manual_link_url TEXT DEFAULT ''`,
	`ALTER TABLE social_campaigns ADD COLUMN call_to_action TEXT DEFAULT ''`,
	`ALTER TABLE social_campaigns ADD COLUMN publish_mode TEXT DEFAULT 'now'`,
	`ALTER TABLE social_campaigns ADD COLUMN recurring_minutes INTEGER DEFAULT 0`,
	`ALTER TABLE social_campaigns ADD COLUMN days_of_week TEXT DEFAULT ''`,
	`ALTER TABLE social_campaigns ADD COLUMN scheduled_at TIMESTAMP NULL`,

	// social_posts
	`ALTER TABLE social_posts ADD COLUMN publish_mode TEXT DEFAULT 'now'`,
	`ALTER TABLE social_posts ADD COLUMN image_mode TEXT DEFAULT 'none'`,
	`ALTER TABLE social_posts ADD COLUMN image_prompt TEXT DEFAULT ''`,
	`ALTER TABLE social_posts ADD COLUMN facebook_post_id TEXT DEFAULT ''`,
	`ALTER TABLE social_posts ADD COLUMN published_at TIMESTAMP NULL`,

	`ALTER TABLE clients ADD COLUMN plan TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE ads_campaigns ADD COLUMN bot_id TEXT DEFAULT ''`,
	`ALTER TABLE ads_campaigns ADD COLUMN landing_id TEXT DEFAULT ''`,
	`ALTER TABLE ads_campaigns ADD COLUMN ecosystem_status TEXT DEFAULT 'draft'`,
	`ALTER TABLE ads_campaigns ADD COLUMN auto_bot_enabled INTEGER DEFAULT 1`,
	`ALTER TABLE ads_campaigns ADD COLUMN auto_landing_enabled INTEGER DEFAULT 1`,
	`ALTER TABLE ads_campaigns ADD COLUMN auto_creatives_enabled INTEGER DEFAULT 1`,
	`ALTER TABLE group_bots ADD COLUMN group_jid TEXT DEFAULT ''`,
	// facebook groups growth
	`ALTER TABLE facebook_group_targets ADD COLUMN auto_join_enabled INTEGER DEFAULT 0`,
	`ALTER TABLE facebook_group_targets ADD COLUMN last_join_attempt TIMESTAMP NULL`,
	`ALTER TABLE facebook_group_targets ADD COLUMN joined_at TIMESTAMP NULL`,
}

	for _, q := range softMigrations {
		_, _ = db.Exec(q)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	queries := []string{
		`PRAGMA journal_mode=WAL;`,

		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			client_id TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMP NOT NULL
		);`,

		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`,

		`CREATE TABLE IF NOT EXISTS clients (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			plan TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);`,

			`CREATE TABLE IF NOT EXISTS plans (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE,
		description TEXT NOT NULL DEFAULT '',
		price_monthly REAL NOT NULL DEFAULT 0,
		price_yearly REAL NOT NULL DEFAULT 0,
		features TEXT NOT NULL DEFAULT '[]',
		is_free INTEGER NOT NULL DEFAULT 0,
		is_active INTEGER NOT NULL DEFAULT 1,
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS subscriptions (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		plan_id TEXT NOT NULL,
		plan_slug TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		billing_cycle TEXT NOT NULL DEFAULT 'monthly',
		amount REAL NOT NULL DEFAULT 0,
		payment_method TEXT NOT NULL DEFAULT 'usdt_bep20',
		tx_hash TEXT NOT NULL DEFAULT '',
		wallet_address TEXT NOT NULL DEFAULT '',
		paid_at TIMESTAMP NULL,
		starts_at TIMESTAMP NULL,
		expires_at TIMESTAMP NULL,
		validated_by TEXT NOT NULL DEFAULT '',
		validation_notes TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS plan_config (
		id TEXT PRIMARY KEY,
		usdt_bep20_wallet TEXT NOT NULL DEFAULT '',
		card_payments_enabled INTEGER NOT NULL DEFAULT 0,
		default_free_plan_slug TEXT NOT NULL DEFAULT 'free',
		require_plan_selection INTEGER NOT NULL DEFAULT 1,
		updated_at TIMESTAMP NOT NULL
	);`,


		`CREATE TABLE IF NOT EXISTS bots (
			id TEXT PRIMARY KEY,
			client_id TEXT NOT NULL,
			name TEXT NOT NULL,
			phone TEXT,
			status TEXT NOT NULL DEFAULT 'created',
			last_qr TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			FOREIGN KEY (client_id) REFERENCES clients(id)
		);`,

		`CREATE TABLE IF NOT EXISTS bot_configs (
			bot_id TEXT PRIMARY KEY,
			system_prompt TEXT NOT NULL DEFAULT '',
			business_name TEXT NOT NULL DEFAULT '',
			business_description TEXT NOT NULL DEFAULT '',
			offer TEXT NOT NULL DEFAULT '',
			target_audience TEXT NOT NULL DEFAULT '',
			tone TEXT NOT NULL DEFAULT 'professional',
			cta_button_text TEXT NOT NULL DEFAULT 'Quiero más información',
			cta_link TEXT NOT NULL DEFAULT '',
			fallback_message TEXT NOT NULL DEFAULT 'Gracias por escribirnos. En breve te ayudamos.',
			human_handoff_phone TEXT NOT NULL DEFAULT '',
			temperature REAL NOT NULL DEFAULT 0.7,
			model TEXT NOT NULL DEFAULT 'gpt-4o-mini',
			followup_enabled INTEGER NOT NULL DEFAULT 1,
			followup_delay_mins INTEGER NOT NULL DEFAULT 60,
			reply_mode TEXT NOT NULL DEFAULT 'manual',
			template_id TEXT NOT NULL DEFAULT '',
			updated_at TIMESTAMP NOT NULL,
			FOREIGN KEY (bot_id) REFERENCES bots(id)
		);`,

		`CREATE TABLE IF NOT EXISTS templates (
			id TEXT PRIMARY KEY,
			client_id TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'sales',
			business_type TEXT NOT NULL DEFAULT 'general',
			stage TEXT NOT NULL DEFAULT 'new',
			prompt_snippet TEXT NOT NULL DEFAULT '',
			message_template TEXT NOT NULL DEFAULT '',
			is_default INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);`,

	   `CREATE TABLE IF NOT EXISTS landing_pages (
			id TEXT PRIMARY KEY,
			client_id TEXT NOT NULL,
			bot_id TEXT NOT NULL,
			name TEXT NOT NULL,
			prompt TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'draft',
			style_preset TEXT NOT NULL DEFAULT 'dark_premium',
			logo_url TEXT NOT NULL DEFAULT '',
			favicon_url TEXT NOT NULL DEFAULT '',
			hero_image_url TEXT NOT NULL DEFAULT '',
			youtube_url TEXT NOT NULL DEFAULT '',
			facebook_pixel_id TEXT NOT NULL DEFAULT '',
			google_analytics TEXT NOT NULL DEFAULT '',
			primary_color TEXT NOT NULL DEFAULT '#2563eb',
			secondary_color TEXT NOT NULL DEFAULT '#0f172a',
			show_video INTEGER NOT NULL DEFAULT 0,
			show_image INTEGER NOT NULL DEFAULT 0,
			html TEXT NOT NULL DEFAULT '',
			css TEXT NOT NULL DEFAULT '',
			js TEXT NOT NULL DEFAULT '',
			preview_html TEXT NOT NULL DEFAULT '',
			whatsapp_url TEXT NOT NULL DEFAULT '',
			tracking_mode TEXT NOT NULL DEFAULT 'auto',
			tracking_base_url TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			FOREIGN KEY (client_id) REFERENCES clients(id),
			FOREIGN KEY (bot_id) REFERENCES bots(id)
		);`,

		`CREATE TABLE IF NOT EXISTS leads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bot_id TEXT NOT NULL,
			chat_jid TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			stage TEXT NOT NULL DEFAULT 'new',
			last_intent TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			last_inbound_text TEXT NOT NULL DEFAULT '',
			last_reply_text TEXT NOT NULL DEFAULT '',
			followup_count INTEGER NOT NULL DEFAULT 0,
			next_followup_at TIMESTAMP NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_message_at TIMESTAMP NULL,
			UNIQUE(bot_id, chat_jid),
			FOREIGN KEY (bot_id) REFERENCES bots(id)
		);`,

		`CREATE TABLE IF NOT EXISTS funnel_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		client_id TEXT NOT NULL DEFAULT '',
		bot_id TEXT NOT NULL DEFAULT '',
		landing_id TEXT NOT NULL DEFAULT '',
		event_type TEXT NOT NULL,
		metadata TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bot_id TEXT NOT NULL,
			chat_jid TEXT NOT NULL,
			direction TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			FOREIGN KEY (bot_id) REFERENCES bots(id)
		);`,

		// ================= SOCIAL IA =================

		`CREATE TABLE IF NOT EXISTS social_credentials (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		platform TEXT NOT NULL,
		access_token TEXT NOT NULL,
		page_id TEXT NOT NULL,
		page_name TEXT NOT NULL DEFAULT '',
		enabled INTEGER NOT NULL DEFAULT 1,
		ad_account_id TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,
	

		`CREATE TABLE IF NOT EXISTS social_campaigns (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		name TEXT NOT NULL,
		objective TEXT NOT NULL,
		bot_id TEXT NOT NULL DEFAULT '',
		landing_id TEXT NOT NULL DEFAULT '',
		prompt TEXT NOT NULL DEFAULT '',
		image_mode TEXT NOT NULL DEFAULT 'ai',
		image_prompt TEXT NOT NULL DEFAULT '',
		manual_image_url TEXT NOT NULL DEFAULT '',
		manual_link_url TEXT NOT NULL DEFAULT '',
		call_to_action TEXT NOT NULL DEFAULT '',
		publish_mode TEXT NOT NULL DEFAULT 'now',
		recurring_minutes INTEGER NOT NULL DEFAULT 0,
		days_of_week TEXT NOT NULL DEFAULT '',
		scheduled_at TIMESTAMP NULL,
		status TEXT NOT NULL DEFAULT 'draft',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,

		`CREATE TABLE IF NOT EXISTS social_posts (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		campaign_id TEXT NOT NULL,
		platform TEXT NOT NULL,
		content TEXT NOT NULL,
		image_url TEXT NOT NULL,
		target_url TEXT NOT NULL,
		publish_mode TEXT NOT NULL DEFAULT 'now',
		image_mode TEXT NOT NULL DEFAULT 'none',
		image_prompt TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		error TEXT NOT NULL DEFAULT '',
		facebook_post_id TEXT NOT NULL DEFAULT '',
		scheduled_at TIMESTAMP NULL,
		published_at TIMESTAMP NULL,
		created_at TIMESTAMP NOT NULL
	);`,

		`CREATE TABLE IF NOT EXISTS social_jobs (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		campaign_id TEXT NOT NULL DEFAULT '',
		post_id TEXT NOT NULL DEFAULT '',
		job_type TEXT NOT NULL,
		run_at TIMESTAMP NOT NULL,
		recurring_minutes INTEGER NOT NULL DEFAULT 0,
		days_of_week TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		last_error TEXT NOT NULL DEFAULT '',
		last_run_at TIMESTAMP NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS social_logs (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		campaign_id TEXT NOT NULL DEFAULT '',
		post_id TEXT NOT NULL DEFAULT '',
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS group_bots (
	id TEXT PRIMARY KEY,
	client_id TEXT NOT NULL,
	name TEXT NOT NULL,
	platform TEXT NOT NULL DEFAULT 'whatsapp',
	status TEXT NOT NULL DEFAULT 'draft',
	system_prompt TEXT NOT NULL DEFAULT '',
	business_name TEXT NOT NULL DEFAULT '',
	business_description TEXT NOT NULL DEFAULT '',
	offer TEXT NOT NULL DEFAULT '',
	target_audience TEXT NOT NULL DEFAULT '',
	rules TEXT NOT NULL DEFAULT '',
	welcome_message TEXT NOT NULL DEFAULT '',
	moderation_enabled INTEGER NOT NULL DEFAULT 1,
	auto_reply_enabled INTEGER NOT NULL DEFAULT 1,
	lead_capture_enabled INTEGER NOT NULL DEFAULT 1,
	human_handoff_phone TEXT NOT NULL DEFAULT '',
	group_jid TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL
);`,

	`CREATE TABLE IF NOT EXISTS facebook_group_targets (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT '',
		niche TEXT NOT NULL DEFAULT '',
		members_count INTEGER NOT NULL DEFAULT 0,
		relevance_score INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'discovered',
		join_status TEXT NOT NULL DEFAULT 'pending_manual_join',
		rules_summary TEXT NOT NULL DEFAULT '',
		notes TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,

		`CREATE TABLE IF NOT EXISTS group_growth_settings (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL UNIQUE,
		auto_join_enabled INTEGER NOT NULL DEFAULT 0,
		safe_mode INTEGER NOT NULL DEFAULT 1,
		max_joins_per_day INTEGER NOT NULL DEFAULT 2,
		max_total_groups INTEGER NOT NULL DEFAULT 50,
		min_delay_minutes INTEGER NOT NULL DEFAULT 120,
		max_delay_minutes INTEGER NOT NULL DEFAULT 360,
		allowed_hours TEXT NOT NULL DEFAULT '08:00-20:00',
		timezone TEXT NOT NULL DEFAULT 'America/Bogota',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS facebook_group_join_queue (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		group_target_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		scheduled_for TIMESTAMP NULL,
		executed_at TIMESTAMP NULL,
		attempts INTEGER NOT NULL DEFAULT 0,
		last_error TEXT NOT NULL DEFAULT '',
		notes TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS facebook_group_activity_logs (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		group_target_id TEXT NOT NULL,
		action_type TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT '',
		message TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL
	);`,

		`CREATE TABLE IF NOT EXISTS ads_campaigns (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		objective TEXT NOT NULL DEFAULT 'leads',
		product TEXT NOT NULL DEFAULT '',
		offer TEXT NOT NULL DEFAULT '',
		target_audience TEXT NOT NULL DEFAULT '',
		budget_daily REAL NOT NULL DEFAULT 0,
		budget_monthly REAL NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'draft',
		ai_plan_json TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
		);`,

		`CREATE TABLE IF NOT EXISTS assistant_messages (
			id TEXT PRIMARY KEY,
			client_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`,

		`CREATE INDEX IF NOT EXISTS idx_assistant_messages_client_id ON assistant_messages(client_id);`,

		`CREATE INDEX IF NOT EXISTS idx_social_jobs_status_run_at ON social_jobs(status, run_at);`,
		`CREATE INDEX IF NOT EXISTS idx_social_logs_client_id ON social_logs(client_id);`,
		`CREATE INDEX IF NOT EXISTS idx_social_posts_campaign_id ON social_posts(campaign_id);`,

		`CREATE INDEX IF NOT EXISTS idx_bots_client_id ON bots(client_id);`,
		`CREATE INDEX IF NOT EXISTS idx_leads_bot_id ON leads(bot_id);`,
		`CREATE INDEX IF NOT EXISTS idx_leads_chat_jid ON leads(chat_jid);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_bot_chat ON messages(bot_id, chat_jid);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_landing_pages_client_id ON landing_pages(client_id);`,
		`CREATE INDEX IF NOT EXISTS idx_landing_pages_bot_id ON landing_pages(bot_id);`,
		`CREATE INDEX IF NOT EXISTS idx_landing_pages_status ON landing_pages(status);`,
		`CREATE INDEX IF NOT EXISTS idx_funnel_events_client_id ON funnel_events(client_id);`,
		`CREATE INDEX IF NOT EXISTS idx_funnel_events_bot_id ON funnel_events(bot_id);`,
		`CREATE INDEX IF NOT EXISTS idx_funnel_events_landing_id ON funnel_events(landing_id);`,
		`CREATE INDEX IF NOT EXISTS idx_funnel_events_event_type ON funnel_events(event_type);`,
		`CREATE INDEX IF NOT EXISTS idx_funnel_events_created_at ON funnel_events(created_at);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migration error: %w", err)
		}
	}

	return seedTemplates(db)
}

func seedTemplates(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM templates WHERE is_default=1`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now()

	templates := []struct {
		ID, Name, Category, BusinessType, Stage, Prompt, Message string
	}{
		{
			"tpl-copy-1",
			"Copy Trading - Inicio",
			"sales",
			"trading",
			"new",
			"Habla como asesor comercial de copy trading. Responde breve y genera confianza sin prometer ganancias.",
			"Hola 👋 gracias por escribirnos. Te explico cómo funciona nuestro copy trading automatizado y vemos si encaja contigo. ¿Ya has invertido antes o sería tu primera vez?",
		},
		{
			"tpl-copy-2",
			"Copy Trading - Precio",
			"sales",
			"trading",
			"pricing",
			"Cuando pregunten precio, primero valida capital y experiencia.",
			"Claro 👌 manejamos planes según capital y objetivo. Para orientarte bien, ¿con cuánto te gustaría empezar y qué buscas lograr?",
		},
		{
			"tpl-realestate-1",
			"Inmobiliaria - Agenda",
			"sales",
			"real_estate",
			"new",
			"Eres asesor inmobiliario. Tu objetivo es agendar visita.",
			"Hola 👋 gracias por escribirnos. Te ayudo a encontrar la opción ideal. ¿Buscas para vivir o invertir, y en qué zona te interesa?",
		},
		{
			"tpl-ecommerce-1",
			"Ecommerce - Compra",
			"sales",
			"ecommerce",
			"new",
			"Eres asesor de ecommerce. Tu objetivo es resolver objeciones y cerrar compra.",
			"Hola 👋 gracias por escribirnos. Te ayudo con disponibilidad, precio y envío. ¿Qué producto te interesa exactamente?",
		},
	}

	for _, t := range templates {
		_, err := db.Exec(`
			INSERT INTO templates (
				id, client_id, name, category, business_type, stage,
				prompt_snippet, message_template, is_default, created_at, updated_at
			) VALUES (?, '', ?, ?, ?, ?, ?, ?, 1, ?, ?)
		`,
			t.ID, t.Name, t.Category, t.BusinessType, t.Stage, t.Prompt, t.Message, now, now,
		)
		if err != nil {
			return err
		}
	}

	return nil
}