package services

import (
	"database/sql"
	"strings"
	"time"

	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type MemoryService struct { DB *sql.DB }
func NewMemoryService(db *sql.DB) *MemoryService { return &MemoryService{DB: db} }

func (m *MemoryService) UpsertLead(botID, chatJID, displayName, phone, incoming string) (models.Lead, error) {
	now := time.Now(); lead, err := m.GetLead(botID, chatJID)
	if err == sql.ErrNoRows {
		stage, intent, summary, tags := classifyLead("new", incoming)
		next := now.Add(60 * time.Minute)
		_, err = m.DB.Exec(`INSERT INTO leads (bot_id, chat_jid, display_name, phone, stage, last_intent, summary, tags, last_inbound_text, created_at, updated_at, last_message_at, followup_count, next_followup_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`, botID, chatJID, displayName, phone, stage, intent, summary, tags, incoming, now, now, now, next)
		if err != nil { return models.Lead{}, err }
		return m.GetLead(botID, chatJID)
	}
	if err != nil { return models.Lead{}, err }
	stage, intent, summary, tags := classifyLead(lead.Stage, incoming)
	combinedSummary := strings.TrimSpace(strings.Join(filterEmpty([]string{lead.Summary, summary}), " | "))
	if tags == "" { tags = lead.Tags }
	followAt := now.Add(60 * time.Minute)
	_, err = m.DB.Exec(`UPDATE leads SET display_name=?, phone=?, stage=?, last_intent=?, summary=?, tags=?, last_inbound_text=?, updated_at=?, last_message_at=?, next_followup_at=? WHERE bot_id=? AND chat_jid=?`, displayName, phone, stage, intent, combinedSummary, tags, incoming, now, now, followAt, botID, chatJID)
	if err != nil { return models.Lead{}, err }
	return m.GetLead(botID, chatJID)
}

func (m *MemoryService) SaveReply(botID, chatJID, reply string) error { _, err := m.DB.Exec(`UPDATE leads SET last_reply_text=?, updated_at=? WHERE bot_id=? AND chat_jid=?`, reply, time.Now(), botID, chatJID); return err }
func (m *MemoryService) SaveMessage(botID, chatJID, direction, content string) error { _, err := m.DB.Exec(`INSERT INTO messages (bot_id, chat_jid, direction, content, created_at) VALUES (?, ?, ?, ?, ?)`, botID, chatJID, direction, content, time.Now()); return err }

func (m *MemoryService) GetLead(botID, chatJID string) (models.Lead, error) {
	var lead models.Lead; var last, next sql.NullTime
	err := m.DB.QueryRow(`SELECT id, bot_id, chat_jid, COALESCE(display_name,''), COALESCE(phone,''), stage, last_intent, COALESCE(summary,''), COALESCE(tags,''), COALESCE(last_inbound_text,''), COALESCE(last_reply_text,''), followup_count, next_followup_at, created_at, updated_at, last_message_at FROM leads WHERE bot_id=? AND chat_jid=?`, botID, chatJID).Scan(&lead.ID,&lead.BotID,&lead.ChatJID,&lead.DisplayName,&lead.Phone,&lead.Stage,&lead.LastIntent,&lead.Summary,&lead.Tags,&lead.LastInboundText,&lead.LastReplyText,&lead.FollowupCount,&next,&lead.CreatedAt,&lead.UpdatedAt,&last)
	if err != nil { return models.Lead{}, err }
	if last.Valid { lead.LastMessageAt = &last.Time }
	if next.Valid { lead.NextFollowupAt = &next.Time }
	return lead,nil
}

func (m *MemoryService) ListLeads(botID string) ([]models.Lead, error) {
	rows, err := m.DB.Query(`SELECT id, bot_id, chat_jid, COALESCE(display_name,''), COALESCE(phone,''), stage, last_intent, COALESCE(summary,''), COALESCE(tags,''), COALESCE(last_inbound_text,''), COALESCE(last_reply_text,''), followup_count, next_followup_at, created_at, updated_at, last_message_at FROM leads WHERE bot_id=? ORDER BY updated_at DESC`, botID)
	if err != nil { return nil, err }
	defer rows.Close(); out := []models.Lead{}
	for rows.Next() { var lead models.Lead; var last, next sql.NullTime; if err := rows.Scan(&lead.ID,&lead.BotID,&lead.ChatJID,&lead.DisplayName,&lead.Phone,&lead.Stage,&lead.LastIntent,&lead.Summary,&lead.Tags,&lead.LastInboundText,&lead.LastReplyText,&lead.FollowupCount,&next,&lead.CreatedAt,&lead.UpdatedAt,&last); err != nil { return nil, err }; if last.Valid { lead.LastMessageAt = &last.Time }; if next.Valid { lead.NextFollowupAt = &next.Time }; out = append(out,lead) }
	return out,nil
}

func (m *MemoryService) UpdateLeadStage(botID string, leadID int64, stage string) error { _, err := m.DB.Exec(`UPDATE leads SET stage=?, updated_at=? WHERE bot_id=? AND id=?`, stage, time.Now(), botID, leadID); return err }

func (m *MemoryService) ListMessages(botID, chatJID string) ([]models.Message, error) {
	rows, err := m.DB.Query(`SELECT id, bot_id, chat_jid, direction, content, created_at FROM messages WHERE bot_id=? AND chat_jid=? ORDER BY created_at ASC`, botID, chatJID)
	if err != nil { return nil, err }
	defer rows.Close(); out := []models.Message{}
	for rows.Next() { var msg models.Message; if err := rows.Scan(&msg.ID,&msg.BotID,&msg.ChatJID,&msg.Direction,&msg.Content,&msg.CreatedAt); err != nil { return nil, err }; out = append(out,msg) }
	return out,nil
}

func (m *MemoryService) DueFollowups(now time.Time) ([]models.Lead, error) {
	rows, err := m.DB.Query(`SELECT id, bot_id, chat_jid, COALESCE(display_name,''), COALESCE(phone,''), stage, last_intent, COALESCE(summary,''), COALESCE(tags,''), COALESCE(last_inbound_text,''), COALESCE(last_reply_text,''), followup_count, next_followup_at, created_at, updated_at, last_message_at FROM leads WHERE next_followup_at IS NOT NULL AND next_followup_at <= ? AND stage NOT IN ('closed','lost')`, now)
	if err != nil { return nil, err }
	defer rows.Close(); out := []models.Lead{}
	for rows.Next() { var lead models.Lead; var last, next sql.NullTime; if err := rows.Scan(&lead.ID,&lead.BotID,&lead.ChatJID,&lead.DisplayName,&lead.Phone,&lead.Stage,&lead.LastIntent,&lead.Summary,&lead.Tags,&lead.LastInboundText,&lead.LastReplyText,&lead.FollowupCount,&next,&lead.CreatedAt,&lead.UpdatedAt,&last); err != nil { return nil, err }; if last.Valid { lead.LastMessageAt = &last.Time }; if next.Valid { lead.NextFollowupAt = &next.Time }; out = append(out,lead) }
	return out,nil
}

func (m *MemoryService) MarkFollowupSent(botID string, leadID int64, delayMins int) error {
	next := time.Now().Add(time.Duration(delayMins) * time.Minute)
	_, err := m.DB.Exec(`UPDATE leads SET followup_count = followup_count + 1, next_followup_at=?, updated_at=? WHERE bot_id=? AND id=?`, next, time.Now(), botID, leadID)
	return err
}

func classifyLead(currentStage, incoming string) (stage, intent, summary, tags string) {
	text := strings.ToLower(strings.TrimSpace(incoming)); stage=currentStage; intent="unknown"; if stage=="" { stage="new" }
	switch {
	case containsAny(text, "precio", "cuánto", "cuanto", "valor", "coste"):
		intent="pricing"; stage=maxStage(stage,"interested"); summary="Preguntó por precio"; tags="pricing"
	case containsAny(text, "resultado", "rentabilidad", "ganancia"):
		intent="results"; stage=maxStage(stage,"interested"); summary="Preguntó por resultados"; tags="results"
	case containsAny(text, "copy", "copy trading", "señales", "senales"):
		intent="copy_trading"; stage=maxStage(stage,"qualified"); summary="Interés en copy trading"; tags="copy_trading"
	case containsAny(text, "bot", "robot", "automatizado", "ea"):
		intent="bots"; stage=maxStage(stage,"qualified"); summary="Interés en bots"; tags="bots"
	case containsAny(text, "fondeo", "funded", "challenge", "cuenta"):
		intent="funding"; stage=maxStage(stage,"qualified"); summary="Interés en fondeo"; tags="funding"
	case containsAny(text, "sí", "si", "quiero", "me interesa", "ok", "dale"):
		intent="positive"; stage=maxStage(stage,"hot"); summary="Lead mostró interés directo"; tags="positive"
	default:
		if stage=="new" { stage="qualified" }; summary="Conversación general"
	}
	return
}
func maxStage(current,candidate string) string { order:=map[string]int{"new":0,"qualified":1,"interested":2,"hot":3,"closed":4,"lost":5}; if order[candidate]>order[current] { return candidate }; return current }
func containsAny(text string, words ...string) bool { for _,w := range words { if strings.Contains(text,w) { return true } }; return false }
func filterEmpty(items []string) []string { out:=make([]string,0,len(items)); for _,item:= range items { item=strings.TrimSpace(item); if item!="" { out=append(out,item) } }; return out }
