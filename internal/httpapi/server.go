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
}

func New(
	db *sql.DB,
	manager *services.BotManager,
	auth *services.AuthService,
	templates *services.TemplateService,
	funnel *services.FunnelService,
	social *services.SocialService,
) *Server {
	s := &Server{
		DB:        db,
		Router:    mux.NewRouter(),
		Manager:   manager,
		Auth:      auth,
		Templates: templates,
		Funnel:    funnel,
		Social:    social,
	}
	s.routes()
	return s
}

func NewServer() (*Server, error) {
	dataDir := filepath.Join("data")
	dbPath := filepath.Join(dataDir, "app.db")

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

	return New(db, manager, auth, templates, funnel, social), nil
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