package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"whatsapp-sales-os-enterprise/backend/internal/services"
)

func (s *Server) handleListPaymentLinks(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	clientID := r.URL.Query().Get("client_id")
	if u.Role != "admin" {
		clientID = u.ClientID
	}

	items, err := s.PaymentLinks.List(clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreatePaymentLink(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var body services.PaymentLink
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if u.Role != "admin" {
		body.ClientID = u.ClientID
	}

	body.CreatedBy = u.ID

	publicBaseURL := strings.TrimRight(os.Getenv("PUBLIC_APP_URL"), "/")
	if publicBaseURL == "" {
		publicBaseURL = strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	}
	if publicBaseURL == "" {
		publicBaseURL = "https://app.workticai.com"
	}

	item, err := s.PaymentLinks.Create(body, publicBaseURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleApprovePaymentLink(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	u := currentUser(r)

	if err := s.PaymentLinks.Approve(id, u.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	link, err := s.PaymentLinks.Get(id)
	if err == nil && link.PaymentScope == "agency_license" && strings.TrimSpace(link.AgencyID) != "" {
		if s.Agencies != nil {
			_ = s.Agencies.Activate(link.AgencyID, 1)
		}
	}

	updated, _ := s.PaymentLinks.Get(id)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleRejectPaymentLink(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var body struct {
		Reason string `json:"reason"`
	}

	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := s.PaymentLinks.Reject(id, body.Reason); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	updated, _ := s.PaymentLinks.Get(id)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handlePublicPaymentLink(w http.ResponseWriter, r *http.Request) {
	item, err := s.PaymentLinks.Get(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "payment link not found"})
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleSubmitPaymentLinkTx(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var body struct {
		CustomerName  string `json:"customer_name"`
		CustomerEmail string `json:"customer_email"`
		CustomerPhone string `json:"customer_phone"`
		TxHash        string `json:"tx_hash"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if err := s.PaymentLinks.SubmitTx(
		id,
		body.CustomerName,
		body.CustomerEmail,
		body.CustomerPhone,
		body.TxHash,
	); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	item, _ := s.PaymentLinks.Get(id)
	writeJSON(w, http.StatusOK, item)
}