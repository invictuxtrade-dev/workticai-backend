package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
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

type AdsAdSet struct {
	Name       string   `json:"name"`
	Location   []string `json:"locations"`
	AgeRange   string   `json:"age_range"`
	Gender     string   `json:"gender"`
	Interests  []string `json:"interests"`
	Behaviors  []string `json:"behaviors"`
	Exclusions []string `json:"exclusions"`
	Message    string   `json:"message"`
}

type AdsCreativeVariant struct {
	Name           string `json:"name"`
	Angle          string `json:"angle"`
	PrimaryText    string `json:"primary_text"`
	Headline       string `json:"headline"`
	Description    string `json:"description"`
	CTA            string `json:"cta"`
	CreativePrompt string `json:"creative_prompt"`
}

type AdsFunnelStrategy struct {
	Destination        string   `json:"destination"`
	RecommendedBotFlow string   `json:"recommended_bot_flow"`
	LandingStructure   []string `json:"landing_structure"`
	LeadQualification  []string `json:"lead_qualification"`
	FollowUpSequence   []string `json:"follow_up_sequence"`
	TrackingEvents     []string `json:"tracking_events"`
}

type AdsROIProjection struct {
	Currency         string  `json:"currency"`
	BudgetDaily      float64 `json:"budget_daily"`
	BudgetMonthly    float64 `json:"budget_monthly"`
	EstimatedCPM     float64 `json:"estimated_cpm"`
	EstimatedCPC     float64 `json:"estimated_cpc"`
	EstimatedCTR     float64 `json:"estimated_ctr"`
	EstimatedCPL     float64 `json:"estimated_cpl"`
	EstimatedReach   int     `json:"estimated_reach"`
	EstimatedClicks  int     `json:"estimated_clicks"`
	EstimatedLeads   int     `json:"estimated_leads"`
	ConversionRate   float64 `json:"conversion_rate"`
	EstimatedSales   int     `json:"estimated_sales"`
	TicketAverage     float64 `json:"ticket_average"`
	EstimatedRevenue float64 `json:"estimated_revenue"`
	EstimatedProfit  float64 `json:"estimated_profit"`
	EstimatedROI     float64 `json:"estimated_roi"`
	BreakEvenCPL     float64 `json:"break_even_cpl"`
}

type AdsROIScenario struct {
	Name                string  `json:"name"`
	Currency            string  `json:"currency"`
	BudgetDaily         float64 `json:"budget_daily"`
	BudgetMonthly       float64 `json:"budget_monthly"`
	EstimatedCPM        float64 `json:"estimated_cpm"`
	EstimatedCTR        float64 `json:"estimated_ctr"`
	EstimatedCPC        float64 `json:"estimated_cpc"`
	LandingConvRate     float64 `json:"landing_conversion_rate"`
	LeadCloseRate       float64 `json:"lead_close_rate"`
	EstimatedReach      int     `json:"estimated_reach"`
	EstimatedClicks     int     `json:"estimated_clicks"`
	EstimatedLeads      int     `json:"estimated_leads"`
	EstimatedCPL        float64 `json:"estimated_cpl"`
	EstimatedSales      int     `json:"estimated_sales"`
	TicketAverage        float64 `json:"ticket_average"`
	EstimatedRevenue    float64 `json:"estimated_revenue"`
	EstimatedProfit     float64 `json:"estimated_profit"`
	EstimatedROI        float64 `json:"estimated_roi"`
	BreakEvenCPL        float64 `json:"break_even_cpl"`
	Recommendation      string  `json:"recommendation"`
	Decision            string  `json:"decision"`
	ScaleSignal         string  `json:"scale_signal"`
	OptimizationTrigger string  `json:"optimization_trigger"`
}

type AdsCampaignPlan struct {
	Name              string   `json:"name"`
	Objective         string   `json:"objective"`
	Currency          string   `json:"currency"`
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

	BudgetDaily      float64 `json:"budget_daily"`
	BudgetMonthly    float64 `json:"budget_monthly"`
	EstimatedReach   int     `json:"estimated_reach"`
	EstimatedLeads   int     `json:"estimated_leads"`
	EstimatedCPL     float64 `json:"estimated_cpl"`
	EstimatedSales   int     `json:"estimated_sales"`
	EstimatedRevenue float64 `json:"estimated_revenue"`
	EstimatedROI     float64 `json:"estimated_roi"`

	Recommendations []string `json:"recommendations"`

	CampaignSummary  string               `json:"campaign_summary"`
	MarketAnalysis   string               `json:"market_analysis"`
	CustomerAvatar   string               `json:"customer_avatar"`
	ValueProposition string               `json:"value_proposition"`
	AdSets           []AdsAdSet           `json:"adsets"`
	CreativeVariants []AdsCreativeVariant `json:"creative_variants"`
	Funnel           AdsFunnelStrategy    `json:"funnel"`
	ROI              AdsROIProjection     `json:"roi"`
	ROIScenarios     []AdsROIScenario     `json:"roi_scenarios"`

	OptimizationPlan []string `json:"optimization_plan"`
	LaunchChecklist  []string `json:"launch_checklist"`
	TestingPlan      []string `json:"testing_plan"`
	RiskWarnings     []string `json:"risk_warnings"`
	NextActions      []string `json:"next_actions"`

	AutomationRules []string `json:"automation_rules"`
	ScaleRules      []string `json:"scale_rules"`
	KillRules       []string `json:"kill_rules"`
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

	currency := "USD"

	system := `Eres un experto ELITE en Meta Ads, funnels, psicología de ventas, WhatsApp automation, copywriting de respuesta directa y estrategia de adquisición.

Tu trabajo es crear una campaña publicitaria profesional para generar leads y ventas.

IMPORTANTE:
- La IA NO debe inventar métricas financieras finales.
- No prometas resultados garantizados.
- Las métricas finales serán calculadas por el backend.
- Enfócate en estrategia, segmentación, creativos, funnel, mensajes y optimización.
- Responde SOLO JSON válido.
- No uses markdown.
- No uses texto fuera del JSON.
- Usa español profesional y accionable.`

	user := fmt.Sprintf(`Crea un plan PRO de campaña Ads IA para:

Negocio: %s
Producto/servicio: %s
Oferta: %s
Público objetivo deseado: %s
Ubicación/mercado: %s
Moneda: %s
Presupuesto diario: %.2f
Ticket promedio: %.2f

Devuelve únicamente JSON con esta estructura exacta:

{
  "name": "",
  "objective": "lead_generation",
  "currency": "%s",
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
  "recommendations": [],

  "campaign_summary": "",
  "market_analysis": "",
  "customer_avatar": "",
  "value_proposition": "",

  "adsets": [
    {
      "name": "",
      "locations": [],
      "age_range": "",
      "gender": "all",
      "interests": [],
      "behaviors": [],
      "exclusions": [],
      "message": ""
    }
  ],

  "creative_variants": [
    {
      "name": "",
      "angle": "",
      "primary_text": "",
      "headline": "",
      "description": "",
      "cta": "",
      "creative_prompt": ""
    }
  ],

  "funnel": {
    "destination": "whatsapp",
    "recommended_bot_flow": "",
    "landing_structure": [],
    "lead_qualification": [],
    "follow_up_sequence": [],
    "tracking_events": []
  },

  "optimization_plan": [],
  "launch_checklist": [],
  "testing_plan": [],
  "risk_warnings": [],
  "next_actions": [],
  "automation_rules": [],
  "scale_rules": [],
  "kill_rules": []
}

Condiciones:
- Genera mínimo 3 adsets.
- Genera mínimo 4 creative_variants.
- Genera mínimo 5 recomendaciones.
- Genera mínimo 5 pasos de optimization_plan.
- Genera mínimo 5 puntos de launch_checklist.
- Genera mínimo 5 automation_rules.
- Genera mínimo 3 scale_rules.
- Genera mínimo 3 kill_rules.
- La segmentación debe parecer realista para Meta Ads.
- Los copies deben estar listos para anuncio.
- El creative_prompt debe servir para generar imagen con IA.
- El whatsapp_script debe servir como primer mensaje comercial o guion del bot.`,
		businessName,
		product,
		offer,
		target,
		country,
		currency,
		budgetDaily,
		ticketAverage,
		currency,
	)

	raw, err := s.AI.doHeavyCompletion(
		ctx,
		"",
		0.42,
		4500,
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
	fixed, repairErr := s.repairAIJSON(ctx, raw)
	if repairErr != nil {
		return AdsCampaignPlan{}, fmt.Errorf("respuesta IA inválida")
	}

	fixed = cleanAIJSON(fixed)

	if err2 := json.Unmarshal([]byte(fixed), &plan); err2 != nil {
		return AdsCampaignPlan{}, fmt.Errorf("respuesta IA inválida")
	}
}

func (s *AdsService) repairAIJSON(ctx context.Context, broken string) (string, error) {
	system := `Eres un reparador experto de JSON.
Debes recibir texto que intenta ser JSON y devolver SOLO JSON válido.
No expliques nada.
No uses markdown.
No agregues campos nuevos.
Solo corrige comillas, comas, escapes y estructura.`

	user := fmt.Sprintf(`Repara este JSON y devuelve únicamente JSON válido:

%s`, broken)

	return s.AI.doHeavyCompletion(
		ctx,
		"",
		0.1,
		4500,
		[]map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	)
}


	plan = normalizeCampaignPlan(plan, product, offer, target, country, currency, budgetDaily, ticketAverage)

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
	currency string,
	budgetDaily float64,
	ticketAverage float64,
) AdsCampaignPlan {
	product = strings.TrimSpace(product)
	offer = strings.TrimSpace(offer)
	target = strings.TrimSpace(target)
	country = strings.TrimSpace(country)
	currency = strings.TrimSpace(currency)

	if currency == "" {
		currency = "USD"
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

	if strings.TrimSpace(plan.Name) == "" {
		plan.Name = "Campaña IA - " + product
	}
	if strings.TrimSpace(plan.Objective) == "" {
		plan.Objective = "lead_generation"
	}
	if strings.TrimSpace(plan.Currency) == "" {
		plan.Currency = currency
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
	if len(plan.Interests) == 0 {
		plan.Interests = []string{"Compras online", "Educación", "Familia", "Bienestar", "Emprendimiento"}
	}
	if len(plan.PainPoints) == 0 {
		plan.PainPoints = []string{
			"No sabe qué solución elegir",
			"Tiene poco tiempo para investigar",
			"Necesita confianza antes de comprar",
		}
	}
	if len(plan.Angles) == 0 {
		plan.Angles = []string{
			"Beneficio directo",
			"Dolor + solución",
			"Confianza",
			"Oferta directa",
		}
	}
	if strings.TrimSpace(plan.PrimaryText) == "" {
		plan.PrimaryText = fmt.Sprintf("Descubre cómo %s puede ayudarte con %s. Escríbenos y recibe más información.", product, offer)
	}
	if strings.TrimSpace(plan.Headline) == "" {
		plan.Headline = product + " para " + target
	}
	if strings.TrimSpace(plan.Description) == "" {
		plan.Description = "Conoce una solución pensada para tus necesidades."
	}
	if strings.TrimSpace(plan.CTA) == "" {
		plan.CTA = "Enviar mensaje"
	}
	if strings.TrimSpace(plan.CreativePrompt) == "" {
		plan.CreativePrompt = "Imagen publicitaria profesional del producto en uso, estilo moderno, alta conversión, sin exceso de texto."
	}
	if strings.TrimSpace(plan.LandingSuggestion) == "" {
		plan.LandingSuggestion = "Landing con promesa clara, beneficios, prueba social, explicación simple, CTA a WhatsApp y formulario."
	}
	if strings.TrimSpace(plan.WhatsAppScript) == "" {
		plan.WhatsAppScript = "Hola, gracias por escribirnos. Cuéntame qué estás buscando y te ayudo a elegir la mejor opción."
	}

	plan = normalizeROI(plan, budgetDaily, ticketAverage, currency)
	plan = normalizeStrategicBlocks(plan, product, offer, target, country)

	return plan
}

func normalizeStrategicBlocks(plan AdsCampaignPlan, product, offer, target, country string) AdsCampaignPlan {
	if len(plan.Recommendations) == 0 {
		plan.Recommendations = []string{
			"Probar mínimo 4 creativos durante los primeros 7 días.",
			"Separar audiencias por intención: fría, tibia y remarketing.",
			"Enviar todos los leads a WhatsApp con respuesta automática inmediata.",
			"Medir CPL, tasa de respuesta, tasa de cierre y ROAS antes de escalar.",
			"No aumentar presupuesto hasta identificar el mejor ángulo creativo.",
		}
	}

	if strings.TrimSpace(plan.CampaignSummary) == "" {
		plan.CampaignSummary = "Campaña enfocada en captación de leads y conversaciones comerciales usando anuncios de respuesta directa."
	}
	if strings.TrimSpace(plan.MarketAnalysis) == "" {
		plan.MarketAnalysis = "El mercado requiere una comunicación clara, orientada a beneficios y con confianza suficiente para mover al usuario hacia WhatsApp o landing."
	}
	if strings.TrimSpace(plan.CustomerAvatar) == "" {
		plan.CustomerAvatar = target
	}
	if strings.TrimSpace(plan.ValueProposition) == "" {
		plan.ValueProposition = offer
	}

	if len(plan.AdSets) == 0 {
		plan.AdSets = []AdsAdSet{
			{
				Name:       "Audiencia principal",
				Location:   []string{country},
				AgeRange:   plan.AgeRange,
				Gender:     "all",
				Interests:  plan.Interests,
				Behaviors:  []string{"Engaged shoppers", "Usuarios activos en redes sociales"},
				Exclusions: []string{},
				Message:    "Audiencia base alineada con el producto y oferta.",
			},
			{
				Name:       "Audiencia por dolor",
				Location:   []string{country},
				AgeRange:   plan.AgeRange,
				Gender:     "all",
				Interests:  plan.PainPoints,
				Behaviors:  []string{"Interacción con contenido educativo o comercial"},
				Exclusions: []string{},
				Message:    "Segmentación enfocada en problemas que el producto resuelve.",
			},
			{
				Name:       "Audiencia amplia controlada",
				Location:   []string{country},
				AgeRange:   plan.AgeRange,
				Gender:     "all",
				Interests:  []string{"Compras online", "Servicios profesionales", "Intereses relacionados"},
				Behaviors:  []string{"Engaged shoppers"},
				Exclusions: []string{},
				Message:    "Audiencia amplia para que el algoritmo encuentre patrones de conversión.",
			},
		}
	}

	if len(plan.CreativeVariants) == 0 {
		plan.CreativeVariants = []AdsCreativeVariant{
			{
				Name:           "Creativo principal",
				Angle:          firstOr(plan.Angles, "Beneficio principal"),
				PrimaryText:    plan.PrimaryText,
				Headline:       plan.Headline,
				Description:    plan.Description,
				CTA:            plan.CTA,
				CreativePrompt: plan.CreativePrompt,
			},
			{
				Name:           "Dolor + solución",
				Angle:          "Dolor + solución",
				PrimaryText:    fmt.Sprintf("¿Te pasa que necesitas %s pero no sabes por dónde empezar? %s puede ayudarte.", offer, product),
				Headline:       "Solución simple para avanzar",
				Description:    "Habla con nosotros y recibe orientación.",
				CTA:            "Enviar mensaje",
				CreativePrompt: plan.CreativePrompt,
			},
			{
				Name:           "Confianza",
				Angle:          "Autoridad y confianza",
				PrimaryText:    fmt.Sprintf("Conoce una alternativa clara y confiable para %s. Te ayudamos paso a paso.", target),
				Headline:       "Recibe asesoría personalizada",
				Description:    "Haz clic y escríbenos por WhatsApp.",
				CTA:            "Enviar mensaje",
				CreativePrompt: plan.CreativePrompt,
			},
			{
				Name:           "Oferta directa",
				Angle:          "Oferta directa",
				PrimaryText:    fmt.Sprintf("%s. Solicita información ahora y descubre si es ideal para ti.", offer),
				Headline:       "Disponible ahora",
				Description:    "Respuesta rápida por WhatsApp.",
				CTA:            "Más información",
				CreativePrompt: plan.CreativePrompt,
			},
		}
	}

	if strings.TrimSpace(plan.Funnel.Destination) == "" {
		plan.Funnel.Destination = "whatsapp"
	}
	if strings.TrimSpace(plan.Funnel.RecommendedBotFlow) == "" {
		plan.Funnel.RecommendedBotFlow = "Saludo inicial, detección de necesidad, explicación breve de la oferta, manejo de objeciones y CTA para cierre o asesoría."
	}
	if len(plan.Funnel.LandingStructure) == 0 {
		plan.Funnel.LandingStructure = []string{
			"Hero con promesa clara",
			"Beneficios principales",
			"Cómo funciona",
			"Prueba social o confianza",
			"Preguntas frecuentes",
			"CTA a WhatsApp",
		}
	}
	if len(plan.Funnel.LeadQualification) == 0 {
		plan.Funnel.LeadQualification = []string{
			"¿Qué necesidad quieres resolver?",
			"¿Cuándo te gustaría empezar?",
			"¿Cuál es tu presupuesto aproximado?",
			"¿Quieres recibir asesoría personalizada?",
		}
	}
	if len(plan.Funnel.FollowUpSequence) == 0 {
		plan.Funnel.FollowUpSequence = []string{
			"Día 0: respuesta inmediata",
			"Día 1: recordatorio de beneficio",
			"Día 2: prueba social o caso de uso",
			"Día 3: manejo de objeción",
			"Día 5: CTA final",
		}
	}
	if len(plan.Funnel.TrackingEvents) == 0 {
		plan.Funnel.TrackingEvents = []string{
			"ad_click",
			"landing_view",
			"whatsapp_click",
			"lead_created",
			"lead_qualified",
			"conversion",
		}
	}

	if len(plan.OptimizationPlan) == 0 {
		plan.OptimizationPlan = []string{
			"Revisar CPL, CTR y tasa de conversación después de 48 horas.",
			"Pausar creativos con CTR bajo y CPL alto.",
			"Duplicar el mejor creativo con nueva variación visual.",
			"Separar presupuesto hacia el adset con mayor tasa de conversación.",
			"Crear remarketing para usuarios que hicieron clic y no escribieron.",
		}
	}
	if len(plan.LaunchChecklist) == 0 {
		plan.LaunchChecklist = []string{
			"Confirmar que el bot de WhatsApp responde correctamente.",
			"Verificar que la landing o enlace de destino funcione.",
			"Validar que el pixel o tracking esté configurado.",
			"Revisar copy, imagen y CTA antes de publicar.",
			"Definir presupuesto mínimo de prueba por 5 a 7 días.",
		}
	}
	if len(plan.TestingPlan) == 0 {
		plan.TestingPlan = []string{
			"Test A: beneficio directo.",
			"Test B: dolor principal.",
			"Test C: prueba social.",
			"Test D: oferta directa.",
			"Comparar por CPL, CTR, CPC y tasa de respuesta en WhatsApp.",
		}
	}
	if len(plan.RiskWarnings) == 0 {
		plan.RiskWarnings = []string{
			"Las métricas son estimadas y dependen del mercado, creatividad, oferta y seguimiento.",
			"No escalar presupuesto sin datos suficientes.",
			"Un buen anuncio no compensa una mala oferta o mala atención de leads.",
		}
	}
	if len(plan.NextActions) == 0 {
		plan.NextActions = []string{
			"Elegir destino: WhatsApp o landing.",
			"Generar imagen del anuncio.",
			"Configurar bot de atención.",
			"Publicar campaña de prueba.",
			"Medir resultados y optimizar.",
		}
	}
	if len(plan.AutomationRules) == 0 {
		plan.AutomationRules = []string{
			"Si CTR < 0.8% después de 1,500 impresiones, crear nuevo hook y nuevo creativo.",
			"Si CPL supera el break-even CPL, pausar adset y ajustar oferta.",
			"Si WhatsApp responde en más de 5 minutos, priorizar automatización del bot.",
			"Si un creativo genera 2x más leads que los demás, duplicarlo con nuevo ángulo.",
			"Si hay clics sin leads, revisar landing, enlace o fricción del formulario.",
		}
	}
	if len(plan.ScaleRules) == 0 {
		plan.ScaleRules = []string{
			"Escalar 20% cada 48 horas si CPL se mantiene bajo el objetivo.",
			"Duplicar el adset ganador antes de aumentar presupuesto agresivamente.",
			"Crear remarketing cuando existan suficientes clics o visitas.",
		}
	}
	if len(plan.KillRules) == 0 {
		plan.KillRules = []string{
			"Pausar anuncio si CTR es bajo y no genera leads tras 48 horas.",
			"Pausar adset si CPL supera el break-even CPL por dos días seguidos.",
			"Pausar creativo si consume 1.5x el CPL objetivo sin resultados.",
		}
	}

	return plan
}

func normalizeROI(plan AdsCampaignPlan, budgetDaily float64, ticketAverage float64, currency string) AdsCampaignPlan {
	scenarios := buildROIScenarios(budgetDaily, ticketAverage, currency)

	realistic := scenarios[1]

	plan.BudgetDaily = realistic.BudgetDaily
	plan.BudgetMonthly = realistic.BudgetMonthly
	plan.EstimatedCPL = realistic.EstimatedCPL
	plan.EstimatedLeads = realistic.EstimatedLeads
	plan.EstimatedReach = realistic.EstimatedReach
	plan.EstimatedSales = realistic.EstimatedSales
	plan.EstimatedRevenue = realistic.EstimatedRevenue
	plan.EstimatedROI = realistic.EstimatedROI

	plan.ROI = AdsROIProjection{
		Currency:         realistic.Currency,
		BudgetDaily:      realistic.BudgetDaily,
		BudgetMonthly:    realistic.BudgetMonthly,
		EstimatedCPM:     realistic.EstimatedCPM,
		EstimatedCPC:     realistic.EstimatedCPC,
		EstimatedCTR:     realistic.EstimatedCTR,
		EstimatedCPL:     realistic.EstimatedCPL,
		EstimatedReach:   realistic.EstimatedReach,
		EstimatedClicks:  realistic.EstimatedClicks,
		EstimatedLeads:   realistic.EstimatedLeads,
		ConversionRate:   realistic.LeadCloseRate,
		EstimatedSales:   realistic.EstimatedSales,
		TicketAverage:    realistic.TicketAverage,
		EstimatedRevenue: realistic.EstimatedRevenue,
		EstimatedProfit:  realistic.EstimatedProfit,
		EstimatedROI:     realistic.EstimatedROI,
		BreakEvenCPL:     realistic.BreakEvenCPL,
	}

	plan.ROIScenarios = scenarios

	return plan
}

func buildROIScenarios(budgetDaily float64, ticketAverage float64, currency string) []AdsROIScenario {
	if budgetDaily <= 0 {
		budgetDaily = 10
	}
	if ticketAverage <= 0 {
		ticketAverage = 50
	}

	budgetMonthly := budgetDaily * 30

	return []AdsROIScenario{
		calcROIScenario(
			"Conservador",
			currency,
			budgetDaily,
			budgetMonthly,
			ticketAverage,
			7.5,
			0.75,
			7.0,
			2.5,
			"Validar oferta y creativos antes de escalar. Si el CPL supera el break-even, ajustar copy, oferta o segmentación.",
			"Probar sin escalar hasta tener mejores señales.",
			"Escalar solo si el CPL baja por debajo del punto de equilibrio.",
			"CTR bajo o CPL alto durante 48 horas.",
		),
		calcROIScenario(
			"Realista",
			currency,
			budgetDaily,
			budgetMonthly,
			ticketAverage,
			5.5,
			1.25,
			10.0,
			5.0,
			"Escenario base para tomar decisiones iniciales. Medir 5 a 7 días antes de escalar presupuesto.",
			"Lanzar prueba controlada con 3 adsets y 4 creativos.",
			"Escalar 15% a 20% si el CPL se mantiene estable.",
			"CPL superior al break-even CPL o baja respuesta en WhatsApp.",
		),
		calcROIScenario(
			"Agresivo",
			currency,
			budgetDaily,
			budgetMonthly,
			ticketAverage,
			4.0,
			2.0,
			16.0,
			8.0,
			"Solo escalar a este escenario si hay buen CTR, buen CPL, respuesta rápida y cierre comercial comprobado.",
			"Escalar creativos ganadores y activar remarketing.",
			"Duplicar adset ganador y aumentar presupuesto gradualmente.",
			"Caída de CTR, aumento de CPL o caída de tasa de cierre.",
		),
	}
}

func calcROIScenario(
	name string,
	currency string,
	budgetDaily float64,
	budgetMonthly float64,
	ticketAverage float64,
	cpm float64,
	ctrPercent float64,
	landingConvPercent float64,
	closeRatePercent float64,
	recommendation string,
	decision string,
	scaleSignal string,
	optimizationTrigger string,
) AdsROIScenario {
	reach := 0
	if cpm > 0 {
		reach = int(math.Round((budgetMonthly / cpm) * 1000))
	}

	clicks := int(math.Round(float64(reach) * (ctrPercent / 100)))

	cpc := 0.0
	if clicks > 0 {
		cpc = budgetMonthly / float64(clicks)
	}

	leads := int(math.Round(float64(clicks) * (landingConvPercent / 100)))

	cpl := 0.0
	if leads > 0 {
		cpl = budgetMonthly / float64(leads)
	}

	sales := int(math.Round(float64(leads) * (closeRatePercent / 100)))

	revenue := float64(sales) * ticketAverage
	profit := revenue - budgetMonthly

	roi := 0.0
	if budgetMonthly > 0 {
		roi = (profit / budgetMonthly) * 100
	}

	breakEvenCPL := 0.0
	if closeRatePercent > 0 {
		breakEvenCPL = ticketAverage * (closeRatePercent / 100)
	}

	return AdsROIScenario{
		Name:                name,
		Currency:            currency,
		BudgetDaily:         round2(budgetDaily),
		BudgetMonthly:       round2(budgetMonthly),
		EstimatedCPM:        round2(cpm),
		EstimatedCTR:        round2(ctrPercent),
		EstimatedCPC:        round2(cpc),
		LandingConvRate:     round2(landingConvPercent),
		LeadCloseRate:       round2(closeRatePercent),
		EstimatedReach:      reach,
		EstimatedClicks:     clicks,
		EstimatedLeads:      leads,
		EstimatedCPL:        round2(cpl),
		EstimatedSales:      sales,
		TicketAverage:       round2(ticketAverage),
		EstimatedRevenue:    round2(revenue),
		EstimatedProfit:     round2(profit),
		EstimatedROI:        round2(roi),
		BreakEvenCPL:        round2(breakEvenCPL),
		Recommendation:      recommendation,
		Decision:            decision,
		ScaleSignal:         scaleSignal,
		OptimizationTrigger: optimizationTrigger,
	}
}

func firstOr(items []string, fallback string) string {
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			return item
		}
	}
	return fallback
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
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