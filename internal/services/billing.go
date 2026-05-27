package services

import (
	"database/sql"
	"encoding/json"
	"time"
	"errors"
	"strings"
	"fmt"

	"github.com/google/uuid"
	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type BillingService struct {
	DB *sql.DB
}

type PlanLimits map[string]int
type PlanPermissions map[string]bool

func NewBillingService(db *sql.DB) *BillingService {
	return &BillingService{DB: db}
}

func (b *BillingService) GetClientPlan(clientID string) (models.Plan, error) {
	var p models.Plan
	var isFree, isActive int

	var slug string

	err := b.DB.QueryRow(`
		SELECT COALESCE(plan, 'free')
		FROM clients
		WHERE id=?
	`, clientID).Scan(&slug)

	if err != nil {
		return p, err
	}

	err = b.DB.QueryRow(`
		SELECT
			id,
			name,
			slug,
			description,
			price_monthly,
			price_yearly,
			features,
			permissions,
			limits,
			grace_days,
			is_free,
			is_active,
			sort_order,
			created_at,
			updated_at
		FROM plans
		WHERE slug=?
		LIMIT 1
	`, slug).Scan(
		&p.ID,
		&p.Name,
		&p.Slug,
		&p.Description,
		&p.PriceMonthly,
		&p.PriceYearly,
		&p.Features,
		&p.Permissions,
		&p.Limits,
		&p.GraceDays,
		&isFree,
		&isActive,
		&p.SortOrder,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
   
	p.IsFree = isFree == 1
    p.IsActive = isActive == 1
	return p, err
}

func parsePlanLimits(raw string) PlanLimits {
	out := PlanLimits{}

	_ = json.Unmarshal([]byte(raw), &out)

	return out
}

func parsePlanPermissions(raw string) PlanPermissions {
	out := PlanPermissions{}

	_ = json.Unmarshal([]byte(raw), &out)

	return out
}

func (b *BillingService) HasFeature(
	role string,
	clientID string,
	feature string,
) bool {

	if strings.TrimSpace(role) == "admin" {
		return true
	}

	plan, err := b.GetClientPlan(clientID)
	if err != nil {
		return false
	}

	perms := parsePlanPermissions(plan.Permissions)

	return perms[feature]
}

func (b *BillingService) GetLimit(
	clientID string,
	metric string,
) int {

	plan, err := b.GetClientPlan(clientID)
	if err != nil {
		return 0
	}

	limits := parsePlanLimits(plan.Limits)

	return limits[metric]
}

func (b *BillingService) GetMonthlyUsage(
	clientID string,
	metric string,
) (int, error) {

	period := time.Now().Format("2006-01")

	var used int

	err := b.DB.QueryRow(`
		SELECT COALESCE(used,0)
		FROM usage_stats
		WHERE client_id=?
		AND metric=?
		AND period=?
		LIMIT 1
	`,
		clientID,
		metric,
		period,
	).Scan(&used)

	if err == sql.ErrNoRows {
		return 0, nil
	}

	return used, err
}

func (b *BillingService) IncrementUsage(
	clientID string,
	metric string,
	amount int,
) error {

	period := time.Now().Format("2006-01")
	now := time.Now()

	_, err := b.DB.Exec(`
		INSERT INTO usage_stats (
			id,
			client_id,
			metric,
			used,
			period,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_id, metric, period)
		DO UPDATE SET
			used = used + excluded.used,
			updated_at = excluded.updated_at
	`,
		uuid.NewString(),
		clientID,
		metric,
		amount,
		period,
		now,
		now,
	)

	return err
}	

func (b *BillingService) CheckLimit(
	role string,
	clientID string,
	metric string,
) error {

	if role == "admin" {
		return nil
	}

    if strings.TrimSpace(clientID) == "" {
	return errors.New("client id required for plan validation")
	}
	limit := b.GetLimit(clientID, metric)

	if limit <= 0 {
		return errors.New("feature not available")
	}

	used, err := b.GetMonthlyUsage(clientID, metric)
	if err != nil {
		return err
	}

	if used >= limit {
		return fmt.Errorf(
			"monthly limit reached for %s",
			metric,
		)
	}

	return nil
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

		Features: `[
			"1 bot WhatsApp",
			"1 landing page",
			"1 funnel básico",
			"100 leads activos",
			"IA limitada",
			"Plantillas básicas",
			"Branding Worktic",
			"Soporte básico"
		]`,

		Permissions: `{
			"whatsapp_ai": true,
			"landings": true,
			"funnels": true,
			"social_ai": true,
			"video_ai": false,
			"ads_ai": false,
			"groups_ai": false,
			"assistant_ai": false,
			"academy_ai": false,
			"marketplace": false,
			"agenda_ai": true
		}`,

		Limits: `{
			"bots": 1,
			"users": 1,
			"landing_pages": 1,
			"funnels": 1,
			"templates": 5,
			"social_posts_month": 20,
			"ai_images_month": 10,
			"ai_videos_month": 0,
			"ads_campaigns_month": 0,
			"group_bots": 0,
			"academy_courses": 0,
			"storage_mb": 500,
			"appointments_month": 0
		}`,

		GraceDays: 1,

		IsFree:    true,
		IsActive:  true,
		SortOrder: 1,
		CreatedAt: now,
		UpdatedAt: now,
	},

	{
		ID:           uuid.NewString(),
		Name:         "Starter",
		Slug:         "starter",
		Description:  "Para negocios que están empezando.",
		PriceMonthly: 17,
		PriceYearly:  157,

		Features: `[
			"1 bot WhatsApp",
			"3 landing pages",
			"1 funnel completo",
			"IA de atención y ventas",
			"Publicaciones básicas",
			"Métricas básicas",
			"1 usuario"
		]`,

		Permissions: `{
			"whatsapp_ai": true,
			"landings": true,
			"funnels": true,
			"social_ai": true,
			"video_ai": true,
			"ads_ai": false,
			"groups_ai": false,
			"assistant_ai": true,
			"academy_ai": false,
			"marketplace": true,
			"agenda_ai": true
		}`,

		Limits: `{
			"bots": 1,
			"users": 1,
			"landing_pages": 3,
			"funnels": 3,
			"templates": 20,
			"social_posts_month": 200,
			"ai_images_month": 100,
			"ai_videos_month": 20,
			"ads_campaigns_month": 5,
			"group_bots": 0,
			"academy_courses": 2,
			"storage_mb": 2048,
			"appointments_month": 10
		}`,

		GraceDays: 2,

		IsFree:    false,
		IsActive:  true,
		SortOrder: 2,
		CreatedAt: now,
		UpdatedAt: now,
	},

	{
		ID:           uuid.NewString(),
		Name:         "Pro",
		Slug:         "pro",
		Description:  "El plan principal para marketers y pymes.",
		PriceMonthly: 47,
		PriceYearly:  477,

		Features: `[
			"Hasta 5 bots",
			"Funnels ilimitados",
			"Landing pages ilimitadas",
			"IA avanzada",
			"Automatización inteligente",
			"Programación de contenido",
			"Métricas de leads y conversiones",
			"Hasta 3 usuarios"
		]`,

		Permissions: `{
			"whatsapp_ai": true,
			"landings": true,
			"funnels": true,
			"social_ai": true,
			"video_ai": true,
			"ads_ai": true,
			"groups_ai": true,
			"assistant_ai": true,
			"academy_ai": true,
			"marketplace": true,
			"agenda_ai": true
		}`,

		Limits: `{
			"bots": 5,
			"users": 3,
			"landing_pages": 100,
			"funnels": 50,
			"templates": 100,
			"social_posts_month": 5000,
			"ai_images_month": 1000,
			"ai_videos_month": 500,
			"ads_campaigns_month": 100,
			"group_bots": 10,
			"academy_courses": 20,
			"storage_mb": 10240,
			"appointments_month": 100
		}`,

		GraceDays: 3,

		IsFree:    false,
		IsActive:  true,
		SortOrder: 3,
		CreatedAt: now,
		UpdatedAt: now,
	},

	{
		ID:           uuid.NewString(),
		Name:         "Business",
		Slug:         "business",
		Description:  "Para agencias y empresas serias.",
		PriceMonthly: 97,
		PriceYearly:  897,

		Features: `[
			"Bots altos o ilimitados",
			"CRM/funnel avanzado",
			"Administrador de grupos",
			"Anuncios IA avanzados",
			"Automatizaciones de seguimiento",
			"Branding personalizado",
			"Soporte prioritario"
		]`,

		Permissions: `{
			"whatsapp_ai": true,
			"landings": true,
			"funnels": true,
			"social_ai": true,
			"video_ai": true,
			"ads_ai": true,
			"groups_ai": true,
			"assistant_ai": true,
			"academy_ai": true,
			"marketplace": true,
			"white_label": true,
			"agenda_ai": true
		}`,

		Limits: `{
			"bots": 999,
			"users": 999,
			"landing_pages": 999,
			"funnels": 999,
			"templates": 999,
			"social_posts_month": 999999,
			"ai_images_month": 999999,
			"ai_videos_month": 999999,
			"ads_campaigns_month": 999999,
			"group_bots": 999,
			"academy_courses": 999,
			"storage_mb": 102400
			"appointments_month": 1000
		}`,

		GraceDays: 5,

		IsFree:    false,
		IsActive:  true,
		SortOrder: 4,
		CreatedAt: now,
		UpdatedAt: now,
	},
}

	for _, p := range defaultPlans {

	if p.Features == "" {
		p.Features = "[]"
	}

	if p.Permissions == "" {
		p.Permissions = "{}"
	}

	if p.Limits == "" {
		p.Limits = "{}"
	}

	if p.GraceDays <= 0 {
		p.GraceDays = 1
	}

	_, err := b.DB.Exec(`
		INSERT OR IGNORE INTO plans (
			id,
			name,
			slug,
			description,
			price_monthly,
			price_yearly,
			features,
			permissions,
			limits,
			grace_days,
			is_free,
			is_active,
			sort_order,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		p.ID,
		p.Name,
		p.Slug,
		p.Description,
		p.PriceMonthly,
		p.PriceYearly,
		p.Features,
		p.Permissions,
		p.Limits,
		p.GraceDays,
		boolToInt(p.IsFree),
		boolToInt(p.IsActive),
		p.SortOrder,
		p.CreatedAt,
		p.UpdatedAt,
	)

	if err != nil {
		return err
	}
}

_, err := b.DB.Exec(`
	INSERT OR IGNORE INTO plan_config (
		id,
		usdt_bep20_wallet,
		card_payments_enabled,
		default_free_plan_slug,
		require_plan_selection,
		updated_at
	)
	VALUES (?, ?, ?, ?, ?, ?)
`,
	"main",
	"",
	0,
	"free",
	1,
	now,
)

return err
}

func (b *BillingService) ListPlans() ([]models.Plan, error) {
	rows, err := b.DB.Query(`
		SELECT id, name, slug, description, price_monthly, price_yearly, features, permissions, limits, grace_days, is_free, is_active, sort_order, created_at, updated_at
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
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Slug,
			&p.Description,
			&p.PriceMonthly,
			&p.PriceYearly,
			&p.Features,
			&p.Permissions,
			&p.Limits,
			&p.GraceDays,
			&isFree,
			&isActive,
			&p.SortOrder,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
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
		SELECT
			id,
			name,
			slug,
			description,
			price_monthly,
			price_yearly,
			features,
			permissions,
			limits,
			grace_days,
			is_free,
			is_active,
			sort_order,
			created_at,
			updated_at
		FROM plans
		WHERE slug=?
	`, slug).Scan(
		&p.ID,
		&p.Name,
		&p.Slug,
		&p.Description,
		&p.PriceMonthly,
		&p.PriceYearly,
		&p.Features,
		&p.Permissions,
		&p.Limits,
		&p.GraceDays,
		&isFree,
		&isActive,
		&p.SortOrder,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

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

func (b *BillingService) CreatePlan(p models.Plan) (models.Plan, error) {
	now := time.Now()

	if p.ID == "" {
		p.ID = uuid.NewString()
	}

	if p.Features == "" {
		p.Features = "[]"
	}

	if p.Permissions == "" {
		p.Permissions = "{}"
	}

	if p.Limits == "" {
		p.Limits = "{}"
	}

	if p.GraceDays <= 0 {
		p.GraceDays = 1
	}

	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := b.DB.Exec(`
		INSERT INTO plans (
			id, name, slug, description, price_monthly, price_yearly,
			features, permissions, limits, grace_days,
			is_free, is_active, sort_order, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		p.ID,
		p.Name,
		p.Slug,
		p.Description,
		p.PriceMonthly,
		p.PriceYearly,
		p.Features,
		p.Permissions,
		p.Limits,
		p.GraceDays,
		boolToInt(p.IsFree),
		boolToInt(p.IsActive),
		p.SortOrder,
		p.CreatedAt,
		p.UpdatedAt,
	)

	return p, err
}

func (b *BillingService) UpdatePlan(p models.Plan) error {
	if p.Features == "" {
		p.Features = "[]"
	}

	if p.Permissions == "" {
		p.Permissions = "{}"
	}

	if p.Limits == "" {
		p.Limits = "{}"
	}

	if p.GraceDays <= 0 {
		p.GraceDays = 1
	}

	_, err := b.DB.Exec(`
		UPDATE plans
		SET name=?, slug=?, description=?, price_monthly=?, price_yearly=?,
			features=?, permissions=?, limits=?, grace_days=?,
			is_free=?, is_active=?, sort_order=?, updated_at=?
		WHERE id=?
	`,
		p.Name,
		p.Slug,
		p.Description,
		p.PriceMonthly,
		p.PriceYearly,
		p.Features,
		p.Permissions,
		p.Limits,
		p.GraceDays,
		boolToInt(p.IsFree),
		boolToInt(p.IsActive),
		p.SortOrder,
		time.Now(),
		p.ID,
	)

	return err
}

func (b *BillingService) DeletePlan(id string) error {
	_, err := b.DB.Exec(`UPDATE plans SET is_active=0, updated_at=? WHERE id=?`, time.Now(), id)
	return err
}