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

// ─────────────────────────────────────────────
// INDUSTRY BENCHMARK ENGINE
// ─────────────────────────────────────────────

// IndustryBenchmark contiene los rangos reales de métricas Meta Ads
// por industria. Fuente: datos promedio del sector 2023-2024.
type IndustryBenchmark struct {
	// CTR (%)
	MinCTR float64
	AvgCTR float64
	MaxCTR float64

	// CPC (USD)
	MinCPC float64
	AvgCPC float64
	MaxCPC float64

	// CPM (USD)
	MinCPM float64
	AvgCPM float64
	MaxCPM float64

	// Tasa conversión landing → lead (%)
	MinLandingConv float64
	AvgLandingConv float64
	MaxLandingConv float64

	// Tasa cierre lead → venta (%)
	MinCloseRate float64
	AvgCloseRate float64
	MaxCloseRate float64
}

// industryBenchmarks: mapa de industrias con benchmarks reales.
var industryBenchmarks = map[string]IndustryBenchmark{
	"wellness": {
		MinCTR: 1.2, AvgCTR: 2.4, MaxCTR: 4.0,
		MinCPC: 0.08, AvgCPC: 0.18, MaxCPC: 0.40,
		MinCPM: 3.0, AvgCPM: 5.5, MaxCPM: 9.0,
		MinLandingConv: 8, AvgLandingConv: 18, MaxLandingConv: 30,
		MinCloseRate: 5, AvgCloseRate: 12, MaxCloseRate: 22,
	},
	"salud": {
		MinCTR: 1.0, AvgCTR: 2.1, MaxCTR: 3.8,
		MinCPC: 0.10, AvgCPC: 0.25, MaxCPC: 0.60,
		MinCPM: 4.0, AvgCPM: 6.5, MaxCPM: 11.0,
		MinLandingConv: 7, AvgLandingConv: 16, MaxLandingConv: 28,
		MinCloseRate: 4, AvgCloseRate: 10, MaxCloseRate: 20,
	},
	"educacion": {
		MinCTR: 0.9, AvgCTR: 1.8, MaxCTR: 3.2,
		MinCPC: 0.15, AvgCPC: 0.35, MaxCPC: 0.80,
		MinCPM: 5.0, AvgCPM: 8.0, MaxCPM: 14.0,
		MinLandingConv: 10, AvgLandingConv: 20, MaxLandingConv: 35,
		MinCloseRate: 5, AvgCloseRate: 13, MaxCloseRate: 25,
	},
	"ecommerce": {
		MinCTR: 0.7, AvgCTR: 1.5, MaxCTR: 3.0,
		MinCPC: 0.20, AvgCPC: 0.50, MaxCPC: 1.20,
		MinCPM: 6.0, AvgCPM: 9.5, MaxCPM: 16.0,
		MinLandingConv: 2, AvgLandingConv: 4, MaxLandingConv: 8,
		MinCloseRate: 1.5, AvgCloseRate: 3.5, MaxCloseRate: 7,
	},
	"inmobiliaria": {
		MinCTR: 0.5, AvgCTR: 1.2, MaxCTR: 2.5,
		MinCPC: 0.60, AvgCPC: 1.50, MaxCPC: 4.00,
		MinCPM: 7.0, AvgCPM: 12.0, MaxCPM: 20.0,
		MinLandingConv: 3, AvgLandingConv: 8, MaxLandingConv: 15,
		MinCloseRate: 1, AvgCloseRate: 4, MaxCloseRate: 10,
	},
	"finanzas": {
		MinCTR: 0.4, AvgCTR: 1.0, MaxCTR: 2.2,
		MinCPC: 0.80, AvgCPC: 2.00, MaxCPC: 5.00,
		MinCPM: 8.0, AvgCPM: 14.0, MaxCPM: 24.0,
		MinLandingConv: 4, AvgLandingConv: 9, MaxLandingConv: 18,
		MinCloseRate: 2, AvgCloseRate: 6, MaxCloseRate: 14,
	},
	"restaurante": {
		MinCTR: 1.5, AvgCTR: 3.0, MaxCTR: 5.5,
		MinCPC: 0.05, AvgCPC: 0.12, MaxCPC: 0.30,
		MinCPM: 2.5, AvgCPM: 4.5, MaxCPM: 8.0,
		MinLandingConv: 10, AvgLandingConv: 22, MaxLandingConv: 40,
		MinCloseRate: 8, AvgCloseRate: 18, MaxCloseRate: 35,
	},
	"moda": {
		MinCTR: 0.8, AvgCTR: 1.8, MaxCTR: 3.5,
		MinCPC: 0.15, AvgCPC: 0.40, MaxCPC: 1.00,
		MinCPM: 5.0, AvgCPM: 8.5, MaxCPM: 15.0,
		MinLandingConv: 3, AvgLandingConv: 7, MaxLandingConv: 14,
		MinCloseRate: 2, AvgCloseRate: 5, MaxCloseRate: 12,
	},
	"tecnologia": {
		MinCTR: 0.5, AvgCTR: 1.2, MaxCTR: 2.5,
		MinCPC: 0.40, AvgCPC: 1.00, MaxCPC: 2.50,
		MinCPM: 6.0, AvgCPM: 11.0, MaxCPM: 18.0,
		MinLandingConv: 5, AvgLandingConv: 12, MaxLandingConv: 22,
		MinCloseRate: 3, AvgCloseRate: 8, MaxCloseRate: 18,
	},
	"servicios_profesionales": {
		MinCTR: 0.6, AvgCTR: 1.4, MaxCTR: 2.8,
		MinCPC: 0.30, AvgCPC: 0.80, MaxCPC: 2.00,
		MinCPM: 5.5, AvgCPM: 9.0, MaxCPM: 16.0,
		MinLandingConv: 6, AvgLandingConv: 14, MaxLandingConv: 25,
		MinCloseRate: 5, AvgCloseRate: 15, MaxCloseRate: 30,
	},
	"belleza": {
		MinCTR: 1.3, AvgCTR: 2.8, MaxCTR: 5.0,
		MinCPC: 0.06, AvgCPC: 0.15, MaxCPC: 0.35,
		MinCPM: 3.0, AvgCPM: 5.5, MaxCPM: 9.5,
		MinLandingConv: 8, AvgLandingConv: 18, MaxLandingConv: 32,
		MinCloseRate: 6, AvgCloseRate: 14, MaxCloseRate: 26,
	},
	"viajes": {
		MinCTR: 0.6, AvgCTR: 1.4, MaxCTR: 3.0,
		MinCPC: 0.25, AvgCPC: 0.70, MaxCPC: 1.80,
		MinCPM: 5.0, AvgCPM: 9.0, MaxCPM: 15.0,
		MinLandingConv: 4, AvgLandingConv: 10, MaxLandingConv: 20,
		MinCloseRate: 2, AvgCloseRate: 6, MaxCloseRate: 15,
	},
	"default": {
		MinCTR: 0.8, AvgCTR: 1.6, MaxCTR: 3.0,
		MinCPC: 0.20, AvgCPC: 0.55, MaxCPC: 1.40,
		MinCPM: 5.0, AvgCPM: 8.5, MaxCPM: 15.0,
		MinLandingConv: 6, AvgLandingConv: 14, MaxLandingConv: 25,
		MinCloseRate: 4, AvgCloseRate: 10, MaxCloseRate: 20,
	},
}

// industryKeywords mapea palabras clave detectables hacia el benchmark correcto.
var industryKeywords = map[string][]string{
	"wellness":                {"bienestar", "meditación", "mindfulness", "yoga", "meditacion", "wellness", "spa", "relajación", "relajacion", "estrés", "estres", "mental"},
	"salud":                   {"salud", "médico", "medico", "clínica", "clinica", "hospital", "nutrición", "nutricion", "dieta", "suplemento", "vitamina", "fisioterapia", "odontología", "odontologia", "dental", "farmacia", "psicología", "psicologia"},
	"educacion":               {"curso", "educación", "educacion", "academia", "aprendizaje", "formación", "formacion", "certificación", "certificacion", "coaching", "mentoring", "clase", "taller", "capacitación", "capacitacion", "enseñanza", "online", "virtual"},
	"ecommerce":               {"tienda", "shop", "store", "producto", "compra", "venta online", "ecommerce", "delivery", "envío", "envio", "catálogo", "catalogo"},
	"inmobiliaria":            {"inmobiliaria", "propiedad", "apartamento", "casa", "arriendo", "venta inmueble", "finca raíz", "finca raiz", "lote", "terreno", "constructora", "conjunto"},
	"finanzas":                {"finanzas", "crédito", "credito", "préstamo", "prestamo", "inversión", "inversion", "seguro", "ahorro", "banco", "crypto", "criptomoneda", "bolsa", "trading"},
	"restaurante":             {"restaurante", "comida", "menú", "menu", "gastronomía", "gastronomia", "cafetería", "cafeteria", "delivery comida", "pizzería", "pizzeria", "sushi", "burger", "panadería", "panaderia"},
	"moda":                    {"ropa", "moda", "fashion", "vestido", "camiseta", "zapato", "accesorio", "joyería", "joyeria", "bolso", "calzado", "outfit"},
	"tecnologia":              {"software", "app", "tecnología", "tecnologia", "saas", "desarrollo", "programación", "programacion", "startup", "digital", "automatización", "automatizacion", "inteligencia artificial", "ia"},
	"servicios_profesionales": {"consultoría", "consultoria", "abogado", "contador", "arquitecto", "diseñador", "disenador", "marketing", "publicidad", "agencia", "asesoría", "asesoria", "freelance", "servicio"},
	"belleza":                 {"belleza", "peluquería", "peluqueria", "estética", "estetica", "maquillaje", "manicure", "pedicure", "extensiones", "tratamiento capilar", "barbería", "barberia"},
	"viajes":                  {"viaje", "turismo", "hotel", "vuelo", "paquete turístico", "paquete turistico", "tour", "excursión", "excursion", "crucero", "hospedaje", "alojamiento"},
}

// detectIndustry analiza el texto del producto, oferta y target
// para encontrar el benchmark más adecuado por score de keywords.
func detectIndustry(product, offer, target string) string {
	text := strings.ToLower(product + " " + offer + " " + target)

	bestIndustry := "default"
	bestScore := 0

	for industry, keywords := range industryKeywords {
		score := 0
		for _, kw := range keywords {
			if strings.Contains(text, kw) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestIndustry = industry
		}
	}

	if bestScore == 0 {
		return "default"
	}

	return bestIndustry
}

// getBenchmark retorna el benchmark de una industria o el default.
func getBenchmark(industry string) IndustryBenchmark {
	if b, ok := industryBenchmarks[industry]; ok {
		return b
	}
	return industryBenchmarks["default"]
}

// ─────────────────────────────────────────────
// CAMPAIGN METRICS
// ─────────────────────────────────────────────

type CampaignMetrics struct {
	Budget float64
	Ticket float64

	CPL float64
	CTR float64
	CPM float64

	Leads   float64
	Sales   float64
	Revenue float64
	Profit  float64
	ROI     float64

	ConversionRate      float64
	BreakEvenConversion float64
}

// ─────────────────────────────────────────────
// BREAK EVEN
// ─────────────────────────────────────────────

func calculateBreakEven(metrics *CampaignMetrics) {
	if metrics.Leads == 0 || metrics.Ticket == 0 {
		return
	}
	breakEvenSales := metrics.Budget / metrics.Ticket
	metrics.BreakEvenConversion = breakEvenSales / metrics.Leads
}

// ─────────────────────────────────────────────
// DIAGNÓSTICO INTELIGENTE
// ─────────────────────────────────────────────

func diagnoseCampaign(m *CampaignMetrics) []string {
	issues := []string{}

	if m.ROI < -20 {
		issues = append(issues, "ROI muy negativo (peor de -20%)")
	} else if m.ROI < 0 {
		issues = append(issues, "ROI negativo leve — requiere ajuste")
	}

	if m.CTR < 0.5 {
		issues = append(issues, "CTR bajo (< 0.5%) — revisar creativos y copy")
	}

	if m.CPL > m.Ticket*0.6 {
		issues = append(issues, "CPL demasiado alto respecto al ticket promedio")
	}

	if m.BreakEvenConversion > 0 && m.ConversionRate < m.BreakEvenConversion {
		issues = append(issues, "Tasa de conversión por debajo del punto de equilibrio")
	}

	return issues
}

// ─────────────────────────────────────────────
// AUTO-OPTIMIZACIÓN
// ─────────────────────────────────────────────

func optimizeCampaign(m *CampaignMetrics) {
	if m.CTR < 1.0 {
		m.CTR *= 1.3
	}
	if m.CPL > m.Ticket*0.6 {
		m.CPL *= 0.75
	}
	if m.ConversionRate < 0.04 {
		m.ConversionRate *= 1.5
	}

	if m.CPL > 0 {
		m.Leads = m.Budget / m.CPL
	}
	m.Sales = m.Leads * m.ConversionRate
	m.Revenue = m.Sales * m.Ticket
	m.Profit = m.Revenue - m.Budget
	if m.Budget > 0 {
		m.ROI = (m.Profit / m.Budget) * 100
	}
}

// ─────────────────────────────────────────────
// LOOP INTELIGENTE
// ─────────────────────────────────────────────

func autoImproveCampaign(m *CampaignMetrics) {
	for i := 0; i < 5; i++ {
		calculateBreakEven(m)
		if m.ROI > 0 {
			break
		}
		optimizeCampaign(m)
	}
}

// ─────────────────────────────────────────────
// SCORE PROFESIONAL
// ─────────────────────────────────────────────

func calculateScore(m *CampaignMetrics) int {
	score := 100

	// ROI — penalidad proporcional
	if m.ROI < -50 {
		score -= 40
	} else if m.ROI < -20 {
		score -= 25
	} else if m.ROI < 0 {
		score -= 12
	}

	// CTR — umbral reducido a 0.5%
	if m.CTR < 0.5 {
		score -= 20
	} else if m.CTR < 1.0 {
		score -= 8
	}

	// CPL — umbral generoso: 60% del ticket
	if m.CPL > m.Ticket*0.6 {
		score -= 20
	} else if m.CPL > m.Ticket*0.4 {
		score -= 8
	}

	// Conversión vs break-even
	if m.BreakEvenConversion > 0 && m.ConversionRate < m.BreakEvenConversion {
		score -= 20
	}

	if score < 0 {
		score = 0
	}

	// Nunca devolver score perfecto: máximo honesto es 94
	if score > 94 {
		score = 94
	}

	return score
}

// ─────────────────────────────────────────────
// DECISIÓN AUTOMÁTICA
// ─────────────────────────────────────────────

func generateDecision(m *CampaignMetrics, score int) string {
	if m.ROI < -50 {
		return "❌ No lanzar campaña. Ajustar oferta, precio o proceso de cierre antes de invertir."
	}
	if score < 50 {
		return "⚠️ Campaña con riesgo alto. Optimizar creativos, CPL y tasa de conversión antes de escalar."
	}
	if score < 70 {
		return "🟡 Campaña viable con ajustes. Probar con presupuesto mínimo y medir CPL real."
	}
	if score < 85 {
		return "🟢 Campaña prometedora. Lanzar prueba y escalar al confirmar CPL objetivo."
	}
	return "🚀 Campaña con alto potencial. Escalar agresivamente si el CPL real se mantiene."
}

// ─────────────────────────────────────────────
// EVALUACIÓN DE CAMPAÑA
// ─────────────────────────────────────────────

type CampaignEvaluation struct {
	RealMetrics  *CampaignMetrics
	ScoreReal    int
	DecisionReal string
	IssuesReal   []string

	OptimizedProjection *CampaignMetrics
	ScoreOptimized      int
	DecisionOptimized   string
}

func evaluateCampaign(budget, ticket, cpl, ctr, cpm, conversionRate float64) CampaignEvaluation {
	if conversionRate <= 0 {
		conversionRate = 5
	}
	if conversionRate > 1 {
		conversionRate = conversionRate / 100
	}

	real := &CampaignMetrics{
		Budget:         budget,
		Ticket:         ticket,
		CPL:            cpl,
		CTR:            ctr,
		CPM:            cpm,
		ConversionRate: conversionRate,
	}

	if real.CPL > 0 {
		real.Leads = real.Budget / real.CPL
	}
	real.Sales = real.Leads * real.ConversionRate
	real.Revenue = real.Sales * real.Ticket
	real.Profit = real.Revenue - real.Budget
	if real.Budget > 0 {
		real.ROI = (real.Profit / real.Budget) * 100
	}
	calculateBreakEven(real)

	issuesReal := diagnoseCampaign(real)
	scoreReal := calculateScore(real)
	decisionReal := generateDecision(real, scoreReal)

	opt := &CampaignMetrics{
		Budget:         real.Budget,
		Ticket:         real.Ticket,
		CPL:            real.CPL,
		CTR:            real.CTR,
		CPM:            real.CPM,
		ConversionRate: real.ConversionRate,
		Leads:          real.Leads,
		Sales:          real.Sales,
		Revenue:        real.Revenue,
		Profit:         real.Profit,
		ROI:            real.ROI,
	}
	autoImproveCampaign(opt)
	calculateBreakEven(opt)

	scoreOpt := calculateScore(opt)
	decisionOpt := generateDecision(opt, scoreOpt)

	return CampaignEvaluation{
		RealMetrics:         real,
		ScoreReal:           scoreReal,
		DecisionReal:        decisionReal,
		IssuesReal:          issuesReal,
		OptimizedProjection: opt,
		ScoreOptimized:      scoreOpt,
		DecisionOptimized:   decisionOpt,
	}
}

// ─────────────────────────────────────────────
// STRUCTS
// ─────────────────────────────────────────────

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
	TicketAverage    float64 `json:"ticket_average"`
	EstimatedRevenue float64 `json:"estimated_revenue"`
	EstimatedProfit  float64 `json:"estimated_profit"`
	EstimatedROI     float64 `json:"estimated_roi"`
	BreakEvenCPL     float64 `json:"break_even_cpl"`
	Industry         string  `json:"industry"`

	// Reality Engine
	RawROI          float64  `json:"raw_roi"`
	AdjustedROI     float64  `json:"adjusted_roi"`
	ConfidenceScore float64  `json:"confidence_score"`
	ConfidenceLevel string   `json:"confidence_level"`
	RiskLevel       string   `json:"risk_level"`
	RealityWarnings []string `json:"reality_warnings"`
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
	TicketAverage       float64 `json:"ticket_average"`
	EstimatedRevenue    float64 `json:"estimated_revenue"`
	EstimatedProfit     float64 `json:"estimated_profit"`
	EstimatedROI        float64 `json:"estimated_roi"`
	BreakEvenCPL        float64 `json:"break_even_cpl"`
	Recommendation      string  `json:"recommendation"`
	Decision            string  `json:"decision"`
	ScaleSignal         string  `json:"scale_signal"`
	OptimizationTrigger string  `json:"optimization_trigger"`
	Industry            string  `json:"industry"`

	// Reality Engine
	RawROI          float64  `json:"raw_roi"`
	AdjustedROI     float64  `json:"adjusted_roi"`
	ConfidenceScore float64  `json:"confidence_score"`
	ConfidenceLevel string   `json:"confidence_level"`
	RiskLevel       string   `json:"risk_level"`
	RealityWarnings []string `json:"reality_warnings"`
}

type AdsCampaignPlan struct {
	Name           string   `json:"name"`
	Objective      string   `json:"objective"`
	Currency       string   `json:"currency"`
	Product        string   `json:"product"`
	Offer          string   `json:"offer"`
	TargetAudience string   `json:"target_audience"`
	Locations      []string `json:"locations"`
	AgeRange       string   `json:"age_range"`
	Interests      []string `json:"interests"`
	PainPoints     []string `json:"pain_points"`
	Angles         []string `json:"angles"`
	PrimaryText    string   `json:"primary_text"`
	Headline       string   `json:"headline"`
	Description    string   `json:"description"`
	CTA            string   `json:"cta"`
	CreativePrompt string   `json:"creative_prompt"`

	LandingSuggestion string `json:"landing_suggestion"`
	WhatsAppScript    string `json:"whatsapp_script"`

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

	Industry string `json:"industry"`

	CampaignScoreReal    int      `json:"campaign_score_real"`
	CampaignDecisionReal string   `json:"campaign_decision_real"`
	CampaignIssues       []string `json:"campaign_issues"`

	CampaignScoreOptimized    int    `json:"campaign_score_optimized"`
	CampaignDecisionOptimized string `json:"campaign_decision_optimized"`
}

// ─────────────────────────────────────────────
// GENERATE CAMPAIGN PLAN
// ─────────────────────────────────────────────

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

	// Detectar industria ANTES de normalizar
	industry := detectIndustry(product, offer, target)
	plan.Industry = industry

	plan = normalizeCampaignPlan(plan, product, offer, target, country, currency, budgetDaily, ticketAverage, industry)

	// Evaluar con el sistema de métricas
	eval := evaluateCampaign(
		plan.BudgetMonthly,
		ticketAverage,
		plan.ROI.EstimatedCPL,
		plan.ROI.EstimatedCTR,
		plan.ROI.EstimatedCPM,
		plan.ROI.ConversionRate,
	)

	plan.CampaignScoreReal = eval.ScoreReal
	plan.CampaignDecisionReal = eval.DecisionReal
	plan.CampaignIssues = eval.IssuesReal
	plan.CampaignScoreOptimized = eval.ScoreOptimized
	plan.CampaignDecisionOptimized = eval.DecisionOptimized

	return plan, nil
}

// ─────────────────────────────────────────────
// REPAIR AI JSON
// ─────────────────────────────────────────────

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

// ─────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────

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
	industry string,
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

	plan = normalizeROI(plan, budgetDaily, ticketAverage, currency, industry)
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

// ─────────────────────────────────────────────
// ROI — CONTEXTUAL POR INDUSTRIA + REALITY ENGINE
// ─────────────────────────────────────────────

func normalizeROI(plan AdsCampaignPlan, budgetDaily float64, ticketAverage float64, currency string, industry string) AdsCampaignPlan {
	scenarios := buildROIScenarios(budgetDaily, ticketAverage, currency, industry)

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
		Industry:         industry,
		// Reality Engine heredado del escenario realista
		RawROI:          realistic.RawROI,
		AdjustedROI:     realistic.AdjustedROI,
		ConfidenceScore: realistic.ConfidenceScore,
		ConfidenceLevel: realistic.ConfidenceLevel,
		RiskLevel:       realistic.RiskLevel,
		RealityWarnings: realistic.RealityWarnings,
	}

	plan.ROIScenarios = scenarios

	return plan
}

// buildROIScenarios construye 3 escenarios usando benchmarks reales de la industria.
// Conservador usa los peores valores (MaxCPM + Min tasas),
// Realista usa promedios, Agresivo usa los mejores valores.
func buildROIScenarios(budgetDaily float64, ticketAverage float64, currency string, industry string) []AdsROIScenario {
	if budgetDaily <= 0 {
		budgetDaily = 10
	}
	if ticketAverage <= 0 {
		ticketAverage = 50
	}

	budgetMonthly := budgetDaily * 30
	b := getBenchmark(industry)

	return []AdsROIScenario{
		calcROIScenario(
			"Conservador",
			currency,
			budgetDaily,
			budgetMonthly,
			ticketAverage,
			b.MaxCPM,         // CPM más caro → menos alcance → más conservador
			b.MinCTR,
			b.MinLandingConv,
			b.MinCloseRate,
			industry,
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
			b.AvgCPM,
			b.AvgCTR,
			b.AvgLandingConv,
			b.AvgCloseRate,
			industry,
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
			b.MinCPM,         // CPM más barato → más alcance → más agresivo
			b.MaxCTR,
			b.MaxLandingConv,
			b.MaxCloseRate,
			industry,
			"Solo escalar a este escenario si hay buen CTR, buen CPL, respuesta rápida y cierre comercial comprobado.",
			"Escalar creativos ganadores y activar remarketing.",
			"Duplicar adset ganador y aumentar presupuesto gradualmente.",
			"Caída de CTR, aumento de CPL o caída de tasa de cierre.",
		),
	}
}

// calcROIScenario calcula un escenario completo incluyendo el Reality Engine.
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
	industry string,
	recommendation string,
	decision string,
	scaleSignal string,
	optimizationTrigger string,
) AdsROIScenario {
	// ── Métricas base ────────────────────────────────────────────────────────
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

	salesFloat := float64(leads) * (closeRatePercent / 100)
	sales := int(math.Floor(salesFloat))
	if sales == 0 && leads > 3 {
		sales = 1
	}

	revenue := float64(sales) * ticketAverage
	profit := revenue - budgetMonthly

	roi := 0.0
	if budgetMonthly > 0 {
		roi = (profit / budgetMonthly) * 100
	}

	// Techo de ROI bruto antes del ajuste por reality factor
	if roi > 250 {
		roi = 250
	}

	breakEvenCPL := 0.0
	if closeRatePercent > 0 {
		breakEvenCPL = ticketAverage * (closeRatePercent / 100)
	}

	// ── REALITY ENGINE ───────────────────────────────────────────────────────
	rawROI := roi
	confidenceScore := 100.0
	warnings := []string{}

	if ctrPercent > 3.5 {
		confidenceScore -= 10
		warnings = append(warnings, "CTR alto: validar con datos reales antes de escalar.")
	}

	if landingConvPercent > 25 {
		confidenceScore -= 12
		warnings = append(warnings, "Conversión landing alta: puede requerir una landing muy optimizada.")
	}

	if closeRatePercent > 15 {
		confidenceScore -= 15
		warnings = append(warnings, "Tasa de cierre alta: depende mucho del seguimiento comercial.")
	}

	if cpl > 0 && cpl < 0.75 {
		confidenceScore -= 15
		warnings = append(warnings, "CPL muy bajo: posible escenario optimista para tráfico frío.")
	}

	if rawROI > 150 {
		confidenceScore -= 20
		warnings = append(warnings, "ROI elevado: tratar como proyección agresiva, no como garantía.")
	}

	if budgetMonthly < 150 {
		confidenceScore -= 10
		warnings = append(warnings, "Presupuesto bajo: los datos pueden ser poco estables.")
	}

	if confidenceScore < 25 {
		confidenceScore = 25
	}

	realityFactor := confidenceScore / 100
	adjustedROI := rawROI * realityFactor

	adjustedProfit := budgetMonthly * (adjustedROI / 100)
	adjustedRevenue := budgetMonthly + adjustedProfit

	if adjustedROI > 220 {
		adjustedROI = 220
	}

	confidenceLevel := "Alta"
	if confidenceScore < 50 {
		confidenceLevel = "Baja"
	} else if confidenceScore < 75 {
		confidenceLevel = "Media"
	}

	riskLevel := "Bajo"
	if adjustedROI < 0 {
		riskLevel = "Alto"
	} else if confidenceScore < 65 {
		riskLevel = "Medio"
	}
	// ── FIN REALITY ENGINE ───────────────────────────────────────────────────

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
		EstimatedRevenue: 	 round2(adjustedRevenue),
		EstimatedProfit:     round2(adjustedProfit),
		EstimatedROI:        round2(adjustedROI), // ROI ya ajustado por reality factor
		BreakEvenCPL:        round2(breakEvenCPL),
		Recommendation:      recommendation,
		Decision:            decision,
		ScaleSignal:         scaleSignal,
		OptimizationTrigger: optimizationTrigger,
		Industry:            industry,
		// Reality Engine
		RawROI:          round2(rawROI),
		AdjustedROI:     round2(adjustedROI),
		ConfidenceScore: round2(confidenceScore),
		ConfidenceLevel: confidenceLevel,
		RiskLevel:       riskLevel,
		RealityWarnings: warnings,
	}
}

// ─────────────────────────────────────────────
// MISC HELPERS
// ─────────────────────────────────────────────

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

// ─────────────────────────────────────────────
// SAVE CAMPAIGN
// ─────────────────────────────────────────────

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