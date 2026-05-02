package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"

	dbpkg "whatsapp-sales-os-enterprise/backend/internal/db"
	"whatsapp-sales-os-enterprise/backend/internal/services"
)

type Server struct {
	DB        *sql.DB
	Router    *mux.Router
	Manager   *services.BotManager
	Auth      *services.AuthService
	Templates *services.TemplateService
	Funnel    *services.FunnelService
	Social    *services.SocialService
	Billing   *services.BillingService
	Ads       *services.AdsService
	Groups 	  *services.GroupService
	Assistant *services.AssistantService
}

func New(
	db *sql.DB,
	manager *services.BotManager,
	auth *services.AuthService,
	templates *services.TemplateService,
	funnel *services.FunnelService,
	social *services.SocialService,
	billing *services.BillingService,
	ads *services.AdsService,
	groups *services.GroupService,
	assistant *services.AssistantService,
) *Server {
	s := &Server{
		DB:        db,
		Router:    mux.NewRouter(),
		Manager:   manager,
		Auth:      auth,
		Templates: templates,
		Funnel:    funnel,
		Social:    social,
		Billing:   billing,
		Ads:       ads,
		Groups:    groups,
		Assistant: assistant,
	}
	s.routes()
	return s
}

func NewServer() (*Server, error) {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join(dataDir, "app.db")
	}

	db, err := dbpkg.Open(dbPath)
	if err != nil {
		return nil, err
	}

	openAIKey := os.Getenv("OPENAI_API_KEY")
	openAIModel := os.Getenv("OPENAI_MODEL")
	if openAIModel == "" {
		openAIModel = "gpt-4o-mini"
	}

	publicBaseURL := os.Getenv("PUBLIC_BASE_URL")
	socialAssetsDir := filepath.Join(dataDir, "social_assets")

	ai := services.NewAIService(openAIKey, openAIModel)
	manager := services.NewBotManager(db, ai, filepath.Join(dataDir, "bots"))
	auth := services.NewAuthService(db, 30)
	templates := services.NewTemplateService(db)
	funnel := services.NewFunnelService(db)
	social := services.NewSocialService(db, ai, socialAssetsDir, publicBaseURL)

	manager.Funnel = funnel

	scheduler := services.NewSocialScheduler(db, social)
	scheduler.Start()

	billing := services.NewBillingService(db)
	_ = billing.SeedDefaults()

	ads := services.NewAdsService(db, ai)
	groups := services.NewGroupService(db, ai)
	assistant := services.NewAssistantService(db, ai)

	return New(db, manager, auth, templates, funnel, social, billing, ads, groups, assistant), nil
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.Router)
}

func (s *Server) Shutdown(_ context.Context) error {
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}