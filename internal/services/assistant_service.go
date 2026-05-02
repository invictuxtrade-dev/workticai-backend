package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AssistantService struct {
	DB *sql.DB
	AI *AIService
}

func NewAssistantService(db *sql.DB, ai *AIService) *AssistantService {
	return &AssistantService{DB: db, AI: ai}
}

type AssistantMessage struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type AssistantChatRequest struct {
	Message string `json:"message"`
}

func (s *AssistantService) ListMessages(clientID string) ([]AssistantMessage, error) {
	rows, err := s.DB.Query(`
		SELECT id, client_id, role, content, created_at
		FROM assistant_messages
		WHERE client_id=?
		ORDER BY created_at ASC
		LIMIT 80
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AssistantMessage{}
	for rows.Next() {
		var m AssistantMessage
		if err := rows.Scan(&m.ID, &m.ClientID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *AssistantService) ClearMessages(clientID string) error {
	_, err := s.DB.Exec(`DELETE FROM assistant_messages WHERE client_id=?`, clientID)
	return err
}

func (s *AssistantService) saveMessage(clientID, role, content string) error {
	_, err := s.DB.Exec(`
		INSERT INTO assistant_messages (id, client_id, role, content, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, uuid.NewString(), clientID, role, content, time.Now())
	return err
}

func (s *AssistantService) Chat(ctx context.Context, clientID, userName, message string) (AssistantMessage, error) {
	message = strings.TrimSpace(message)
	if clientID == "" {
		return AssistantMessage{}, fmt.Errorf("client_id required")
	}
	if message == "" {
		return AssistantMessage{}, fmt.Errorf("message required")
	}
	if s.AI == nil {
		return AssistantMessage{}, fmt.Errorf("ai service not configured")
	}

	_ = s.saveMessage(clientID, "user", message)

	history, _ := s.ListMessages(clientID)

	system := `Eres Worktic AI Assistant, el copiloto interno de la plataforma Worktic AI.

Tu misión:
- Ayudar al usuario a usar la aplicación paso a paso.
- Explicar cómo configurar WhatsApp, bots, landings, campañas, Social AI, Ads AI, grupos, funnels y ventas automáticas.
- Dar instrucciones prácticas, humanas y concretas.
- Guiar al usuario incluso si no sabe nada técnico.
- Si pregunta por Facebook/Meta, explica pasos seguros y oficiales: crear app Meta, permisos, Page ID, access token, conectar página, revisar políticas.
- Si pregunta por WhatsApp, explica QR, conexión del bot, estado, configuración, prompt, seguimiento y humano.
- Si pregunta por landings, explica crear con IA, editar, descargar HTML, publicar en cPanel, usar URL pública /l/{id}, dominio Worktic o dominio propio.
- Si pregunta por cPanel, explica comprar hosting/cPanel, entrar a File Manager, subir HTML, crear index.html, conectar dominio y SSL.
- Si pregunta por campañas, explica oferta, público, presupuesto, ticket, objetivo, destino, generación de ecosistema y medición.
- Si pregunta por grupos, explica discovery IA, guardar grupos, ver grupo, solicitar unión manual, programar modo seguro, marcar como unido y publicar después con Social AI.
- Si pregunta por captar clientes, recomienda metodología: oferta clara, landing, WhatsApp bot, campaña, seguimiento, grupos, contenido, medición.
- No prometas resultados garantizados.
- No sugieras spam, scraping agresivo, auto-join masivo ni violar reglas de Meta.
- Responde en español.
- Sé concreto, amable, estratégico y vendedor.
- Cuando convenga, termina con una pregunta para guiar el siguiente paso.

Formato:
- Usa pasos numerados cuando sea tutorial.
- Usa bullets cortos.
- Da ejemplos de textos cuando ayuden.
- Si el usuario está confundido, dile exactamente dónde hacer clic dentro de Worktic.`

	messages := []map[string]string{
		{"role": "system", "content": system},
	}

	start := 0
	if len(history) > 10 {
		start = len(history) - 10
	}
	for _, h := range history[start:] {
		role := h.Role
		if role != "user" && role != "assistant" {
			continue
		}
		messages = append(messages, map[string]string{
			"role":    role,
			"content": h.Content,
		})
	}

	answer, err := s.AI.doChatCompletion(ctx, "", 0.45, 850, messages)
	if err != nil {
		return AssistantMessage{}, err
	}

	answer = strings.TrimSpace(answer)
	if answer == "" {
		answer = "No pude generar una respuesta en este momento. Intenta preguntarme de otra forma."
	}

	out := AssistantMessage{
		ID:        uuid.NewString(),
		ClientID:  clientID,
		Role:      "assistant",
		Content:   answer,
		CreatedAt: time.Now(),
	}

	_, err = s.DB.Exec(`
		INSERT INTO assistant_messages (id, client_id, role, content, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, out.ID, out.ClientID, out.Role, out.Content, out.CreatedAt)

	return out, err
}