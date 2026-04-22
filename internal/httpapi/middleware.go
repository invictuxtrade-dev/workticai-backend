package httpapi

import (
	"context"
	"net/http"
	"strings"

	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type ctxKey string

const userKey ctxKey = "user"

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))

		allowedOrigins := map[string]bool{
			"http://localhost:5173":     true,
			"https://app.workticai.com": true,
		}

		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// dejar pasar preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		h := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing bearer token"})
			return
		}

		tok := strings.TrimSpace(h[7:])
		user, err := s.Auth.GetUserByToken(tok)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid session"})
			return
		}

		ctx := context.WithValue(r.Context(), userKey, user)
		next(w, r.WithContext(ctx))
	}
}

func currentUser(r *http.Request) models.User {
	if v := r.Context().Value(userKey); v != nil {
		return v.(models.User)
	}
	return models.User{}
}

func requireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[role] = true
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			u := currentUser(r)
			if !allowed[u.Role] {
				writeJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
				return
			}
			next(w, r)
		}
	}
}