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

	// Público para visualizar landings compartibles
	r.HandleFunc("/l/{id}", s.handlePublicLanding).Methods("GET", "OPTIONS")

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
	secured.HandleFunc("/social/instagram/verify", s.handleVerifyInstagram).Methods("POST", "OPTIONS")
	secured.HandleFunc("/social/instagram/data", s.handleInstagramData).Methods("GET", "OPTIONS")
	secured.HandleFunc("/social/publish-multi", s.handlePublishMulti).Methods("POST", "OPTIONS")
	

	secured.HandleFunc("/plans", s.handlePlans).Methods("GET", "OPTIONS")
	secured.HandleFunc("/billing/config", requireRole("admin")(s.handleGetBillingConfig)).Methods("GET", "OPTIONS")
	secured.HandleFunc("/billing/config", requireRole("admin")(s.handleUpdateBillingConfig)).Methods("PUT", "OPTIONS")
	secured.HandleFunc("/subscriptions/current", s.handleCurrentSubscription).Methods("GET", "OPTIONS")
	secured.HandleFunc("/subscriptions/select", s.handleSelectPlan).Methods("POST", "OPTIONS")
	secured.HandleFunc("/subscriptions/pay", s.handleSubmitTxHash).Methods("POST", "OPTIONS")
	secured.HandleFunc("/subscriptions/pending", requireRole("admin")(s.handlePendingSubscriptions)).Methods("GET", "OPTIONS")
	secured.HandleFunc("/subscriptions/{id}/approve", requireRole("admin")(s.handleApproveSubscription)).Methods("POST", "OPTIONS")

	secured.HandleFunc("/ads/campaigns", s.handleListAdsCampaigns).Methods("GET", "OPTIONS")
	secured.HandleFunc("/ads/generate-campaign", s.handleGenerateAdsCampaign).Methods("POST", "OPTIONS")
	secured.HandleFunc("/ads/ecosystem", s.handleCreateAdsEcosystem).Methods("POST", "OPTIONS")
	secured.HandleFunc("/ads/campaigns", s.handleCreateAdsCampaign).Methods("POST", "OPTIONS")
	secured.HandleFunc("/ads/campaigns/{id}/status", s.handleUpdateAdsCampaignStatus).Methods("PATCH", "OPTIONS")

	secured.HandleFunc("/groups/whatsapp-bots", s.handleListGroupBots).Methods("GET", "OPTIONS")
	secured.HandleFunc("/groups/whatsapp-bots", s.handleCreateGroupBot).Methods("POST", "OPTIONS")
	secured.HandleFunc("/groups/whatsapp-bots/{id}", s.handleUpdateGroupBot).Methods("PUT", "OPTIONS")
	secured.HandleFunc("/groups/whatsapp-bots/{id}", s.handleDeleteGroupBot).Methods("DELETE", "OPTIONS")

	secured.HandleFunc("/groups/facebook-targets", s.handleListFacebookGroupTargets).Methods("GET", "OPTIONS")
	secured.HandleFunc("/groups/facebook-targets", s.handleCreateFacebookGroupTarget).Methods("POST", "OPTIONS")
	secured.HandleFunc("/groups/facebook-targets/{id}", s.handleUpdateFacebookGroupTarget).Methods("PUT", "OPTIONS")
	secured.HandleFunc("/groups/facebook-targets/{id}", s.handleDeleteFacebookGroupTarget).Methods("DELETE", "OPTIONS")
	secured.HandleFunc("/groups/facebook-discover", s.handleDiscoverFacebookGroups).Methods("POST", "OPTIONS")

	secured.HandleFunc("/groups/growth-settings", s.handleGetGroupGrowthSettings).Methods("GET", "OPTIONS")
	secured.HandleFunc("/groups/growth-settings", s.handleSaveGroupGrowthSettings).Methods("PUT", "OPTIONS")

	secured.HandleFunc("/groups/facebook-join-queue", s.handleListFacebookJoinQueue).Methods("GET", "OPTIONS")
	secured.HandleFunc("/groups/facebook-targets/{id}/request-join", s.handleRequestFacebookGroupJoin).Methods("POST", "OPTIONS")
	secured.HandleFunc("/groups/facebook-targets/{id}/mark-joined", s.handleMarkFacebookGroupJoined).Methods("POST", "OPTIONS")
	secured.HandleFunc("/groups/facebook-join-queue/{id}/status", s.handleUpdateFacebookJoinQueueStatus).Methods("PATCH", "OPTIONS")
	secured.HandleFunc("/groups/facebook-logs", s.handleListFacebookGroupLogs).Methods("GET", "OPTIONS")

	secured.HandleFunc("/assistant/messages", s.handleAssistantMessages).Methods("GET", "OPTIONS")
	secured.HandleFunc("/assistant/messages", s.handleClearAssistantMessages).Methods("DELETE", "OPTIONS")
	secured.HandleFunc("/assistant/chat", s.handleAssistantChat).Methods("POST", "OPTIONS")
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

	// 🔥 AQUÍ VIENE LO QUE TE FALTABA
	igID, igUser, _ := s.Social.GetInstagramFromPage(c.AccessToken, c.PageID)

	resp := map[string]any{
		"id": c.ID,
		"client_id": c.ClientID,
		"platform": c.Platform,
		"access_token": c.AccessToken,
		"page_id": c.PageID,
		"page_name": c.PageName,
		"enabled": c.Enabled,
		"ad_account_id": c.AdAccountID,

		// 🔥 ESTO ES LO CLAVE
		"instagram_connected": igID != "",
		"instagram_id": igID,
		"instagram_username": igUser,
	}

	writeJSON(w, http.StatusOK, resp)
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
	client, err := s.Manager.CreateClient(body.CompanyName, body.Email, body.Phone, "")
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

func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	plans, err := s.Billing.ListPlans()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, plans)
}

func (s *Server) handleGetBillingConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.Billing.GetPlanConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleUpdateBillingConfig(w http.ResponseWriter, r *http.Request) {
	var cfg models.PlanConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if err := s.Billing.UpdatePlanConfig(cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleCurrentSubscription(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	sub, err := s.Billing.GetLatestSubscription(u.ClientID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"subscription": nil})
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (s *Server) handleSelectPlan(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var body struct {
		PlanSlug     string `json:"plan_slug"`
		BillingCycle string `json:"billing_cycle"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if body.BillingCycle == "" {
		body.BillingCycle = "monthly"
	}
	sub, err := s.Billing.SelectPlan(u.ClientID, body.PlanSlug, body.BillingCycle)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

func (s *Server) handleSubmitTxHash(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SubscriptionID string `json:"subscription_id"`
		TxHash         string `json:"tx_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(body.SubscriptionID) == "" || strings.TrimSpace(body.TxHash) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "subscription_id and tx_hash required"})
		return
	}
	if err := s.Billing.SubmitTxHash(body.SubscriptionID, body.TxHash); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handlePendingSubscriptions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`
		SELECT id, client_id, plan_id, plan_slug, status, billing_cycle, amount, payment_method, tx_hash, wallet_address,
		       paid_at, starts_at, expires_at, validated_by, validation_notes, created_at, updated_at
		FROM subscriptions
		WHERE status='pending'
		ORDER BY created_at DESC
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []models.Subscription{}
	for rows.Next() {
		var ssub models.Subscription
		if err := rows.Scan(
			&ssub.ID, &ssub.ClientID, &ssub.PlanID, &ssub.PlanSlug, &ssub.Status, &ssub.BillingCycle, &ssub.Amount, &ssub.PaymentMethod, &ssub.TxHash, &ssub.WalletAddress,
			&ssub.PaidAt, &ssub.StartsAt, &ssub.ExpiresAt, &ssub.ValidatedBy, &ssub.ValidationNotes, &ssub.CreatedAt, &ssub.UpdatedAt,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		out = append(out, ssub)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleApproveSubscription(w http.ResponseWriter, r *http.Request) {
	admin := currentUser(r)
	var body struct {
		Notes string `json:"notes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	id := mux.Vars(r)["id"]
	if err := s.Billing.ApproveSubscription(id, admin.ID, body.Notes); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleGenerateAdsCampaign(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body struct {
		BusinessName  string  `json:"business_name"`
		Product       string  `json:"product"`
		Offer         string  `json:"offer"`
		Target        string  `json:"target"`
		Country       string  `json:"country"`
		BudgetDaily   float64 `json:"budget_daily"`
		TicketAverage float64 `json:"ticket_average"`
		Save          bool    `json:"save"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	body.BusinessName = strings.TrimSpace(body.BusinessName)
	body.Product = strings.TrimSpace(body.Product)
	body.Offer = strings.TrimSpace(body.Offer)
	body.Target = strings.TrimSpace(body.Target)
	body.Country = strings.TrimSpace(body.Country)

	if body.Product == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "product required"})
		return
	}
	if body.Offer == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "offer required"})
		return
	}
	if body.Target == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "target required"})
		return
	}

	clientID := u.ClientID
	if u.Role == "admin" {
		clientID = r.URL.Query().Get("client_id")
	}

	plan, err := s.Ads.GenerateCampaignPlan(
		r.Context(),
		body.BusinessName,
		body.Product,
		body.Offer,
		body.Target,
		body.Country,
		body.BudgetDaily,
		body.TicketAverage,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	var campaignID string
	if body.Save {
		b, _ := json.Marshal(plan)
		campaignID, err = s.Ads.SaveCampaign(clientID, plan, string(b))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"campaign_id": campaignID,
		"plan":        plan,
	})
}

func (s *Server) handleCreateAdsCampaign(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body struct {
		Plan services.AdsCampaignPlan `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	clientID := u.ClientID
	if u.Role == "admin" {
		clientID = r.URL.Query().Get("client_id")
	}

	b, _ := json.Marshal(body.Plan)

	id, err := s.Ads.SaveCampaign(clientID, body.Plan, string(b))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleListAdsCampaigns(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	clientID := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		clientID = u.ClientID
	}

	rows, err := s.DB.Query(`
		SELECT id, client_id, name, objective, product, offer, target_audience,
		       budget_daily, budget_monthly, status, ai_plan_json, created_at, updated_at
		FROM ads_campaigns
		WHERE (?='' OR client_id=?)
		ORDER BY created_at DESC
	`, clientID, clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []map[string]any{}

	for rows.Next() {
		var id, cid, name, objective, product, offer, targetAudience, status, aiPlanJSON string
		var budgetDaily, budgetMonthly float64
		var createdAt, updatedAt time.Time

		if err := rows.Scan(
			&id, &cid, &name, &objective, &product, &offer, &targetAudience,
			&budgetDaily, &budgetMonthly, &status, &aiPlanJSON, &createdAt, &updatedAt,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		var plan any
		_ = json.Unmarshal([]byte(aiPlanJSON), &plan)

		out = append(out, map[string]any{
			"id":              id,
			"client_id":       cid,
			"name":            name,
			"objective":       objective,
			"product":         product,
			"offer":           offer,
			"target_audience": targetAudience,
			"budget_daily":    budgetDaily,
			"budget_monthly":  budgetMonthly,
			"status":          status,
			"plan":            plan,
			"created_at":      createdAt,
			"updated_at":      updatedAt,
		})
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleUpdateAdsCampaignStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var body struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	status := strings.TrimSpace(strings.ToLower(body.Status))
	if status == "" {
		status = "draft"
	}

	_, err := s.DB.Exec(`
		UPDATE ads_campaigns
		SET status=?, updated_at=?
		WHERE id=?
	`, status, time.Now(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
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

func (s *Server) handleCreateAdsEcosystem(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body struct {
		BusinessName  string  `json:"business_name"`
		Product       string  `json:"product"`
		Offer         string  `json:"offer"`
		Target        string  `json:"target"`
		Country       string  `json:"country"`
		BudgetDaily   float64 `json:"budget_daily"`
		TicketAverage float64 `json:"ticket_average"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	body.BusinessName = strings.TrimSpace(body.BusinessName)
	body.Product = strings.TrimSpace(body.Product)
	body.Offer = strings.TrimSpace(body.Offer)
	body.Target = strings.TrimSpace(body.Target)
	body.Country = strings.TrimSpace(body.Country)

	if body.Product == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "product required"})
		return
	}
	if body.Offer == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "offer required"})
		return
	}
	if body.Target == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "target required"})
		return
	}

	clientID := u.ClientID
	if u.Role == "admin" {
		clientID = r.URL.Query().Get("client_id")
		if clientID == "" {
			clientID = u.ClientID
		}
	}

	if strings.TrimSpace(clientID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id required"})
		return
	}

	// 1) Generar plan de campaña
	plan, err := s.Ads.GenerateCampaignPlan(
		r.Context(),
		body.BusinessName,
		body.Product,
		body.Offer,
		body.Target,
		body.Country,
		body.BudgetDaily,
		body.TicketAverage,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// 2) Crear bot automático
	botName := "Bot - " + plan.Product
	if strings.TrimSpace(plan.Name) != "" {
		botName = "Bot - " + plan.Name
	}

	bot, err := s.Manager.CreateBot(clientID, botName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// 3) Configurar bot desde la campaña
	cfg := models.BotConfig{
		BotID:               bot.ID,
		SystemPrompt:        buildAdsBotPrompt(plan),
		BusinessName:        body.BusinessName,
		BusinessDescription: plan.ValueProposition,
		Offer:               plan.Offer,
		TargetAudience:      plan.TargetAudience,
		Tone:                "Profesional, humano, asesor y vendedor",
		CTAButtonText:       plan.CTA,
		CTALink:             "",
		FallbackMessage:     "Gracias por escribirnos. En breve te ayudo con toda la información.",
		HumanHandoffPhone:   "",
		Temperature:         0.7,
		Model:               "gpt-4o-mini",
		FollowupEnabled:     true,
		FollowupDelayMins:   60,
		ReplyMode:           "manual",
		TemplateID:          "",
	}

	_, _ = s.Manager.UpsertBotConfig(cfg)

	// 4) Generar landing automática
	landingPrompt := buildAdsLandingPrompt(plan)

	lp, err := s.Manager.Landing.GenerateLanding(
		r.Context(),
		bot,
		cfg,
		models.LandingPage{
			ClientID:        clientID,
			BotID:           bot.ID,
			Name:            "Landing - " + plan.Product,
			Prompt:          landingPrompt,
			Status:          "generated",
			StylePreset:     "dark_premium",
			PrimaryColor:    "#2563eb",
			SecondaryColor:  "#0f172a",
			ShowVideo:       false,
			ShowImage:       false,
			TrackingMode:    "auto",
			TrackingBaseURL: "",
		},
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if lp.ID == "" {
		lp.ID = uuid.NewString()
	}
	if lp.ClientID == "" {
		lp.ClientID = clientID
	}
	if lp.BotID == "" {
		lp.BotID = bot.ID
	}
	if lp.Status == "" {
		lp.Status = "generated"
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

	// 5) Guardar campaña asociada al bot y landing
	raw, _ := json.Marshal(plan)

	campaignID, err := s.Ads.SaveCampaignEcosystem(
		clientID,
		plan,
		string(raw),
		bot.ID,
		lp.ID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"success":          true,
		"campaign_id":      campaignID,
		"bot_id":           bot.ID,
		"landing_id":       lp.ID,
		"ecosystem_status": "ready_pending_whatsapp",
		"plan":             plan,
		"bot":              bot,
		"landing":          lp,
	})
}

func buildAdsBotPrompt(plan services.AdsCampaignPlan) string {
	var b strings.Builder

	b.WriteString("Eres un asesor comercial experto por WhatsApp.\n")
	b.WriteString("Tu tarea es atender leads provenientes de una campaña publicitaria y convertirlos en clientes.\n\n")

	b.WriteString("PRODUCTO/SERVICIO:\n")
	b.WriteString(plan.Product + "\n\n")

	b.WriteString("OFERTA:\n")
	b.WriteString(plan.Offer + "\n\n")

	b.WriteString("PÚBLICO OBJETIVO:\n")
	b.WriteString(plan.TargetAudience + "\n\n")

	b.WriteString("PROPUESTA DE VALOR:\n")
	b.WriteString(plan.ValueProposition + "\n\n")

	b.WriteString("DOLORES DEL CLIENTE:\n")
	for _, p := range plan.PainPoints {
		b.WriteString("- " + p + "\n")
	}

	b.WriteString("\nÁNGULOS DE VENTA:\n")
	for _, a := range plan.Angles {
		b.WriteString("- " + a + "\n")
	}

	b.WriteString("\nGUION BASE DE WHATSAPP:\n")
	b.WriteString(plan.WhatsAppScript + "\n\n")

	b.WriteString("REGLAS:\n")
	b.WriteString("- Responde de forma humana, breve y natural.\n")
	b.WriteString("- Haz preguntas para entender la necesidad del cliente.\n")
	b.WriteString("- No prometas resultados garantizados.\n")
	b.WriteString("- Maneja objeciones con calma.\n")
	b.WriteString("- Lleva al usuario hacia la acción principal.\n")
	b.WriteString("- Si el usuario pide hablar con un humano, deriva amablemente.\n")

	return b.String()
}

func buildAdsLandingPrompt(plan services.AdsCampaignPlan) string {
	var b strings.Builder

	b.WriteString("Crear una landing page de alta conversión para esta campaña.\n\n")

	b.WriteString("Nombre campaña: " + plan.Name + "\n")
	b.WriteString("Producto: " + plan.Product + "\n")
	b.WriteString("Oferta: " + plan.Offer + "\n")
	b.WriteString("Público: " + plan.TargetAudience + "\n")
	b.WriteString("Propuesta de valor: " + plan.ValueProposition + "\n\n")

	b.WriteString("Copy principal:\n")
	b.WriteString(plan.PrimaryText + "\n\n")

	b.WriteString("Headline:\n")
	b.WriteString(plan.Headline + "\n\n")

	b.WriteString("Descripción:\n")
	b.WriteString(plan.Description + "\n\n")

	b.WriteString("Dolores:\n")
	for _, p := range plan.PainPoints {
		b.WriteString("- " + p + "\n")
	}

	b.WriteString("\nÁngulos:\n")
	for _, a := range plan.Angles {
		b.WriteString("- " + a + "\n")
	}

	b.WriteString("\nEstructura recomendada del funnel:\n")
	for _, s := range plan.Funnel.LandingStructure {
		b.WriteString("- " + s + "\n")
	}

	b.WriteString("\nCTA principal: " + plan.CTA + "\n")
	b.WriteString("Destino principal: WhatsApp.\n")
	b.WriteString("La landing debe ser profesional, persuasiva, responsive y orientada a conversión.\n")

	return b.String()
}

func (s *Server) handleListGroupBots(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	clientID := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		clientID = u.ClientID
	}

	items, err := s.Groups.ListGroupBots(clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateGroupBot(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body services.GroupBot
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if u.Role != "admin" {
		body.ClientID = u.ClientID
	}

	item, err := s.Groups.CreateGroupBot(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateGroupBot(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var body services.GroupBot
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if err := s.Groups.UpdateGroupBot(id, body); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleDeleteGroupBot(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if err := s.Groups.DeleteGroupBot(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleListFacebookGroupTargets(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	clientID := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		clientID = u.ClientID
	}

	items, err := s.Groups.ListFacebookGroupTargets(clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateFacebookGroupTarget(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body services.FacebookGroupTarget
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if u.Role != "admin" {
		body.ClientID = u.ClientID
	}

	item, err := s.Groups.SaveFacebookGroupTarget(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateFacebookGroupTarget(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var body services.FacebookGroupTarget
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if err := s.Groups.UpdateFacebookGroupTarget(id, body); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleDeleteFacebookGroupTarget(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if err := s.Groups.DeleteFacebookGroupTarget(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleDiscoverFacebookGroups(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body services.FacebookGroupDiscoveryRequest
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

	out, err := s.Groups.DiscoverFacebookGroups(r.Context(), body, clientID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *Server) groupClientID(r *http.Request) string {
	u := currentUser(r)
	clientID := u.ClientID

	if u.Role == "admin" && r.URL.Query().Get("client_id") != "" {
		clientID = r.URL.Query().Get("client_id")
	}

	return clientID
}

func (s *Server) handleGetGroupGrowthSettings(w http.ResponseWriter, r *http.Request) {
	clientID := s.groupClientID(r)

	settings, err := s.Groups.GetGrowthSettings(clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleSaveGroupGrowthSettings(w http.ResponseWriter, r *http.Request) {
	clientID := s.groupClientID(r)

	var body services.GroupGrowthSettings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	body.ClientID = clientID

	settings, err := s.Groups.SaveGrowthSettings(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleListFacebookJoinQueue(w http.ResponseWriter, r *http.Request) {
	clientID := s.groupClientID(r)

	items, err := s.Groups.ListJoinQueue(clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleRequestFacebookGroupJoin(w http.ResponseWriter, r *http.Request) {
	clientID := s.groupClientID(r)
	groupID := mux.Vars(r)["id"]

	var body struct {
		Mode string `json:"mode"` // manual | auto
	}

	_ = json.NewDecoder(r.Body).Decode(&body)

	item, err := s.Groups.RequestFacebookGroupJoin(clientID, groupID, body.Mode)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleMarkFacebookGroupJoined(w http.ResponseWriter, r *http.Request) {
	clientID := s.groupClientID(r)
	groupID := mux.Vars(r)["id"]

	if err := s.Groups.MarkFacebookGroupJoined(clientID, groupID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleUpdateFacebookJoinQueueStatus(w http.ResponseWriter, r *http.Request) {
	clientID := s.groupClientID(r)
	queueID := mux.Vars(r)["id"]

	var body struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if err := s.Groups.UpdateJoinQueueStatus(clientID, queueID, body.Status, body.Message); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleListFacebookGroupLogs(w http.ResponseWriter, r *http.Request) {
	clientID := s.groupClientID(r)
	groupID := r.URL.Query().Get("group_target_id")

	items, err := s.Groups.ListFacebookGroupLogs(clientID, groupID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handlePublicLanding(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var html string
	var previewHTML string
	var status string

	err := s.DB.QueryRow(`
		SELECT html, preview_html, status
		FROM landing_pages
		WHERE id=?
	`, id).Scan(&html, &previewHTML, &status)

	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`
			<!doctype html>
			<html>
			<head>
				<meta charset="utf-8">
				<meta name="viewport" content="width=device-width, initial-scale=1">
				<title>Landing no encontrada</title>
				<style>
					body{font-family:Arial,sans-serif;background:#0f172a;color:white;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;text-align:center}
					.card{max-width:520px;padding:32px;background:#111827;border-radius:20px}
				</style>
			</head>
			<body>
				<div class="card">
					<h1>Landing no encontrada</h1>
					<p>Esta landing no existe o ya no está disponible.</p>
				</div>
			</body>
			</html>
		`))
		return
	}

	page := strings.TrimSpace(html)
	if page == "" {
		page = strings.TrimSpace(previewHTML)
	}

	if page == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("<h1>Landing vacía</h1>"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(page))
}

func (s *Server) assistantClientID(r *http.Request) string {
	u := currentUser(r)

	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	if clientID != "" {
		return clientID
	}

	if strings.TrimSpace(u.ClientID) != "" {
		return u.ClientID
	}

	return ""
}

func (s *Server) handleAssistantMessages(w http.ResponseWriter, r *http.Request) {
	clientID := s.assistantClientID(r)
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Selecciona un cliente primero"})
		return
	}

	items, err := s.Assistant.ListMessages(clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleClearAssistantMessages(w http.ResponseWriter, r *http.Request) {
	clientID := s.assistantClientID(r)
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Selecciona un cliente primero"})
		return
	}

	if err := s.Assistant.ClearMessages(clientID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleAssistantChat(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	clientID := s.assistantClientID(r)
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Selecciona un cliente primero"})
		return
	}

	var body services.AssistantChatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	msg, err := s.Assistant.Chat(r.Context(), clientID, u.Name, body.Message)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

func (s *Server) handleVerifyInstagram(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = u.ClientID
	}

	var cred models.SocialCredential
	err := s.DB.QueryRow(`
		SELECT id, access_token, page_id
		FROM social_credentials
		WHERE client_id=? AND platform='facebook'
		LIMIT 1
	`, clientID).Scan(&cred.ID, &cred.AccessToken, &cred.PageID)

	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "No hay credenciales de Facebook"})
		return
	}

	igID, igUser, err := s.Social.GetInstagramFromPage(cred.AccessToken, cred.PageID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if igID == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"connected": false,
			"message": "No hay cuenta de Instagram conectada a esta página",
		})
		return
	}

	_, _ = s.DB.Exec(`
		UPDATE social_credentials
		SET instagram_account_id=?, instagram_username=?, instagram_connected=1
		WHERE id=?
	`, igID, igUser, cred.ID)

	writeJSON(w, http.StatusOK, map[string]any{
		"connected": true,
		"instagram_account_id": igID,
		"instagram_username": igUser,
	})
}

func (s *Server) handleInstagramData(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	cred, err := s.Social.GetCredentialByClient(u.ClientID)
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": "credenciales no encontradas"})
		return
	}

	igID, username, err := s.Social.GetInstagramFromPage(cred.AccessToken, cred.PageID)
	if err != nil || igID == "" {
		writeJSON(w, 400, map[string]any{"error": "Instagram no conectado"})
		return
	}

	url := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/%s?fields=username,followers_count,media_count&access_token=%s",
		igID,
		cred.AccessToken,
	)

	resp, _ := http.Get(url)
	defer resp.Body.Close()

	var data map[string]any
	json.NewDecoder(resp.Body).Decode(&data)

	data["instagram_id"] = igID
	data["instagram_username"] = username

	writeJSON(w, 200, data)
}

func (s *Server) publishInstagram(accessToken, igID, imageURL, caption string) error {

	// 1. Crear contenedor
	createURL := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/%s/media?image_url=%s&caption=%s&access_token=%s",
		igID,
		url.QueryEscape(imageURL),
		url.QueryEscape(caption),
		accessToken,
	)

	resp, err := http.Post(createURL, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res map[string]any
	json.NewDecoder(resp.Body).Decode(&res)

	creationID := res["id"].(string)

	// 2. Publicar
	publishURL := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/%s/media_publish?creation_id=%s&access_token=%s",
		igID,
		creationID,
		accessToken,
	)

	_, err = http.Post(publishURL, "application/json", nil)
	return err
}

func (s *Server) handlePublishMulti(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body struct {
		Platforms []string `json:"platforms"`
		Content   string   `json:"content"`
		ImageURL  string   `json:"image_url"`
	}

	json.NewDecoder(r.Body).Decode(&body)

	cred, err := s.Social.GetCredentialByClient(u.ClientID)
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": "credenciales no encontradas"})
		return
	}

	results := map[string]any{}

	// FACEBOOK
	for _, p := range body.Platforms {
		if p == "facebook" {
			_, err := s.Social.Publisher.PublishFacebookPost(
				r.Context(),
				u.ClientID,
				body.Content,
				body.ImageURL,
				"",
			)
			results["facebook"] = err == nil
		}
	}

	// INSTAGRAM
	for _, p := range body.Platforms {
		if p == "instagram" {
			igID, _, _ := s.Social.GetInstagramFromPage(cred.AccessToken, cred.PageID)

			err := s.publishInstagram(
				cred.AccessToken,
				igID,
				body.ImageURL,
				body.Content,
			)

			results["instagram"] = err == nil
		}
	}

	writeJSON(w, 200, results)
}
