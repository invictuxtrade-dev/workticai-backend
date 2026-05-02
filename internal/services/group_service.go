package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type GroupService struct {
	DB *sql.DB
	AI *AIService
}

func NewGroupService(db *sql.DB, ai *AIService) *GroupService {
	return &GroupService{DB: db, AI: ai}
}

type GroupBot struct {
	ID                string    `json:"id"`
	ClientID          string    `json:"client_id"`
	Name              string    `json:"name"`
	Platform          string    `json:"platform"`
	Status            string    `json:"status"`
	GroupJID          string    `json:"group_jid"`
	SystemPrompt      string    `json:"system_prompt"`
	BusinessName      string    `json:"business_name"`
	BusinessDesc      string    `json:"business_description"`
	Offer             string    `json:"offer"`
	TargetAudience    string    `json:"target_audience"`
	Rules             string    `json:"rules"`
	WelcomeMessage    string    `json:"welcome_message"`
	ModerationEnabled bool      `json:"moderation_enabled"`
	AutoReplyEnabled  bool      `json:"auto_reply_enabled"`
	LeadCapture       bool      `json:"lead_capture_enabled"`
	HandoffPhone       string    `json:"human_handoff_phone"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type FacebookGroupTarget struct {
	ID             string    `json:"id"`
	ClientID       string    `json:"client_id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	Category       string    `json:"category"`
	Niche          string    `json:"niche"`
	MembersCount   int       `json:"members_count"`
	RelevanceScore int       `json:"relevance_score"`
	Status         string    `json:"status"`
	JoinStatus     string    `json:"join_status"`
	RulesSummary   string    `json:"rules_summary"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type FacebookGroupDiscoveryRequest struct {
	Product        string `json:"product"`
	BusinessName   string `json:"business_name"`
	Offer          string `json:"offer"`
	TargetAudience string `json:"target_audience"`
	Country         string `json:"country"`
	Niche           string `json:"niche"`
}

type FacebookGroupDiscoveryResult struct {
	SearchKeywords []string              `json:"search_keywords"`
	Strategy       string                `json:"strategy"`
	Warnings       []string              `json:"warnings"`
	Groups         []FacebookGroupTarget `json:"groups"`
}

type GroupGrowthSettings struct {
	ID               string    `json:"id"`
	ClientID         string    `json:"client_id"`
	AutoJoinEnabled bool      `json:"auto_join_enabled"`
	SafeMode         bool      `json:"safe_mode"`
	MaxJoinsPerDay   int       `json:"max_joins_per_day"`
	MaxTotalGroups   int       `json:"max_total_groups"`
	MinDelayMinutes  int       `json:"min_delay_minutes"`
	MaxDelayMinutes  int       `json:"max_delay_minutes"`
	AllowedHours     string    `json:"allowed_hours"`
	Timezone         string    `json:"timezone"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type FacebookGroupJoinQueueItem struct {
	ID            string     `json:"id"`
	ClientID      string     `json:"client_id"`
	GroupTargetID string     `json:"group_target_id"`
	GroupName     string     `json:"group_name,omitempty"`
	GroupURL      string     `json:"group_url,omitempty"`
	Status        string     `json:"status"`
	ScheduledFor  *time.Time `json:"scheduled_for,omitempty"`
	ExecutedAt    *time.Time `json:"executed_at,omitempty"`
	Attempts      int        `json:"attempts"`
	LastError     string     `json:"last_error"`
	Notes         string     `json:"notes"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type FacebookGroupActivityLog struct {
	ID            string    `json:"id"`
	ClientID      string    `json:"client_id"`
	GroupTargetID string    `json:"group_target_id"`
	ActionType    string    `json:"action_type"`
	Status        string    `json:"status"`
	Message       string    `json:"message"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *GroupService) CreateGroupBot(in GroupBot) (GroupBot, error) {
	if strings.TrimSpace(in.ClientID) == "" {
		return GroupBot{}, fmt.Errorf("client_id required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return GroupBot{}, fmt.Errorf("name required")
	}

	now := time.Now()
	in.ID = uuid.NewString()
	in.Platform = "whatsapp"
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "draft"
	}
	in.CreatedAt = now
	in.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO group_bots (
			id, client_id, name, platform, status, group_jid, system_prompt,
			business_name, business_description, offer, target_audience,
			rules, welcome_message, moderation_enabled, auto_reply_enabled,
			lead_capture_enabled, human_handoff_phone, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		in.ID, in.ClientID, in.Name, in.Platform, in.Status, in.GroupJID, in.SystemPrompt,
		in.BusinessName, in.BusinessDesc, in.Offer, in.TargetAudience,
		in.Rules, in.WelcomeMessage, groupBoolToInt(in.ModerationEnabled),
		groupBoolToInt(in.AutoReplyEnabled), groupBoolToInt(in.LeadCapture),
		in.HandoffPhone, in.CreatedAt, in.UpdatedAt,
	)

	return in, err
}

func (s *GroupService) ListGroupBots(clientID string) ([]GroupBot, error) {
	rows, err := s.DB.Query(`
		SELECT id, client_id, name, platform, status, group_jid, system_prompt,
		       business_name, business_description, offer, target_audience,
		       rules, welcome_message, moderation_enabled, auto_reply_enabled,
		       lead_capture_enabled, human_handoff_phone, created_at, updated_at
		FROM group_bots
		WHERE (?='' OR client_id=?)
		ORDER BY created_at DESC
	`, clientID, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []GroupBot{}
	for rows.Next() {
		var x GroupBot
		var moderation, autoReply, leadCapture int
		if err := rows.Scan(
			&x.ID, &x.ClientID, &x.Name, &x.Platform, &x.Status, &x.GroupJID, &x.SystemPrompt,
			&x.BusinessName, &x.BusinessDesc, &x.Offer, &x.TargetAudience,
			&x.Rules, &x.WelcomeMessage, &moderation, &autoReply, &leadCapture,
			&x.HandoffPhone, &x.CreatedAt, &x.UpdatedAt,
		); err != nil {
			return nil, err
		}
		x.ModerationEnabled = moderation == 1
		x.AutoReplyEnabled = autoReply == 1
		x.LeadCapture = leadCapture == 1
		out = append(out, x)
	}

	return out, nil
}

func (s *GroupService) UpdateGroupBot(id string, in GroupBot) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("id required")
	}
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "draft"
	}

	_, err := s.DB.Exec(`
		UPDATE group_bots SET
			name=?,
			status=?,
			group_jid=?,
			system_prompt=?,
			business_name=?,
			business_description=?,
			offer=?,
			target_audience=?,
			rules=?,
			welcome_message=?,
			moderation_enabled=?,
			auto_reply_enabled=?,
			lead_capture_enabled=?,
			human_handoff_phone=?,
			updated_at=?
		WHERE id=?
	`,
		in.Name,
		in.Status,
		in.GroupJID,
		in.SystemPrompt,
		in.BusinessName,
		in.BusinessDesc,
		in.Offer,
		in.TargetAudience,
		in.Rules,
		in.WelcomeMessage,
		groupBoolToInt(in.ModerationEnabled),
		groupBoolToInt(in.AutoReplyEnabled),
		groupBoolToInt(in.LeadCapture),
		in.HandoffPhone,
		time.Now(),
		id,
	)
	return err
}

func (s *GroupService) DeleteGroupBot(id string) error {
	_, err := s.DB.Exec(`DELETE FROM group_bots WHERE id=?`, id)
	return err
}

func (s *GroupService) SaveFacebookGroupTarget(in FacebookGroupTarget) (FacebookGroupTarget, error) {
	if strings.TrimSpace(in.ClientID) == "" {
		return FacebookGroupTarget{}, fmt.Errorf("client_id required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return FacebookGroupTarget{}, fmt.Errorf("name required")
	}

	now := time.Now()
	in.ID = uuid.NewString()
	if in.Status == "" {
		in.Status = "discovered"
	}
	if in.JoinStatus == "" {
		in.JoinStatus = "pending_manual_join"
	}
	if in.RelevanceScore <= 0 {
		in.RelevanceScore = 70
	}
	in.CreatedAt = now
	in.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO facebook_group_targets (
			id, client_id, name, url, category, niche, members_count,
			relevance_score, status, join_status, rules_summary, notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		in.ID, in.ClientID, in.Name, in.URL, in.Category, in.Niche,
		in.MembersCount, in.RelevanceScore, in.Status, in.JoinStatus,
		in.RulesSummary, in.Notes, in.CreatedAt, in.UpdatedAt,
	)

	return in, err
}

func (s *GroupService) ListFacebookGroupTargets(clientID string) ([]FacebookGroupTarget, error) {
	rows, err := s.DB.Query(`
		SELECT id, client_id, name, url, category, niche, members_count,
		       relevance_score, status, join_status, rules_summary, notes,
		       created_at, updated_at
		FROM facebook_group_targets
		WHERE (?='' OR client_id=?)
		ORDER BY relevance_score DESC, created_at DESC
	`, clientID, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []FacebookGroupTarget{}
	for rows.Next() {
		var x FacebookGroupTarget
		if err := rows.Scan(
			&x.ID, &x.ClientID, &x.Name, &x.URL, &x.Category, &x.Niche,
			&x.MembersCount, &x.RelevanceScore, &x.Status, &x.JoinStatus,
			&x.RulesSummary, &x.Notes, &x.CreatedAt, &x.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, x)
	}

	return out, nil
}

func (s *GroupService) UpdateFacebookGroupTarget(id string, in FacebookGroupTarget) error {
	_, err := s.DB.Exec(`
		UPDATE facebook_group_targets SET
			name=?,
			url=?,
			category=?,
			niche=?,
			members_count=?,
			relevance_score=?,
			status=?,
			join_status=?,
			rules_summary=?,
			notes=?,
			updated_at=?
		WHERE id=?
	`,
		in.Name,
		in.URL,
		in.Category,
		in.Niche,
		in.MembersCount,
		in.RelevanceScore,
		in.Status,
		in.JoinStatus,
		in.RulesSummary,
		in.Notes,
		time.Now(),
		id,
	)
	return err
}

func (s *GroupService) DeleteFacebookGroupTarget(id string) error {
	_, err := s.DB.Exec(`DELETE FROM facebook_group_targets WHERE id=?`, id)
	return err
}

func (s *GroupService) DiscoverFacebookGroups(ctx context.Context, req FacebookGroupDiscoveryRequest, clientID string) (FacebookGroupDiscoveryResult, error) {
	if s.AI == nil {
		return FacebookGroupDiscoveryResult{}, fmt.Errorf("ai service not configured")
	}

	product := strings.TrimSpace(req.Product)
	if product == "" {
		return FacebookGroupDiscoveryResult{}, fmt.Errorf("product required")
	}

	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = "Latinoamérica"
	}

	system := `Eres un estratega experto en crecimiento orgánico, Facebook Groups y captación ética de clientes.
Debes recomendar grupos objetivo para una campaña, SIN sugerir spam, bots de auto-join ni violar reglas de Meta.
Devuelve SOLO JSON válido.`

	user := fmt.Sprintf(`
Producto/servicio: %s
Negocio: %s
Oferta: %s
Público objetivo: %s
País/mercado: %s
Nicho: %s

Devuelve JSON con esta estructura exacta:
{
  "search_keywords": ["keyword 1", "keyword 2"],
  "strategy": "estrategia segura para entrar y aportar valor",
  "warnings": ["advertencia 1"],
  "groups": [
    {
      "name": "Nombre sugerido o tipo de grupo",
      "url": "",
      "category": "Categoría",
      "niche": "Nicho",
      "members_count": 0,
      "relevance_score": 85,
      "status": "discovered",
      "join_status": "pending_manual_join",
      "rules_summary": "Qué revisar antes de publicar",
      "notes": "Por qué este grupo es relevante"
    }
  ]
}

IMPORTANTE:
- No inventes URLs reales.
- Recomienda tipos de grupos y búsquedas.
- El usuario debe unirse manualmente.
- Nada de spam ni automatización agresiva.
`, product, req.BusinessName, req.Offer, req.TargetAudience, country, req.Niche)

	answer, err := s.AI.doHeavyCompletion(
		ctx,
		"",
		0.5,
		2200,
		[]map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	)
	if err != nil {
		return FacebookGroupDiscoveryResult{}, err
	}

	answer = cleanCodeBlock(answer)

	var out FacebookGroupDiscoveryResult
	if err := json.Unmarshal([]byte(answer), &out); err != nil {
		return FacebookGroupDiscoveryResult{}, fmt.Errorf("facebook groups ai parse: %w", err)
	}

	for i := range out.Groups {
		out.Groups[i].ClientID = clientID
		if strings.TrimSpace(out.Groups[i].Status) == "" {
			out.Groups[i].Status = "discovered"
		}
		if strings.TrimSpace(out.Groups[i].JoinStatus) == "" {
			out.Groups[i].JoinStatus = "pending_manual_join"
		}
		if out.Groups[i].RelevanceScore <= 0 {
			out.Groups[i].RelevanceScore = 70
		}
	}

	if len(out.Warnings) == 0 {
		out.Warnings = []string{
			"No se recomienda auto-unirse ni publicar automáticamente sin aprobación del usuario.",
			"Revisar reglas de cada grupo antes de publicar.",
		}
	}

	return out, nil
}

func (s *GroupService) GetGrowthSettings(clientID string) (GroupGrowthSettings, error) {
	if strings.TrimSpace(clientID) == "" {
		return GroupGrowthSettings{}, fmt.Errorf("client_id required")
	}

	var out GroupGrowthSettings
	var autoJoin, safeMode int

	err := s.DB.QueryRow(`
		SELECT id, client_id, auto_join_enabled, safe_mode, max_joins_per_day,
		       max_total_groups, min_delay_minutes, max_delay_minutes,
		       allowed_hours, timezone, created_at, updated_at
		FROM group_growth_settings
		WHERE client_id=?
	`, clientID).Scan(
		&out.ID,
		&out.ClientID,
		&autoJoin,
		&safeMode,
		&out.MaxJoinsPerDay,
		&out.MaxTotalGroups,
		&out.MinDelayMinutes,
		&out.MaxDelayMinutes,
		&out.AllowedHours,
		&out.Timezone,
		&out.CreatedAt,
		&out.UpdatedAt,
	)

	if err == nil {
		out.AutoJoinEnabled = autoJoin == 1
		out.SafeMode = safeMode == 1
		return out, nil
	}

	if err != sql.ErrNoRows {
		return GroupGrowthSettings{}, err
	}

	now := time.Now()
	out = GroupGrowthSettings{
		ID:               uuid.NewString(),
		ClientID:         clientID,
		AutoJoinEnabled:  false,
		SafeMode:         true,
		MaxJoinsPerDay:   2,
		MaxTotalGroups:   50,
		MinDelayMinutes:  120,
		MaxDelayMinutes:  360,
		AllowedHours:     "08:00-20:00",
		Timezone:         "America/Bogota",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	_, err = s.DB.Exec(`
		INSERT INTO group_growth_settings (
			id, client_id, auto_join_enabled, safe_mode,
			max_joins_per_day, max_total_groups,
			min_delay_minutes, max_delay_minutes,
			allowed_hours, timezone, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		out.ID,
		out.ClientID,
		groupBoolToInt(out.AutoJoinEnabled),
		groupBoolToInt(out.SafeMode),
		out.MaxJoinsPerDay,
		out.MaxTotalGroups,
		out.MinDelayMinutes,
		out.MaxDelayMinutes,
		out.AllowedHours,
		out.Timezone,
		out.CreatedAt,
		out.UpdatedAt,
	)

	return out, err
}

func (s *GroupService) SaveGrowthSettings(in GroupGrowthSettings) (GroupGrowthSettings, error) {
	if strings.TrimSpace(in.ClientID) == "" {
		return GroupGrowthSettings{}, fmt.Errorf("client_id required")
	}

	if in.MaxJoinsPerDay < 1 {
		in.MaxJoinsPerDay = 1
	}
	if in.SafeMode && in.MaxJoinsPerDay > 2 {
		in.MaxJoinsPerDay = 2
	}
	if in.MaxTotalGroups < 1 {
		in.MaxTotalGroups = 50
	}
	if in.MinDelayMinutes < 60 {
		in.MinDelayMinutes = 120
	}
	if in.MaxDelayMinutes < in.MinDelayMinutes {
		in.MaxDelayMinutes = in.MinDelayMinutes + 120
	}
	if strings.TrimSpace(in.AllowedHours) == "" {
		in.AllowedHours = "08:00-20:00"
	}
	if strings.TrimSpace(in.Timezone) == "" {
		in.Timezone = "America/Bogota"
	}

	existing, _ := s.GetGrowthSettings(in.ClientID)
	now := time.Now()

	if existing.ID == "" {
		in.ID = uuid.NewString()
		in.CreatedAt = now
	} else {
		in.ID = existing.ID
		in.CreatedAt = existing.CreatedAt
	}
	in.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO group_growth_settings (
			id, client_id, auto_join_enabled, safe_mode,
			max_joins_per_day, max_total_groups,
			min_delay_minutes, max_delay_minutes,
			allowed_hours, timezone, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_id) DO UPDATE SET
			auto_join_enabled=excluded.auto_join_enabled,
			safe_mode=excluded.safe_mode,
			max_joins_per_day=excluded.max_joins_per_day,
			max_total_groups=excluded.max_total_groups,
			min_delay_minutes=excluded.min_delay_minutes,
			max_delay_minutes=excluded.max_delay_minutes,
			allowed_hours=excluded.allowed_hours,
			timezone=excluded.timezone,
			updated_at=excluded.updated_at
	`,
		in.ID,
		in.ClientID,
		groupBoolToInt(in.AutoJoinEnabled),
		groupBoolToInt(in.SafeMode),
		in.MaxJoinsPerDay,
		in.MaxTotalGroups,
		in.MinDelayMinutes,
		in.MaxDelayMinutes,
		in.AllowedHours,
		in.Timezone,
		in.CreatedAt,
		in.UpdatedAt,
	)

	return in, err
}

func (s *GroupService) RequestFacebookGroupJoin(clientID, groupTargetID, mode string) (FacebookGroupJoinQueueItem, error) {
	if strings.TrimSpace(clientID) == "" {
		return FacebookGroupJoinQueueItem{}, fmt.Errorf("client_id required")
	}
	if strings.TrimSpace(groupTargetID) == "" {
		return FacebookGroupJoinQueueItem{}, fmt.Errorf("group_target_id required")
	}

	var exists int
	if err := s.DB.QueryRow(`
		SELECT COUNT(*) FROM facebook_group_targets
		WHERE id=? AND client_id=?
	`, groupTargetID, clientID).Scan(&exists); err != nil {
		return FacebookGroupJoinQueueItem{}, err
	}
	if exists == 0 {
		return FacebookGroupJoinQueueItem{}, fmt.Errorf("group target not found")
	}

	settings, err := s.GetGrowthSettings(clientID)
	if err != nil {
		return FacebookGroupJoinQueueItem{}, err
	}

	var joinedCount int
	_ = s.DB.QueryRow(`
		SELECT COUNT(*) FROM facebook_group_targets
		WHERE client_id=? AND join_status='joined'
	`, clientID).Scan(&joinedCount)

	if joinedCount >= settings.MaxTotalGroups {
		return FacebookGroupJoinQueueItem{}, fmt.Errorf("max total groups reached")
	}

	var todayCount int
	_ = s.DB.QueryRow(`
		SELECT COUNT(*) FROM facebook_group_join_queue
		WHERE client_id=?
		  AND date(created_at)=date('now')
		  AND status IN ('scheduled','manual_required','joined','processing')
	`, clientID).Scan(&todayCount)

	if todayCount >= settings.MaxJoinsPerDay {
		return FacebookGroupJoinQueueItem{}, fmt.Errorf("daily join limit reached")
	}

	now := time.Now()
	item := FacebookGroupJoinQueueItem{
		ID:            uuid.NewString(),
		ClientID:      clientID,
		GroupTargetID: groupTargetID,
		Status:        "manual_required",
		Attempts:      0,
		Notes:         "Abre el grupo, solicita unirte manualmente y luego marca como unido.",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if strings.ToLower(strings.TrimSpace(mode)) == "auto" && settings.AutoJoinEnabled {
		scheduled := now.Add(time.Duration(settings.MinDelayMinutes) * time.Minute)
		item.Status = "scheduled"
		item.ScheduledFor = &scheduled
		item.Notes = "Programado en modo seguro. Requiere ejecución supervisada/manual."
	}

	_, err = s.DB.Exec(`
		INSERT INTO facebook_group_join_queue (
			id, client_id, group_target_id, status, scheduled_for,
			attempts, last_error, notes, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.ClientID,
		item.GroupTargetID,
		item.Status,
		item.ScheduledFor,
		item.Attempts,
		item.LastError,
		item.Notes,
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return FacebookGroupJoinQueueItem{}, err
	}

	_, _ = s.DB.Exec(`
		UPDATE facebook_group_targets
		SET join_status=?, status='queued', last_join_attempt=?, updated_at=?
		WHERE id=? AND client_id=?
	`,
		item.Status,
		now,
		now,
		groupTargetID,
		clientID,
	)

	_ = s.CreateFacebookGroupLog(clientID, groupTargetID, "join_requested", item.Status, item.Notes)

	return item, nil
}

func (s *GroupService) ListJoinQueue(clientID string) ([]FacebookGroupJoinQueueItem, error) {
	rows, err := s.DB.Query(`
		SELECT q.id, q.client_id, q.group_target_id,
		       COALESCE(t.name, ''), COALESCE(t.url, ''),
		       q.status, q.scheduled_for, q.executed_at,
		       q.attempts, q.last_error, q.notes,
		       q.created_at, q.updated_at
		FROM facebook_group_join_queue q
		LEFT JOIN facebook_group_targets t ON t.id=q.group_target_id
		WHERE (?='' OR q.client_id=?)
		ORDER BY q.created_at DESC
	`, clientID, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []FacebookGroupJoinQueueItem{}
	for rows.Next() {
		var x FacebookGroupJoinQueueItem
		if err := rows.Scan(
			&x.ID,
			&x.ClientID,
			&x.GroupTargetID,
			&x.GroupName,
			&x.GroupURL,
			&x.Status,
			&x.ScheduledFor,
			&x.ExecutedAt,
			&x.Attempts,
			&x.LastError,
			&x.Notes,
			&x.CreatedAt,
			&x.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, x)
	}

	return out, nil
}

func (s *GroupService) MarkFacebookGroupJoined(clientID, groupTargetID string) error {
	now := time.Now()

	_, err := s.DB.Exec(`
		UPDATE facebook_group_targets
		SET join_status='joined', status='joined', joined_at=?, updated_at=?
		WHERE id=? AND client_id=?
	`, now, now, groupTargetID, clientID)
	if err != nil {
		return err
	}

	_, _ = s.DB.Exec(`
		UPDATE facebook_group_join_queue
		SET status='joined', executed_at=?, updated_at=?
		WHERE group_target_id=? AND client_id=?
	`, now, now, groupTargetID, clientID)

	_ = s.CreateFacebookGroupLog(clientID, groupTargetID, "joined_confirmed", "joined", "Usuario confirmó que ya está unido al grupo.")

	return nil
}

func (s *GroupService) UpdateJoinQueueStatus(clientID, queueID, status, message string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("status required")
	}

	now := time.Now()

	_, err := s.DB.Exec(`
		UPDATE facebook_group_join_queue
		SET status=?, last_error=?, updated_at=?
		WHERE id=? AND client_id=?
	`, status, message, now, queueID, clientID)
	if err != nil {
		return err
	}

	var groupTargetID string
	_ = s.DB.QueryRow(`
		SELECT group_target_id FROM facebook_group_join_queue
		WHERE id=? AND client_id=?
	`, queueID, clientID).Scan(&groupTargetID)

	if groupTargetID != "" {
		_, _ = s.DB.Exec(`
			UPDATE facebook_group_targets
			SET join_status=?, updated_at=?
			WHERE id=? AND client_id=?
		`, status, now, groupTargetID, clientID)

		_ = s.CreateFacebookGroupLog(clientID, groupTargetID, "queue_status_updated", status, message)
	}

	return nil
}

func (s *GroupService) CreateFacebookGroupLog(clientID, groupTargetID, actionType, status, message string) error {
	_, err := s.DB.Exec(`
		INSERT INTO facebook_group_activity_logs (
			id, client_id, group_target_id, action_type, status, message, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		uuid.NewString(),
		clientID,
		groupTargetID,
		actionType,
		status,
		message,
		time.Now(),
	)
	return err
}

func (s *GroupService) ListFacebookGroupLogs(clientID, groupTargetID string) ([]FacebookGroupActivityLog, error) {
	query := `
		SELECT id, client_id, group_target_id, action_type, status, message, created_at
		FROM facebook_group_activity_logs
		WHERE (?='' OR client_id=?)
	`
	args := []any{clientID, clientID}

	if strings.TrimSpace(groupTargetID) != "" {
		query += ` AND group_target_id=?`
		args = append(args, groupTargetID)
	}

	query += ` ORDER BY created_at DESC LIMIT 100`

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []FacebookGroupActivityLog{}
	for rows.Next() {
		var x FacebookGroupActivityLog
		if err := rows.Scan(
			&x.ID,
			&x.ClientID,
			&x.GroupTargetID,
			&x.ActionType,
			&x.Status,
			&x.Message,
			&x.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, x)
	}

	return out, nil
}

func groupBoolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}