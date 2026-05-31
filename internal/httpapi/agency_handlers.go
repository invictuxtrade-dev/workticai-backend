package httpapi

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"whatsapp-sales-os-enterprise/backend/internal/services"
)

func (s *Server) handleListAgencies(w http.ResponseWriter, r *http.Request) {
	items, err := s.Agencies.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateAgency(w http.ResponseWriter, r *http.Request) {
	var body struct {
		services.Agency

		AdminName     string `json:"admin_name"`
		AdminEmail    string `json:"admin_email"`
		AdminPassword string `json:"admin_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid json",
		})
		return
	}

	agencyInput := body.Agency

	item, err := s.Agencies.Create(agencyInput)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": err.Error(),
		})
		return
	}

	adminEmail := strings.TrimSpace(body.AdminEmail)
	if adminEmail == "" {
		adminEmail = strings.TrimSpace(item.Email)
	}

	if adminEmail == "" {
		writeJSON(w, http.StatusCreated, map[string]any{
			"agency": item,
			"warning": "Agencia creada, pero no se creó usuario agency_admin porque falta email.",
		})
		return
	}

	adminName := strings.TrimSpace(body.AdminName)
	if adminName == "" {
		adminName = item.Name + " Admin"
	}

	adminPassword := strings.TrimSpace(body.AdminPassword)
	if adminPassword == "" {
		adminPassword = "Agency-" + uuid.NewString()[:8]
	}

	now := item.CreatedAt
	if now.IsZero() {
		// fallback por si la fecha viene vacía
		now = item.UpdatedAt
	}

	clientID := uuid.NewString()

	_, err = s.DB.Exec(`
		INSERT INTO clients (
			id, agency_id, name, email, phone, plan, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`,
		clientID,
		item.ID,
		item.Name,
		item.Email,
		item.Phone,
		item.PlanEquivalent,
		"active",
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "agency created but primary client failed: " + err.Error(),
		})
		return
	}

	adminUser, _, err := s.Auth.CreateUserWithAgency(
		clientID,
		item.ID,
		adminName,
		adminEmail,
		adminPassword,
		"agency_admin",
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "agency and primary client created but agency admin failed: " + err.Error(),
		})
		return
	}

	loginURL := "/a/" + strings.TrimSpace(item.Subdomain)
	if strings.TrimSpace(item.Subdomain) == "" {
		loginURL = "/"
	}

	_ = s.Agencies.SaveAccessData(
	item.ID,
	adminName,
	adminEmail,
	adminPassword,
		)

	writeJSON(w, http.StatusCreated, map[string]any{
		"agency": item,
		"primary_client_id": clientID,
		"agency_admin": adminUser,
		"temporary_password": adminPassword,
		"login_url": loginURL,
		"contract_url": "/agency-contract/" + item.ID,
	})
}

func (s *Server) handleUpdateAgency(w http.ResponseWriter, r *http.Request) {
	var a services.Agency
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	a.ID = mux.Vars(r)["id"]

	if err := s.Agencies.Update(a); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	item, _ := s.Agencies.Get(a.ID)
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDeleteAgency(w http.ResponseWriter, r *http.Request) {
	if err := s.Agencies.Delete(mux.Vars(r)["id"]); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleGetAgency(w http.ResponseWriter, r *http.Request) {
	item, err := s.Agencies.Get(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "agency not found"})
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleActivateAgency(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Months int `json:"months"`
	}

	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := s.Agencies.Activate(mux.Vars(r)["id"], body.Months); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	item, _ := s.Agencies.Get(mux.Vars(r)["id"])
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleSuspendAgency(w http.ResponseWriter, r *http.Request) {
	if err := s.Agencies.Suspend(mux.Vars(r)["id"]); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	item, _ := s.Agencies.Get(mux.Vars(r)["id"])
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleAgencyPrices(w http.ResponseWriter, r *http.Request) {
	items, err := s.Agencies.Prices(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleSaveAgencyPrices(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Prices []services.AgencyPlanPrice `json:"prices"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	agencyID := mux.Vars(r)["id"]

	for i := range body.Prices {
		body.Prices[i].AgencyID = agencyID
	}

	if err := s.Agencies.SavePrices(agencyID, body.Prices); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	items, _ := s.Agencies.Prices(agencyID)
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handlePublicAgencyContract(w http.ResponseWriter, r *http.Request) {
	item, err := s.Agencies.Get(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "agency not found"})
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleSignAgencyContract(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SignedBy    string `json:"signed_by"`
		SignedEmail string `json:"signed_email"`
		Signature   string `json:"signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if strings.TrimSpace(ip) == "" {
		ip = r.RemoteAddr
	}

	if err := s.Agencies.SignContract(
		mux.Vars(r)["id"],
		body.SignedBy,
		body.SignedEmail,
		body.Signature,
		ip,
	); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	item, _ := s.Agencies.Get(mux.Vars(r)["id"])
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleMyAgencyBranding(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	if strings.TrimSpace(u.AgencyID) == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"has_agency": false,
		})
		return
	}

	item, err := s.Agencies.Get(u.AgencyID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"has_agency": false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"has_agency": true,
		"agency":     item,
	})
}

func (s *Server) handlePublicAgencyBySlug(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]

	item, err := s.Agencies.GetBySubdomain(slug)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "agency not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleAgencyAccess(w http.ResponseWriter, r *http.Request) {
	agencyID := mux.Vars(r)["id"]

	item, err := s.Agencies.Get(agencyID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "agency not found",
		})
		return
	}

	loginURL := "/"
	if strings.TrimSpace(item.Subdomain) != "" {
		loginURL = "/a/" + strings.TrimSpace(item.Subdomain)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"agency_id": item.ID,
		"agency_name": item.Name,
		"admin_name": item.AdminName,
		"admin_email": item.AdminEmail,
		"temporary_password": item.LastTempPassword,
		"login_url": loginURL,
		"contract_url": "/agency-contract/" + item.ID,
		"last_password_reset": item.LastPasswordReset,
	})
}

func (s *Server) handleRegenerateAgencyAccess(w http.ResponseWriter, r *http.Request) {
	agencyID := mux.Vars(r)["id"]

	item, err := s.Agencies.Get(agencyID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "agency not found",
		})
		return
	}

	adminEmail := strings.TrimSpace(item.AdminEmail)
	if adminEmail == "" {
		adminEmail = strings.TrimSpace(item.Email)
	}

	if adminEmail == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "agency admin email not configured",
		})
		return
	}

	adminName := strings.TrimSpace(item.AdminName)
	if adminName == "" {
		adminName = item.Name + " Admin"
	}

	newPassword := "Agency-" + uuid.NewString()[:8]

	adminUser, err := s.Auth.ResetAgencyAdminPassword(
		item.ID,
		adminEmail,
		newPassword,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "could not reset agency admin password: " + err.Error(),
		})
		return
	}

	_ = s.Agencies.SaveAccessData(
		item.ID,
		adminName,
		adminEmail,
		newPassword,
	)

	loginURL := "/"
	if strings.TrimSpace(item.Subdomain) != "" {
		loginURL = "/a/" + strings.TrimSpace(item.Subdomain)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"agency": item,
		"agency_admin": adminUser,
		"temporary_password": newPassword,
		"login_url": loginURL,
		"contract_url": "/agency-contract/" + item.ID,
	})
}