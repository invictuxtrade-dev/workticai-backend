package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"whatsapp-sales-os-enterprise/backend/internal/models"
	"whatsapp-sales-os-enterprise/backend/internal/services"
)

func (s *Server) routes() {
	r := s.Router
	r.Use(s.withCORS)

	r.HandleFunc("/api/health", s.handleHealth).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/auth/bootstrap", s.handleBootstrap).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auth/login", s.handleLogin).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auth/me", s.authRequired(s.handleMe)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/auth/register-client", s.handleRegisterClient).Methods("POST", "OPTIONS")

	// Público para tracking de landings
	r.HandleFunc("/api/public/funnel/event", s.handleTrackFunnelEvent).Methods("POST", "OPTIONS")

	// Assets públicos de Social IA
	r.PathPrefix("/social-assets/").Handler(
		http.StripPrefix(
			"/social-assets/",
			http.FileServer(http.Dir(filepath.Join("data", "social_assets"))),
		),
	)

	secured := r.PathPrefix("/api").Subrouter()
	secured.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			s.authRequired(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})(w, r)
		})
	})

	secured.HandleFunc("/dashboard/metrics", s.handleMetrics).Methods("GET", "OPTIONS")

	secured.HandleFunc("/clients", requireRole("admin")(s.handleListClients)).Methods("GET", "OPTIONS")
	secured.HandleFunc("/clients", requireRole("admin")(s.handleCreateClient)).Methods("POST", "OPTIONS")
	secured.HandleFunc("/clients/{id}", requireRole("admin")(s.handleUpdateClient)).Methods("PUT", "OPTIONS")
	secured.HandleFunc("/clients/{id}", requireRole("admin")(s.handleDeleteClient)).Methods("DELETE", "OPTIONS")

	secured.HandleFunc("/users", requireRole("admin")(s.handleListUsers)).Methods("GET", "OPTIONS")
	secured.HandleFunc("/users", requireRole("admin")(s.handleCreateUser)).Methods("POST", "OPTIONS")
	secured.HandleFunc("/users/{id}", requireRole("admin")(s.handleDeleteUser)).Methods("DELETE", "OPTIONS")

	secured.HandleFunc("/templates", s.handleListTemplates).Methods("GET", "OPTIONS")
	secured.HandleFunc("/templates", s.handleCreateTemplate).Methods("POST", "OPTIONS")
	secured.HandleFunc("/templates/{id}", s.handleUpdateTemplate).Methods("PUT", "OPTIONS")
	secured.HandleFunc("/templates/{id}", s.handleDeleteTemplate).Methods("DELETE", "OPTIONS")

	secured.HandleFunc("/bots", s.handleListBots).Methods("GET", "OPTIONS")
	secured.HandleFunc("/bots", s.handleCreateBot).Methods("POST", "OPTIONS")
	secured.HandleFunc("/bots/{id}/start", s.handleStartBot).Methods("POST", "OPTIONS")
	secured.HandleFunc("/bots/{id}", s.handleUpdateBot).Methods("PUT", "OPTIONS")
	secured.HandleFunc("/bots/{id}", s.handleDeleteBot).Methods("DELETE", "OPTIONS")
	secured.HandleFunc("/bots/{id}/stop", s.handleStopBot).Methods("POST", "OPTIONS")
	secured.HandleFunc("/bots/{id}/qr", s.handleGetQR).Methods("GET", "OPTIONS")
	secured.HandleFunc("/bots/{id}/config", s.handleGetBotConfig).Methods("GET", "OPTIONS")
	secured.HandleFunc("/bots/{id}/config", s.handleUpsertBotConfig).Methods("PUT", "OPTIONS")
	secured.HandleFunc("/bots/{id}/send", s.handleSend).Methods("POST", "OPTIONS")

	secured.HandleFunc("/landings", s.handleListLandings).Methods("GET", "OPTIONS")
	secured.HandleFunc("/landings", s.handleCreateLanding).Methods("POST", "OPTIONS")
	secured.HandleFunc("/landings/{id}", s.handleGetLanding).Methods("GET", "OPTIONS")
	secured.HandleFunc("/landings/{id}", s.handleUpdateLanding).Methods("PUT", "OPTIONS")
	secured.HandleFunc("/landings/{id}", s.handleDeleteLanding).Methods("DELETE", "OPTIONS")
	secured.HandleFunc("/landings/generate", s.handleGenerateLanding).Methods("POST", "OPTIONS")

	secured.HandleFunc("/inbox/leads", s.handleInboxLeads).Methods("GET", "OPTIONS")
	secured.HandleFunc("/funnel", s.handleFunnelMetrics).Methods("GET", "OPTIONS")
	secured.HandleFunc("/funnel/event", s.handleTrackFunnelEvent).Methods("POST", "OPTIONS")

	secured.HandleFunc("/bots/{id}/leads", s.handleLeads).Methods("GET", "OPTIONS")
	secured.HandleFunc("/bots/{id}/leads/{leadID}/stage", s.handleUpdateLeadStage).Methods("PATCH", "OPTIONS")
	secured.HandleFunc("/bots/{id}/leads/{leadID}/messages", s.handleLeadMessages).Methods("GET", "OPTIONS")
	secured.HandleFunc("/bots/{id}/leads/{leadID}/send", s.handleLeadSend).Methods("POST", "OPTIONS")

	secured.HandleFunc("/social/generate", s.handleSocialGenerate).Methods("POST", "OPTIONS")
	secured.HandleFunc("/social/posts", s.handleSocialPosts).Methods("GET", "OPTIONS")
	secured.HandleFunc("/social/credentials", s.handleGetSocialCredentials).Methods("GET", "OPTIONS")
	secured.HandleFunc("/social/credentials", s.handleSaveSocialCredentials).Methods("PUT", "OPTIONS")
	secured.HandleFunc("/social/campaigns", s.handleCreateSocialCampaign).Methods("POST", "OPTIONS")
	secured.HandleFunc("/social/publish-now", s.handlePublishSocialNow).Methods("POST", "OPTIONS")
	secured.HandleFunc("/social/schedule", s.handleScheduleSocialPost).Methods("POST", "OPTIONS")
	secured.HandleFunc("/social/logs", s.handleSocialLogs).Methods("GET", "OPTIONS")
	secured.HandleFunc("/social/generate-image", s.handleSocialGenerateImage).Methods("POST", "OPTIONS")
	secured.HandleFunc("/social/upload-image", s.handleSocialUploadImage).Methods("POST", "OPTIONS")
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	user, token, err := s.Auth.BootstrapAdmin(body.Name, body.Email, body.Password)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"token": token,
		"user":  user,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	user, token, err := s.Auth.Login(body.Email, body.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	cid := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		cid = u.ClientID
	}

	m, err := s.Manager.Metrics(cid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleListClients(w http.ResponseWriter, _ *http.Request) {
	clients, err := s.Manager.ListClients()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, clients)
}

func (s *Server) handleCreateClient(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
		Plan  string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	c, err := s.Manager.CreateClient(body.Name, body.Email, body.Phone, body.Plan)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateClient(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var c models.Client

	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	c.ID = id
	if err := s.Manager.UpdateClient(c); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleDeleteClient(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := s.Manager.DeleteClient(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	cid := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		cid = u.ClientID
	}

	users, err := s.Auth.ListUsers(cid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body struct {
		ClientID string `json:"client_id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if u.Role != "admin" {
		body.ClientID = u.ClientID
		if body.Role == "" || body.Role == "admin" {
			body.Role = "client_user"
		}
	}

	user, _, err := s.Auth.CreateUser(body.ClientID, body.Name, body.Email, body.Password, body.Role)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if err := s.Auth.DeleteUser(mux.Vars(r)["id"]); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	cid := u.ClientID
	if u.Role == "admin" && r.URL.Query().Get("client_id") != "" {
		cid = r.URL.Query().Get("client_id")
	}

	items, err := s.Templates.List(cid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var t models.Template

	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if u.Role != "admin" {
		t.ClientID = u.ClientID
	}

	item, err := s.Templates.Create(t)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	var t models.Template

	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	t.ID = mux.Vars(r)["id"]
	if err := s.Templates.Update(t); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	if err := s.Templates.Delete(mux.Vars(r)["id"]); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleListBots(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	cid := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		cid = u.ClientID
	}

	bots, err := s.Manager.ListBots(cid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, bots)
}

func (s *Server) handleCreateBot(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body struct {
		ClientID string `json:"client_id"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if u.Role != "admin" {
		body.ClientID = u.ClientID
	}

	if strings.TrimSpace(body.ClientID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id required"})
		return
	}

	if strings.TrimSpace(body.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name required"})
		return
	}

	bot, err := s.Manager.CreateBot(body.ClientID, body.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, bot)
}

func (s *Server) handleStartBot(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := s.Manager.StartBot(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	bot, _ := s.Manager.GetBot(id)
	writeJSON(w, http.StatusOK, bot)
}

func (s *Server) handleGetQR(w http.ResponseWriter, r *http.Request) {
	qr, err := s.Manager.GetQR(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"qr": qr})
}

func (s *Server) handleGetBotConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.Manager.GetBotConfig(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleUpsertBotConfig(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var cfg models.BotConfig

	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	cfg.BotID = id
	updated, err := s.Manager.UpsertBotConfig(cfg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var body struct {
		Number  string `json:"number"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	body.Number = strings.TrimSpace(strings.TrimPrefix(body.Number, "+"))
	body.Message = strings.TrimSpace(body.Message)

	if body.Number == "" || body.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "number and message are required"})
		return
	}

	if err := s.Manager.SendText(id, body.Number, body.Message); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleListLandings(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	clientID := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		clientID = u.ClientID
	}

	query := `
		SELECT
		id, client_id, bot_id, name, prompt, status,
		style_preset, logo_url, favicon_url, hero_image_url, youtube_url,
		facebook_pixel_id, google_analytics,
		primary_color, secondary_color,
		show_video, show_image,
		html, css, js, preview_html, whatsapp_url,
		tracking_mode, tracking_base_url,
		created_at, updated_at
		FROM landing_pages
	`
	args := []any{}
	if strings.TrimSpace(clientID) != "" {
		query += ` WHERE client_id=?`
		args = append(args, clientID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []models.LandingPage{}
	for rows.Next() {
		var lp models.LandingPage
		if err := rows.Scan(
			&lp.ID, &lp.ClientID, &lp.BotID, &lp.Name, &lp.Prompt, &lp.Status,
			&lp.StylePreset, &lp.LogoURL, &lp.FaviconURL, &lp.HeroImageURL, &lp.YoutubeURL,
			&lp.FacebookPixelID, &lp.GoogleAnalytics,
			&lp.PrimaryColor, &lp.SecondaryColor,
			&lp.ShowVideo, &lp.ShowImage,
			&lp.Html, &lp.Css, &lp.Js, &lp.PreviewHTML, &lp.WhatsappURL,
			&lp.TrackingMode, &lp.TrackingBaseURL,
			&lp.CreatedAt, &lp.UpdatedAt,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		out = append(out, lp)
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateLanding(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var lp models.LandingPage

	if err := json.NewDecoder(r.Body).Decode(&lp); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if u.Role != "admin" {
		lp.ClientID = u.ClientID
	}

	if strings.TrimSpace(lp.ClientID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id required"})
		return
	}
	if strings.TrimSpace(lp.BotID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bot_id required"})
		return
	}
	if strings.TrimSpace(lp.Name) == "" {
		lp.Name = "Landing"
	}
	if strings.TrimSpace(lp.Status) == "" {
		lp.Status = "draft"
	}
	if strings.TrimSpace(lp.TrackingMode) == "" {
		lp.TrackingMode = "auto"
	}

	now := time.Now()
	lp.ID = uuid.NewString()
	lp.CreatedAt = now
	lp.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO landing_pages (
			id, client_id, bot_id, name, prompt, status,
			style_preset, logo_url, favicon_url, hero_image_url, youtube_url,
			facebook_pixel_id, google_analytics,
			primary_color, secondary_color,
			show_video, show_image,
			html, css, js, preview_html, whatsapp_url,
			tracking_mode, tracking_base_url,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		lp.ID, lp.ClientID, lp.BotID, lp.Name, lp.Prompt, lp.Status,
		lp.StylePreset, lp.LogoURL, lp.FaviconURL, lp.HeroImageURL, lp.YoutubeURL,
		lp.FacebookPixelID, lp.GoogleAnalytics,
		lp.PrimaryColor, lp.SecondaryColor,
		lp.ShowVideo, lp.ShowImage,
		lp.Html, lp.Css, lp.Js, lp.PreviewHTML, lp.WhatsappURL,
		lp.TrackingMode, lp.TrackingBaseURL,
		lp.CreatedAt, lp.UpdatedAt,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, lp)
}

func (s *Server) handleGetLanding(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var lp models.LandingPage
	err := s.DB.QueryRow(`
		SELECT
		id, client_id, bot_id, name, prompt, status,
		style_preset, logo_url, favicon_url, hero_image_url, youtube_url,
		facebook_pixel_id, google_analytics,
		primary_color, secondary_color,
		show_video, show_image,
		html, css, js, preview_html, whatsapp_url,
		tracking_mode, tracking_base_url,
		created_at, updated_at
		FROM landing_pages
		WHERE id=?
	`, id).Scan(
		&lp.ID, &lp.ClientID, &lp.BotID, &lp.Name, &lp.Prompt, &lp.Status,
		&lp.StylePreset, &lp.LogoURL, &lp.FaviconURL, &lp.HeroImageURL, &lp.YoutubeURL,
		&lp.FacebookPixelID, &lp.GoogleAnalytics,
		&lp.PrimaryColor, &lp.SecondaryColor,
		&lp.ShowVideo, &lp.ShowImage,
		&lp.Html, &lp.Css, &lp.Js, &lp.PreviewHTML, &lp.WhatsappURL,
		&lp.TrackingMode, &lp.TrackingBaseURL,
		&lp.CreatedAt, &lp.UpdatedAt,
	)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "landing not found"})
		return
	}

	writeJSON(w, http.StatusOK, lp)
}

func (s *Server) handleUpdateLanding(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var lp models.LandingPage

	if err := json.NewDecoder(r.Body).Decode(&lp); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if strings.TrimSpace(lp.TrackingMode) == "" {
		lp.TrackingMode = "auto"
	}

	lp.UpdatedAt = time.Now()

	_, err := s.DB.Exec(`
		UPDATE landing_pages SET
			name=?,
			prompt=?,
			status=?,
			style_preset=?,
			logo_url=?,
			favicon_url=?,
			hero_image_url=?,
			youtube_url=?,
			facebook_pixel_id=?,
			google_analytics=?,
			primary_color=?,
			secondary_color=?,
			show_video=?,
			show_image=?,
			html=?,
			css=?,
			js=?,
			preview_html=?,
			whatsapp_url=?,
			tracking_mode=?,
			tracking_base_url=?,
			updated_at=?
		WHERE id=?
	`,
		lp.Name,
		lp.Prompt,
		lp.Status,
		lp.StylePreset,
		lp.LogoURL,
		lp.FaviconURL,
		lp.HeroImageURL,
		lp.YoutubeURL,
		lp.FacebookPixelID,
		lp.GoogleAnalytics,
		lp.PrimaryColor,
		lp.SecondaryColor,
		lp.ShowVideo,
		lp.ShowImage,
		lp.Html,
		lp.Css,
		lp.Js,
		lp.PreviewHTML,
		lp.WhatsappURL,
		lp.TrackingMode,
		lp.TrackingBaseURL,
		lp.UpdatedAt,
		id,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleDeleteLanding(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	_, err := s.DB.Exec(`DELETE FROM landing_pages WHERE id=?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleGenerateLanding(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body struct {
		BotID           string `json:"bot_id"`
		Name            string `json:"name"`
		Prompt          string `json:"prompt"`
		StylePreset     string `json:"style_preset"`
		LogoURL         string `json:"logo_url"`
		FaviconURL      string `json:"favicon_url"`
		HeroImageURL    string `json:"hero_image_url"`
		YoutubeURL      string `json:"youtube_url"`
		FacebookPixelID string `json:"facebook_pixel_id"`
		GoogleAnalytics string `json:"google_analytics"`
		PrimaryColor    string `json:"primary_color"`
		SecondaryColor  string `json:"secondary_color"`
		ShowVideo       bool   `json:"show_video"`
		ShowImage       bool   `json:"show_image"`
		TrackingMode    string `json:"tracking_mode"`
		TrackingBaseURL string `json:"tracking_base_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if strings.TrimSpace(body.BotID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bot_id required"})
		return
	}

	bot, err := s.Manager.GetBot(body.BotID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	cfg, _ := s.Manager.GetBotConfig(body.BotID)

	lp, err := s.Manager.Landing.GenerateLanding(
		r.Context(),
		bot,
		cfg,
		models.LandingPage{
			ClientID:        bot.ClientID,
			BotID:           body.BotID,
			Name:            body.Name,
			Prompt:          body.Prompt,
			StylePreset:     body.StylePreset,
			LogoURL:         body.LogoURL,
			FaviconURL:      body.FaviconURL,
			HeroImageURL:    body.HeroImageURL,
			YoutubeURL:      body.YoutubeURL,
			FacebookPixelID: body.FacebookPixelID,
			GoogleAnalytics: body.GoogleAnalytics,
			PrimaryColor:    body.PrimaryColor,
			SecondaryColor:  body.SecondaryColor,
			ShowVideo:       body.ShowVideo,
			ShowImage:       body.ShowImage,
			TrackingMode:    body.TrackingMode,
			TrackingBaseURL: body.TrackingBaseURL,
		},
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if u.Role != "admin" {
		lp.ClientID = u.ClientID
	}

	if lp.ID == "" {
		lp.ID = uuid.NewString()
	}
	if lp.Status == "" {
		lp.Status = "generated"
	}
	if strings.TrimSpace(lp.TrackingMode) == "" {
		lp.TrackingMode = "auto"
	}
	if lp.CreatedAt.IsZero() {
		lp.CreatedAt = time.Now()
	}
	lp.UpdatedAt = time.Now()

	_, err = s.DB.Exec(`
		INSERT INTO landing_pages (
			id, client_id, bot_id, name, prompt, status,
			style_preset, logo_url, favicon_url, hero_image_url, youtube_url,
			facebook_pixel_id, google_analytics,
			primary_color, secondary_color,
			show_video, show_image,
			html, css, js, preview_html, whatsapp_url,
			tracking_mode, tracking_base_url,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		lp.ID, lp.ClientID, lp.BotID, lp.Name, lp.Prompt, lp.Status,
		lp.StylePreset, lp.LogoURL, lp.FaviconURL, lp.HeroImageURL, lp.YoutubeURL,
		lp.FacebookPixelID, lp.GoogleAnalytics,
		lp.PrimaryColor, lp.SecondaryColor,
		lp.ShowVideo, lp.ShowImage,
		lp.Html, lp.Css, lp.Js, lp.PreviewHTML, lp.WhatsappURL,
		lp.TrackingMode, lp.TrackingBaseURL,
		lp.CreatedAt, lp.UpdatedAt,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, lp)
}

func (s *Server) handleInboxLeads(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	clientID := r.URL.Query().Get("client_id")
	botID := r.URL.Query().Get("bot_id")

	if u.Role != "admin" {
		clientID = u.ClientID
	}

	leads, err := s.Manager.ListInboxLeads(clientID, botID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, leads)
}

func (s *Server) handleLeads(w http.ResponseWriter, r *http.Request) {
	leads, err := s.Manager.ListLeads(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, leads)
}

func (s *Server) handleUpdateLeadStage(w http.ResponseWriter, r *http.Request) {
	botID := mux.Vars(r)["id"]
	leadID, _ := strconv.ParseInt(mux.Vars(r)["leadID"], 10, 64)

	var body struct {
		Stage string `json:"stage"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	stage := strings.TrimSpace(strings.ToLower(body.Stage))
	if stage == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "stage required"})
		return
	}

	if err := s.Manager.UpdateLeadStage(botID, leadID, stage); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if s.Funnel != nil {
		bot, _ := s.Manager.GetBot(botID)

		switch stage {
		case "qualified", "interested", "hot":
			_ = s.Funnel.TrackEvent(bot.ClientID, botID, "", "lead_qualified", "manual:"+stage)
		case "closed":
			_ = s.Funnel.TrackEvent(bot.ClientID, botID, "", "conversion", "manual:closed")
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleLeadMessages(w http.ResponseWriter, r *http.Request) {
	botID := mux.Vars(r)["id"]
	leadID, _ := strconv.ParseInt(mux.Vars(r)["leadID"], 10, 64)

	msgs, err := s.Manager.LeadMessages(botID, leadID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (s *Server) handleLeadSend(w http.ResponseWriter, r *http.Request) {
	botID := mux.Vars(r)["id"]
	leadID, _ := strconv.ParseInt(mux.Vars(r)["leadID"], 10, 64)

	var body struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	body.Message = strings.TrimSpace(body.Message)
	if body.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "message required"})
		return
	}

	if err := s.Manager.SendToLead(botID, leadID, body.Message); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleStopBot(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := s.Manager.StopBot(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	bot, _ := s.Manager.GetBot(id)
	writeJSON(w, http.StatusOK, bot)
}

func (s *Server) handleUpdateBot(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var body struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	bot, err := s.Manager.UpdateBot(id, body.Name)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, bot)
}

func (s *Server) handleDeleteBot(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if err := s.Manager.DeleteBot(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleFunnelMetrics(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	clientID := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		clientID = u.ClientID
	}

	if s.Funnel == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "funnel service not configured"})
		return
	}

	data, err := s.Funnel.Metrics(clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleTrackFunnelEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientID  string `json:"client_id"`
		BotID     string `json:"bot_id"`
		LandingID string `json:"landing_id"`
		EventType string `json:"event_type"`
		Metadata  string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if strings.TrimSpace(body.EventType) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "event_type required"})
		return
	}

	if s.Funnel == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "funnel service not configured"})
		return
	}

	if err := s.Funnel.TrackEvent(body.ClientID, body.BotID, body.LandingID, body.EventType, body.Metadata); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleSocialPosts(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	clientID := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		clientID = u.ClientID
	}

	rows, err := s.DB.Query(`
		SELECT 
			id, client_id, campaign_id, platform, content, image_url, target_url,
			publish_mode, image_mode, image_prompt,
			status, error, facebook_post_id,
			scheduled_at, published_at, created_at
		FROM social_posts
		WHERE (? = '' OR client_id = ?)
		ORDER BY created_at DESC
	`, clientID, clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []models.SocialPost{}
	for rows.Next() {
		var p models.SocialPost
		if err := rows.Scan(
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
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		out = append(out, p)
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleSocialGenerate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Prompt string `json:"prompt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	content, err := s.Social.GenerateContent(r.Context(), models.SocialCampaign{
		Prompt: body.Prompt,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"content": content,
	})
}

func (s *Server) handleGetSocialCredentials(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	clientID := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		clientID = u.ClientID
	}

	var c models.SocialCredential
	err := s.DB.QueryRow(`
		SELECT id, client_id, platform, access_token, page_id, page_name, enabled, ad_account_id, created_at, updated_at
		FROM social_credentials
		WHERE client_id=? AND platform='facebook'
		LIMIT 1
	`, clientID).Scan(
		&c.ID,
		&c.ClientID,
		&c.Platform,
		&c.AccessToken,
		&c.PageID,
		&c.PageName,
		&c.Enabled,
		&c.AdAccountID,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleSaveSocialCredentials(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var c models.SocialCredential

	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if u.Role != "admin" {
		c.ClientID = u.ClientID
	}
	if strings.TrimSpace(c.Platform) == "" {
		c.Platform = "facebook"
	}

	out, err := s.Social.SaveCredential(c)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateSocialCampaign(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var c models.SocialCampaign

	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if u.Role != "admin" {
		c.ClientID = u.ClientID
	}

	out, err := s.Social.CreateCampaign(c)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) handlePublishSocialNow(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body struct {
		CampaignID  string `json:"campaign_id"`
		Content     string `json:"content"`
		ImageURL    string `json:"image_url"`
		ImageMode   string `json:"image_mode"`
		ImagePrompt string `json:"image_prompt"`
		Objective   string `json:"objective"`
		BotID       string `json:"bot_id"`
		LandingID   string `json:"landing_id"`
		ManualLink  string `json:"manual_link"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	clientID := u.ClientID
	if u.Role == "admin" {
		clientID = r.URL.Query().Get("client_id")
		if clientID == "" {
			clientID = u.ClientID
		}
	}

	targetURL := s.Social.ResolveTargetURL(clientID, body.Objective, body.BotID, body.LandingID, body.ManualLink)

	post, err := s.Social.CreatePost(
		clientID,
		body.CampaignID,
		"facebook",
		body.Content,
		body.ImageURL,
		targetURL,
		"now",
		body.ImageMode,
		body.ImagePrompt,
		nil,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if err := s.Social.PublishNow(r.Context(), post.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "post_id": post.ID})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "post_id": post.ID})
}

func (s *Server) handleScheduleSocialPost(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body struct {
		CampaignID       string     `json:"campaign_id"`
		Content          string     `json:"content"`
		ImageURL         string     `json:"image_url"`
		ImageMode        string     `json:"image_mode"`
		ImagePrompt      string     `json:"image_prompt"`
		Objective        string     `json:"objective"`
		BotID            string     `json:"bot_id"`
		LandingID        string     `json:"landing_id"`
		ManualLink       string     `json:"manual_link"`
		PublishMode      string     `json:"publish_mode"`
		RecurringMinutes int        `json:"recurring_minutes"`
		ScheduledAt      *time.Time `json:"scheduled_at"`
		DaysOfWeek       string     `json:"days_of_week"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	clientID := u.ClientID
	if u.Role == "admin" {
		clientID = r.URL.Query().Get("client_id")
		if clientID == "" {
			clientID = u.ClientID
		}
	}

	targetURL := s.Social.ResolveTargetURL(clientID, body.Objective, body.BotID, body.LandingID, body.ManualLink)

	post, err := s.Social.CreatePost(
		clientID,
		body.CampaignID,
		"facebook",
		body.Content,
		body.ImageURL,
		targetURL,
		body.PublishMode,
		body.ImageMode,
		body.ImagePrompt,
		body.ScheduledAt,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	runAt := time.Now()
	jobType := "publish_once"
	if body.ScheduledAt != nil {
		runAt = *body.ScheduledAt
	}
	if body.PublishMode == "recurring" {
		jobType = "recurring_publish"
	}

	scheduler := services.NewSocialScheduler(s.DB, s.Social)
	if err := scheduler.CreateJob(clientID, body.CampaignID, post.ID, jobType, runAt, body.RecurringMinutes, body.DaysOfWeek); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	_, _ = s.DB.Exec(`UPDATE social_posts SET status='scheduled' WHERE id=?`, post.ID)

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "post_id": post.ID})
}

func (s *Server) handleSocialLogs(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	clientID := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		clientID = u.ClientID
	}

	rows, err := s.DB.Query(`
		SELECT id, client_id, campaign_id, post_id, level, message, created_at
		FROM social_logs
		WHERE (?='' OR client_id=?)
		ORDER BY created_at DESC
		LIMIT 100
	`, clientID, clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()

	var out []models.SocialLog
	for rows.Next() {
		var l models.SocialLog
		if err := rows.Scan(&l.ID, &l.ClientID, &l.CampaignID, &l.PostID, &l.Level, &l.Message, &l.CreatedAt); err == nil {
			out = append(out, l)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleSocialGenerateImage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Prompt string `json:"prompt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	url, err := s.Social.GenerateImage(r.Context(), body.Prompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"image_url": url})
}

func (s *Server) handleSocialUploadImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "no se pudo leer multipart form"})
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "archivo image requerido"})
		return
	}
	defer file.Close()

	dir := filepath.Join("data", "social_assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".png"
	}
	filename := uuid.NewString() + ext
	absPath := filepath.Join(dir, filename)

	out, err := os.Create(absPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	baseURL := strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	imageURL := "/social-assets/" + filename
	if baseURL != "" {
		imageURL = baseURL + imageURL
	}

	writeJSON(w, http.StatusOK, map[string]any{"image_url": imageURL})
}

func (s *Server) handleRegisterClient(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		CompanyName string `json:"company_name"`
		Phone       string `json:"phone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	body.Email = strings.TrimSpace(strings.ToLower(body.Email))
	body.Password = strings.TrimSpace(body.Password)
	body.CompanyName = strings.TrimSpace(body.CompanyName)
	body.Phone = strings.TrimSpace(body.Phone)

	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name required"})
		return
	}
	if body.CompanyName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "company_name required"})
		return
	}
	if body.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "email required"})
		return
	}
	if body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "password required"})
		return
	}

	// 1) crear cliente
	client, err := s.Manager.CreateClient(body.CompanyName, body.Email, body.Phone, "pro")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	// 2) crear usuario principal del cliente
	user, token, err := s.Auth.CreateUser(client.ID, body.Name, body.Email, body.Password, "client_admin")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"token":  token,
		"user":   user,
		"client": client,
	})
}