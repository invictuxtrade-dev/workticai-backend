package services

import (
	"database/sql"
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
	in.Status = "draft"
	in.CreatedAt = now
	in.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO group_bots (
			id, client_id, name, platform, status, system_prompt,
			business_name, business_description, offer, target_audience,
			rules, welcome_message, moderation_enabled, auto_reply_enabled,
			lead_capture_enabled, human_handoff_phone, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		in.ID, in.ClientID, in.Name, in.Platform, in.Status, in.SystemPrompt,
		in.BusinessName, in.BusinessDesc, in.Offer, in.TargetAudience,
		in.Rules, in.WelcomeMessage, groupBoolToInt(in.ModerationEnabled),
		groupBoolToInt(in.AutoReplyEnabled), groupBoolToInt(in.LeadCapture),
		in.HandoffPhone, in.CreatedAt, in.UpdatedAt,
	)

	return in, err
}

func (s *GroupService) ListGroupBots(clientID string) ([]GroupBot, error) {
	rows, err := s.DB.Query(`
		SELECT id, client_id, name, platform, status, system_prompt,
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
			&x.ID, &x.ClientID, &x.Name, &x.Platform, &x.Status, &x.SystemPrompt,
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

	_, err := s.DB.Exec(`
		UPDATE group_bots SET
			name=?,
			status=?,
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
	if strings.TrimSpace(in.URL) == "" {
		return FacebookGroupTarget{}, fmt.Errorf("url required")
	}

	now := time.Now()
	in.ID = uuid.NewString()
	if in.Status == "" {
		in.Status = "discovered"
	}
	if in.JoinStatus == "" {
		in.JoinStatus = "pending_manual_join"
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

func groupBoolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}