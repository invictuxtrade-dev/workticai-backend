package httpapi

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

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
	var a services.Agency
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	item, err := s.Agencies.Create(a)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, item)
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