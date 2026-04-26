package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AdsService struct {
	DB *sql.DB
	AI *AIService
}

func NewAdsService(db *sql.DB, ai *AIService) *AdsService {
	return &AdsService{DB: db, AI: ai}
}

type AdsCampaignPlan struct {
	Name              string   `json:"name"`
	Objective         string   `json:"objective"`
	Product           string   `json:"product"`
	Offer             string   `json:"offer"`
	TargetAudience    string   `json:"target_audience"`
	Locations         []string `json:"locations"`
	AgeRange          string   `json:"age_range"`
	Interests         []string `json:"interests"`
	PainPoints        []string `json:"pain_points"`
	Angles            []string `json:"angles"`
	PrimaryText       string   `json:"primary_text"`
	Headline          string   `json:"headline"`
	Description        string   `json:"description"`
	CTA               string   `json:"cta"`
	CreativePrompt     string   `json:"creative_prompt"`
	LandingSuggestion string   `json:"landing_suggestion"`
	WhatsAppScript    string   `json:"whatsapp_script"`
	BudgetDaily        float64  `json:"budget_daily"`
	BudgetMonthly      float64  `json:"budget_monthly"`
	EstimatedReach     int      `json:"estimated_reach"`
	EstimatedLeads     int      `json:"estimated_leads"`
	EstimatedCPL       float64  `json:"estimated_cpl"`
	EstimatedSales     int      `json:"estimated_sales"`
	EstimatedRevenue   float64  `json:"estimated_revenue"`
	EstimatedROI       float64  `json:"estimated_roi"`
	Recommendations    []string `json:"recommendations"`
}

func (s *AdsService) GenerateCampaignPlan(
	ctx context.Context,
	businessName string,
	product string,
	offer string,
	target string,
	country string,
	budgetDaily float64,
	ticketAverage float64,
) (AdsCampaignPlan, error) {
	businessName = strings.TrimSpace(businessName)
	product = strings.TrimSpace(product)
	offer = strings.TrimSpace(offer)
	target = strings.TrimSpace(target)
	country = strings.TrimSpace(country)

	if product == "" {
		return AdsCampaignPlan{}, fmt.Errorf("product required")
	}
	if offer == "" {
		return AdsCampaignPlan{}, fmt.Errorf("offer required")
	}
	if target == "" {
		return AdsCampaignPlan{}, fmt.Errorf("target required")
	}
	if country == "" {
		country = "Latinoamérica"
	}
	if budgetDaily <= 0 {
		budgetDaily = 10
	}
	if ticketAverage <= 0 {
		ticketAverage = 50
	}

	system := `Eres un experto senior en Meta Ads, funnels de venta, copywriting, segmentación, WhatsApp sales automation y ROI tracking.

Tu tarea es crear una campaña profesional para generar leads y ventas para un negocio real.

Debes responder SOLO JSON válido.
No uses markdown.
No uses texto fuera del JSON.
No uses comentarios.
No prometas resultados garantizados.

Criterios:
- El objetivo principal es llevar leads a WhatsApp, landing page o funnel.
- La segmentación debe parecer lista para configurar en Meta Ads.
- El copy debe ser persuasivo, humano y orientado a conversión.
- Las proyecciones deben ser estimadas, prudentes y realistas.
- Usa español.
- El JSON debe coincidir exactamente con las claves solicitadas.`

	user := fmt.Sprintf(`Crea una campaña publicitaria IA para:

Negocio: %s
Producto/servicio: %s
Oferta: %s
Público objetivo deseado: %s
País/ciudad: %s
Presupuesto diario USD: %.2f
Ticket promedio USD: %.2f

Devuelve únicamente este JSON:

{
  "name": "",
  "objective": "lead_generation",
  "product": "",
  "offer": "",
  "target_audience": "",
  "locations": [],
  "age_range": "",
  "interests": [],
  "pain_points": [],
  "angles": [],
  "primary_text": "",
  "headline": "",
  "description": "",
  "cta": "",
  "creative_prompt": "",
  "landing_suggestion": "",
  "whatsapp_script": "",
  "budget_daily": 0,
  "budget_monthly": 0,
  "estimated_reach": 0,
  "estimated_leads": 0,
  "estimated_cpl": 0,
  "estimated_sales": 0,
  "estimated_revenue": 0,
  "estimated_roi": 0,
  "recommendations": []
}`, businessName, product, offer, target, country, budgetDaily, ticketAverage)

	raw, err := s.AI.doHeavyCompletion(
		ctx,
		"",
		0.55,
		2600,
		[]map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	)
	if err != nil {
		return AdsCampaignPlan{}, err
	}

	raw = cleanAIJSON(raw)

	var plan AdsCampaignPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return AdsCampaignPlan{}, fmt.Errorf("respuesta IA inválida: %w | raw: %s", err, raw)
	}

	plan = normalizeCampaignPlan(plan, product, offer, target, country, budgetDaily, ticketAverage)

	return plan, nil
}

func cleanAIJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```JSON")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}

	return strings.TrimSpace(s)
}

func normalizeCampaignPlan(
	plan AdsCampaignPlan,
	product string,
	offer string,
	target string,
	country string,
	budgetDaily float64,
	ticketAverage float64,
) AdsCampaignPlan {
	if strings.TrimSpace(plan.Name) == "" {
		plan.Name = "Campaña IA - " + product
	}
	if strings.TrimSpace(plan.Objective) == "" {
		plan.Objective = "lead_generation"
	}
	if strings.TrimSpace(plan.Product) == "" {
		plan.Product = product
	}
	if strings.TrimSpace(plan.Offer) == "" {
		plan.Offer = offer
	}
	if strings.TrimSpace(plan.TargetAudience) == "" {
		plan.TargetAudience = target
	}
	if len(plan.Locations) == 0 {
		plan.Locations = []string{country}
	}
	if strings.TrimSpace(plan.AgeRange) == "" {
		plan.AgeRange = "25-55"
	}
	if strings.TrimSpace(plan.CTA) == "" {
		plan.CTA = "Enviar mensaje"
	}
	if plan.BudgetDaily <= 0 {
		plan.BudgetDaily = budgetDaily
	}
	if plan.BudgetMonthly <= 0 {
		plan.BudgetMonthly = budgetDaily * 30
	}

	if plan.EstimatedCPL <= 0 {
		plan.EstimatedCPL = 2.5
	}
	if plan.EstimatedLeads <= 0 {
		plan.EstimatedLeads = int(plan.BudgetMonthly / plan.EstimatedCPL)
	}
	if plan.EstimatedSales <= 0 {
		plan.EstimatedSales = int(float64(plan.EstimatedLeads) * 0.06)
	}
	if plan.EstimatedRevenue <= 0 {
		plan.EstimatedRevenue = float64(plan.EstimatedSales) * ticketAverage
	}
	if plan.EstimatedROI == 0 && plan.BudgetMonthly > 0 {
		plan.EstimatedROI = ((plan.EstimatedRevenue - plan.BudgetMonthly) / plan.BudgetMonthly) * 100
	}
	if len(plan.Recommendations) == 0 {
		plan.Recommendations = []string{
			"Probar mínimo 3 ángulos creativos durante los primeros 7 días.",
			"Enviar los leads a WhatsApp con respuesta automática inmediata.",
			"Medir costo por lead, tasa de respuesta y ventas cerradas antes de escalar presupuesto.",
		}
	}

	return plan
}

func (s *AdsService) SaveCampaign(clientID string, plan AdsCampaignPlan, rawJSON string) (string, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return "", fmt.Errorf("client_id required")
	}

	id := uuid.NewString()
	now := time.Now()

	if rawJSON == "" {
		b, _ := json.Marshal(plan)
		rawJSON = string(b)
	}

	_, err := s.DB.Exec(`
		INSERT INTO ads_campaigns (
			id, client_id, name, objective, product, offer, target_audience,
			budget_daily, budget_monthly, status, ai_plan_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		id,
		clientID,
		plan.Name,
		plan.Objective,
		plan.Product,
		plan.Offer,
		plan.TargetAudience,
		plan.BudgetDaily,
		plan.BudgetMonthly,
		"draft",
		rawJSON,
		now,
		now,
	)

	return id, err
}