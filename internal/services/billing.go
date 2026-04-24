package services

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type BillingService struct {
	DB *sql.DB
}

func NewBillingService(db *sql.DB) *BillingService {
	return &BillingService{DB: db}
}

func (b *BillingService) SeedDefaults() error {
	now := time.Now()

	defaultPlans := []models.Plan{
		{
			ID:           uuid.NewString(),
			Name:         "Free",
			Slug:         "free",
			Description:  "Ideal para probar la plataforma.",
			PriceMonthly: 0,
			PriceYearly:  0,
			Features:     `["1 bot WhatsApp","1 landing page","1 embudo básico","100 leads activos","IA limitada","Plantillas básicas","Branding Worktic","Soporte básico"]`,
			IsFree:       true,
			IsActive:     true,
			SortOrder:    1,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.NewString(),
			Name:         "Starter",
			Slug:         "starter",
			Description:  "Para negocios que están empezando.",
			PriceMonthly: 29,
			PriceYearly:  290,
			Features:     `["1 bot WhatsApp","3 landing pages","1 funnel completo","IA de atención y ventas","Publicaciones básicas","Métricas básicas","1 usuario"]`,
			IsFree:       false,
			IsActive:     true,
			SortOrder:    2,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.NewString(),
			Name:         "Pro",
			Slug:         "pro",
			Description:  "El plan principal para marketers y pymes.",
			PriceMonthly: 79,
			PriceYearly:  790,
			Features:     `["Hasta 5 bots","Funnels ilimitados","Landing pages ilimitadas","IA avanzada","Automatización inteligente","Programación de contenido","Métricas de leads y conversiones","Hasta 3 usuarios"]`,
			IsFree:       false,
			IsActive:     true,
			SortOrder:    3,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.NewString(),
			Name:         "Business",
			Slug:         "business",
			Description:  "Para agencias y empresas serias.",
			PriceMonthly: 149,
			PriceYearly:  1490,
			Features:     `["Bots altos o ilimitados","CRM/funnel avanzado","Administrador de grupos","Anuncios IA avanzados","Automatizaciones de seguimiento","Branding personalizado","Soporte prioritario"]`,
			IsFree:       false,
			IsActive:     true,
			SortOrder:    4,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.NewString(),
			Name:         "Elite / White Label",
			Slug:         "elite",
			Description:  "Para clientes enterprise y white label.",
			PriceMonthly: 399,
			PriceYearly:  3990,
			Features:     `["White label","Dominio personalizado","Subdominios por cliente","Multiempresa completa","Panel admin avanzado","IA entrenable","Soporte VIP"]`,
			IsFree:       false,
			IsActive:     true,
			SortOrder:    5,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	for _, p := range defaultPlans {
		_, err := b.DB.Exec(`
			INSERT OR IGNORE INTO plans
			(id, name, slug, description, price_monthly, price_yearly, features, is_free, is_active, sort_order, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, p.ID, p.Name, p.Slug, p.Description, p.PriceMonthly, p.PriceYearly, p.Features, boolToInt(p.IsFree), boolToInt(p.IsActive), p.SortOrder, p.CreatedAt, p.UpdatedAt)
		if err != nil {
			return err
		}
	}

	_, err := b.DB.Exec(`
		INSERT OR IGNORE INTO plan_config
		(id, usdt_bep20_wallet, card_payments_enabled, default_free_plan_slug, require_plan_selection, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "main", "", 0, "free", 1, now)

	return err
}

func (b *BillingService) ListPlans() ([]models.Plan, error) {
	rows, err := b.DB.Query(`
		SELECT id, name, slug, description, price_monthly, price_yearly, features, is_free, is_active, sort_order, created_at, updated_at
		FROM plans
		WHERE is_active=1
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Plan{}
	for rows.Next() {
		var p models.Plan
		var isFree, isActive int
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.PriceMonthly, &p.PriceYearly, &p.Features, &isFree, &isActive, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.IsFree = isFree == 1
		p.IsActive = isActive == 1
		out = append(out, p)
	}
	return out, nil
}

func (b *BillingService) GetPlanBySlug(slug string) (models.Plan, error) {
	var p models.Plan
	var isFree, isActive int
	err := b.DB.QueryRow(`
		SELECT id, name, slug, description, price_monthly, price_yearly, features, is_free, is_active, sort_order, created_at, updated_at
		FROM plans WHERE slug=?
	`, slug).Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.PriceMonthly, &p.PriceYearly, &p.Features, &isFree, &isActive, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	p.IsFree = isFree == 1
	p.IsActive = isActive == 1
	return p, err
}

func (b *BillingService) GetPlanConfig() (models.PlanConfig, error) {
	var c models.PlanConfig
	var cardEnabled, requireSelection int
	err := b.DB.QueryRow(`
		SELECT id, usdt_bep20_wallet, card_payments_enabled, default_free_plan_slug, require_plan_selection, updated_at
		FROM plan_config WHERE id='main'
	`).Scan(&c.ID, &c.USDTBEP20Wallet, &cardEnabled, &c.DefaultFreePlanSlug, &requireSelection, &c.UpdatedAt)
	c.CardPaymentsEnabled = cardEnabled == 1
	c.RequirePlanSelection = requireSelection == 1
	return c, err
}

func (b *BillingService) UpdatePlanConfig(c models.PlanConfig) error {
	_, err := b.DB.Exec(`
		UPDATE plan_config
		SET usdt_bep20_wallet=?, card_payments_enabled=?, default_free_plan_slug=?, require_plan_selection=?, updated_at=?
		WHERE id='main'
	`, c.USDTBEP20Wallet, boolToInt(c.CardPaymentsEnabled), c.DefaultFreePlanSlug, boolToInt(c.RequirePlanSelection), time.Now())
	return err
}

func (b *BillingService) GetLatestSubscription(clientID string) (models.Subscription, error) {
	var s models.Subscription
	err := b.DB.QueryRow(`
		SELECT id, client_id, plan_id, plan_slug, status, billing_cycle, amount, payment_method, tx_hash, wallet_address,
		       paid_at, starts_at, expires_at, validated_by, validation_notes, created_at, updated_at
		FROM subscriptions
		WHERE client_id=?
		ORDER BY created_at DESC
		LIMIT 1
	`, clientID).Scan(
		&s.ID, &s.ClientID, &s.PlanID, &s.PlanSlug, &s.Status, &s.BillingCycle, &s.Amount, &s.PaymentMethod, &s.TxHash, &s.WalletAddress,
		&s.PaidAt, &s.StartsAt, &s.ExpiresAt, &s.ValidatedBy, &s.ValidationNotes, &s.CreatedAt, &s.UpdatedAt,
	)
	return s, err
}

func (b *BillingService) SelectPlan(clientID, planSlug, billingCycle string) (models.Subscription, error) {
	plan, err := b.GetPlanBySlug(planSlug)
	if err != nil {
		return models.Subscription{}, err
	}

	now := time.Now()
	amount := plan.PriceMonthly
	if billingCycle == "yearly" {
		amount = plan.PriceYearly
	}

	status := "pending"
	var startsAt *time.Time
	var expiresAt *time.Time

	if plan.IsFree {
		status = "active"
		s := now
		e := now.AddDate(50, 0, 0)
		startsAt = &s
		expiresAt = &e
	}

	sub := models.Subscription{
		ID:            uuid.NewString(),
		ClientID:      clientID,
		PlanID:        plan.ID,
		PlanSlug:      plan.Slug,
		Status:        status,
		BillingCycle:  billingCycle,
		Amount:        amount,
		PaymentMethod: "usdt_bep20",
		CreatedAt:     now,
		UpdatedAt:     now,
		StartsAt:      startsAt,
		ExpiresAt:     expiresAt,
	}

	cfg, _ := b.GetPlanConfig()
	sub.WalletAddress = cfg.USDTBEP20Wallet

	_, err = b.DB.Exec(`
		INSERT INTO subscriptions
		(id, client_id, plan_id, plan_slug, status, billing_cycle, amount, payment_method, tx_hash, wallet_address,
		 paid_at, starts_at, expires_at, validated_by, validation_notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sub.ID, sub.ClientID, sub.PlanID, sub.PlanSlug, sub.Status, sub.BillingCycle, sub.Amount, sub.PaymentMethod, sub.TxHash, sub.WalletAddress,
		sub.PaidAt, sub.StartsAt, sub.ExpiresAt, sub.ValidatedBy, sub.ValidationNotes, sub.CreatedAt, sub.UpdatedAt)
	if err != nil {
		return models.Subscription{}, err
	}

	if plan.IsFree {
		_, err = b.DB.Exec(`UPDATE clients SET plan=?, updated_at=? WHERE id=?`, plan.Slug, now, clientID)
		if err != nil {
			return models.Subscription{}, err
		}
	}

	return sub, nil
}

func (b *BillingService) SubmitTxHash(subscriptionID, txHash string) error {
	_, err := b.DB.Exec(`
		UPDATE subscriptions
		SET tx_hash=?, status='pending', paid_at=?, updated_at=?
		WHERE id=?
	`, txHash, time.Now(), time.Now(), subscriptionID)
	return err
}

func (b *BillingService) ApproveSubscription(subscriptionID, adminUserID, notes string) error {
	sub, err := b.getSubscriptionByID(subscriptionID)
	if err != nil {
		return err
	}

	now := time.Now()
	expires := now.AddDate(0, 1, 0)
	if sub.BillingCycle == "yearly" {
		expires = now.AddDate(1, 0, 0)
	}

	_, err = b.DB.Exec(`
		UPDATE subscriptions
		SET status='active', starts_at=?, expires_at=?, validated_by=?, validation_notes=?, updated_at=?
		WHERE id=?
	`, now, expires, adminUserID, notes, now, subscriptionID)
	if err != nil {
		return err
	}

	_, err = b.DB.Exec(`UPDATE clients SET plan=?, updated_at=? WHERE id=?`, sub.PlanSlug, now, sub.ClientID)
	return err
}

func (b *BillingService) getSubscriptionByID(id string) (models.Subscription, error) {
	var s models.Subscription
	err := b.DB.QueryRow(`
		SELECT id, client_id, plan_id, plan_slug, status, billing_cycle, amount, payment_method, tx_hash, wallet_address,
		       paid_at, starts_at, expires_at, validated_by, validation_notes, created_at, updated_at
		FROM subscriptions
		WHERE id=?
	`, id).Scan(
		&s.ID, &s.ClientID, &s.PlanID, &s.PlanSlug, &s.Status, &s.BillingCycle, &s.Amount, &s.PaymentMethod, &s.TxHash, &s.WalletAddress,
		&s.PaidAt, &s.StartsAt, &s.ExpiresAt, &s.ValidatedBy, &s.ValidationNotes, &s.CreatedAt, &s.UpdatedAt,
	)
	return s, err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func EncodeFeatures(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}