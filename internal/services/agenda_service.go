package services

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AgendaService struct {
	DB *sql.DB
}

func NewAgendaService(db *sql.DB) *AgendaService {
	return &AgendaService{DB: db}
}

type Appointment struct {
	ID             string     `json:"id"`
	ClientID       string     `json:"client_id"`
	BotID          string     `json:"bot_id"`
	LeadID         int64      `json:"lead_id"`
	AgentID        string     `json:"agent_id"`
	Title          string     `json:"title"`
	ContactName    string     `json:"contact_name"`
	ContactPhone   string     `json:"contact_phone"`
	ContactEmail   string     `json:"contact_email"`
	Status         string     `json:"status"`
	Source         string     `json:"source"`
	MeetingType    string     `json:"meeting_type"`
	MeetingLink    string     `json:"meeting_link"`
	Location       string     `json:"location"`
	Notes          string     `json:"notes"`
	AISummary      string     `json:"ai_summary"`
	LeadScore      int        `json:"lead_score"`
	StartAt        time.Time  `json:"start_at"`
	EndAt          time.Time  `json:"end_at"`
	Timezone       string     `json:"timezone"`
	ReminderSentAt *time.Time `json:"reminder_sent_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type AppointmentSettings struct {
	ID                    string    `json:"id"`
	ClientID              string    `json:"client_id"`
	BotID                 string    `json:"bot_id"`
	Enabled               bool      `json:"enabled"`
	Goal                  string    `json:"goal"`
	Timezone              string    `json:"timezone"`
	DurationMins          int       `json:"duration_mins"`
	BufferMins            int       `json:"buffer_mins"`
	AvailableDays         string    `json:"available_days"`
	StartTime              string    `json:"start_time"`
	EndTime                string    `json:"end_time"`
	NotifyEmail           string    `json:"notify_email"`
	NotifyWhatsapp        string    `json:"notify_whatsapp"`
	AutoConfirm           bool      `json:"auto_confirm"`
	ReminderBeforeMins    int       `json:"reminder_before_mins"`
	FollowupNoShowEnabled bool      `json:"followup_no_show_enabled"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type AppointmentAgent struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Whatsapp  string    `json:"whatsapp"`
	Role      string    `json:"role"`
	Color     string    `json:"color"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *AgendaService) ListAppointments(clientID, status string) ([]Appointment, error) {
	query := `
		SELECT id, client_id, bot_id, lead_id, agent_id, title, contact_name, contact_phone, contact_email,
		       status, source, meeting_type, meeting_link, location, notes, ai_summary, lead_score,
		       start_at, end_at, timezone, reminder_sent_at, created_at, updated_at
		FROM appointments
		WHERE (?='' OR client_id=?)
	`
	args := []any{clientID, clientID}

	if strings.TrimSpace(status) != "" {
		query += ` AND status=?`
		args = append(args, status)
	}

	query += ` ORDER BY start_at ASC`

	rows, err := a.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Appointment{}
	for rows.Next() {
		var x Appointment
		if err := rows.Scan(
			&x.ID, &x.ClientID, &x.BotID, &x.LeadID, &x.AgentID, &x.Title, &x.ContactName, &x.ContactPhone, &x.ContactEmail,
			&x.Status, &x.Source, &x.MeetingType, &x.MeetingLink, &x.Location, &x.Notes, &x.AISummary, &x.LeadScore,
			&x.StartAt, &x.EndAt, &x.Timezone, &x.ReminderSentAt, &x.CreatedAt, &x.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, x)
	}

	return out, nil
}

func (a *AgendaService) CreateAppointment(x Appointment) (Appointment, error) {
	if strings.TrimSpace(x.ClientID) == "" {
		return Appointment{}, errors.New("client_id required")
	}

	now := time.Now()

	if x.ID == "" {
		x.ID = uuid.NewString()
	}
	if x.Status == "" {
		x.Status = "scheduled"
	}
	if x.Source == "" {
		x.Source = "manual"
	}
	if x.MeetingType == "" {
		x.MeetingType = "call"
	}
	if x.Timezone == "" {
		x.Timezone = "America/Bogota"
	}
	if x.Title == "" {
		x.Title = "Cita comercial"
	}
	if x.EndAt.IsZero() {
		x.EndAt = x.StartAt.Add(30 * time.Minute)
	}

	x.CreatedAt = now
	x.UpdatedAt = now

	_, err := a.DB.Exec(`
		INSERT INTO appointments (
			id, client_id, bot_id, lead_id, agent_id, title, contact_name, contact_phone, contact_email,
			status, source, meeting_type, meeting_link, location, notes, ai_summary, lead_score,
			start_at, end_at, timezone, reminder_sent_at, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		x.ID, x.ClientID, x.BotID, x.LeadID, x.AgentID, x.Title, x.ContactName, x.ContactPhone, x.ContactEmail,
		x.Status, x.Source, x.MeetingType, x.MeetingLink, x.Location, x.Notes, x.AISummary, x.LeadScore,
		x.StartAt, x.EndAt, x.Timezone, x.ReminderSentAt, x.CreatedAt, x.UpdatedAt,
	)

	return x, err
}

func (a *AgendaService) UpdateAppointment(x Appointment) error {
	if strings.TrimSpace(x.ID) == "" {
		return errors.New("appointment id required")
	}

	x.UpdatedAt = time.Now()

	_, err := a.DB.Exec(`
		UPDATE appointments SET
			agent_id=?,
			title=?,
			contact_name=?,
			contact_phone=?,
			contact_email=?,
			status=?,
			meeting_type=?,
			meeting_link=?,
			location=?,
			notes=?,
			ai_summary=?,
			lead_score=?,
			start_at=?,
			end_at=?,
			timezone=?,
			updated_at=?
		WHERE id=?
	`,
		x.AgentID,
		x.Title,
		x.ContactName,
		x.ContactPhone,
		x.ContactEmail,
		x.Status,
		x.MeetingType,
		x.MeetingLink,
		x.Location,
		x.Notes,
		x.AISummary,
		x.LeadScore,
		x.StartAt,
		x.EndAt,
		x.Timezone,
		x.UpdatedAt,
		x.ID,
	)

	return err
}

func (a *AgendaService) DeleteAppointment(id string) error {
	_, err := a.DB.Exec(`DELETE FROM appointments WHERE id=?`, id)
	return err
}

func (a *AgendaService) GetSettings(clientID, botID string) (AppointmentSettings, error) {
	var x AppointmentSettings
	var enabled, autoConfirm, followupNoShow int

	err := a.DB.QueryRow(`
		SELECT id, client_id, bot_id, enabled, goal, timezone, duration_mins, buffer_mins,
		       available_days, start_time, end_time, notify_email, notify_whatsapp,
		       auto_confirm, reminder_before_mins, followup_no_show_enabled, created_at, updated_at
		FROM appointment_settings
		WHERE client_id=? AND bot_id=?
		LIMIT 1
	`, clientID, botID).Scan(
		&x.ID,
		&x.ClientID,
		&x.BotID,
		&enabled,
		&x.Goal,
		&x.Timezone,
		&x.DurationMins,
		&x.BufferMins,
		&x.AvailableDays,
		&x.StartTime,
		&x.EndTime,
		&x.NotifyEmail,
		&x.NotifyWhatsapp,
		&autoConfirm,
		&x.ReminderBeforeMins,
		&followupNoShow,
		&x.CreatedAt,
		&x.UpdatedAt,
	)

	x.Enabled = enabled == 1
	x.AutoConfirm = autoConfirm == 1
	x.FollowupNoShowEnabled = followupNoShow == 1

	return x, err
}

func (a *AgendaService) SaveSettings(x AppointmentSettings) (AppointmentSettings, error) {
	if strings.TrimSpace(x.ClientID) == "" {
		return AppointmentSettings{}, errors.New("client_id required")
	}
	if strings.TrimSpace(x.BotID) == "" {
		return AppointmentSettings{}, errors.New("bot_id required")
	}

	now := time.Now()

	if x.ID == "" {
		x.ID = uuid.NewString()
	}
	if x.Goal == "" {
		x.Goal = "sales_call"
	}
	if x.Timezone == "" {
		x.Timezone = "America/Bogota"
	}
	if x.DurationMins <= 0 {
		x.DurationMins = 30
	}
	if x.BufferMins < 0 {
		x.BufferMins = 0
	}
	if x.AvailableDays == "" {
		x.AvailableDays = "mon,tue,wed,thu,fri"
	}
	if x.StartTime == "" {
		x.StartTime = "09:00"
	}
	if x.EndTime == "" {
		x.EndTime = "18:00"
	}
	if x.ReminderBeforeMins <= 0 {
		x.ReminderBeforeMins = 60
	}

	x.UpdatedAt = now

	_, err := a.DB.Exec(`
		INSERT INTO appointment_settings (
			id, client_id, bot_id, enabled, goal, timezone, duration_mins, buffer_mins,
			available_days, start_time, end_time, notify_email, notify_whatsapp,
			auto_confirm, reminder_before_mins, followup_no_show_enabled, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_id, bot_id)
		DO UPDATE SET
			enabled=excluded.enabled,
			goal=excluded.goal,
			timezone=excluded.timezone,
			duration_mins=excluded.duration_mins,
			buffer_mins=excluded.buffer_mins,
			available_days=excluded.available_days,
			start_time=excluded.start_time,
			end_time=excluded.end_time,
			notify_email=excluded.notify_email,
			notify_whatsapp=excluded.notify_whatsapp,
			auto_confirm=excluded.auto_confirm,
			reminder_before_mins=excluded.reminder_before_mins,
			followup_no_show_enabled=excluded.followup_no_show_enabled,
			updated_at=excluded.updated_at
	`,
		x.ID,
		x.ClientID,
		x.BotID,
		boolToInt(x.Enabled),
		x.Goal,
		x.Timezone,
		x.DurationMins,
		x.BufferMins,
		x.AvailableDays,
		x.StartTime,
		x.EndTime,
		x.NotifyEmail,
		x.NotifyWhatsapp,
		boolToInt(x.AutoConfirm),
		x.ReminderBeforeMins,
		boolToInt(x.FollowupNoShowEnabled),
		now,
		now,
	)

	return x, err
}

func (a *AgendaService) ListAgents(clientID string) ([]AppointmentAgent, error) {
	rows, err := a.DB.Query(`
		SELECT id, client_id, name, email, whatsapp, role, color, is_active, created_at, updated_at
		FROM appointment_agents
		WHERE (?='' OR client_id=?)
		ORDER BY created_at DESC
	`, clientID, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AppointmentAgent{}
	for rows.Next() {
		var x AppointmentAgent
		var active int

		if err := rows.Scan(
			&x.ID,
			&x.ClientID,
			&x.Name,
			&x.Email,
			&x.Whatsapp,
			&x.Role,
			&x.Color,
			&active,
			&x.CreatedAt,
			&x.UpdatedAt,
		); err != nil {
			return nil, err
		}

		x.IsActive = active == 1
		out = append(out, x)
	}

	return out, nil
}

func (a *AgendaService) SaveAgent(x AppointmentAgent) (AppointmentAgent, error) {
	if strings.TrimSpace(x.ClientID) == "" {
		return AppointmentAgent{}, errors.New("client_id required")
	}
	if strings.TrimSpace(x.Name) == "" {
		return AppointmentAgent{}, errors.New("name required")
	}

	now := time.Now()

	if x.ID == "" {
		x.ID = uuid.NewString()
		x.CreatedAt = now
	}
	if x.Color == "" {
		x.Color = "#7430e2"
	}
	if x.Role == "" {
		x.Role = "sales"
	}
	x.IsActive = true
	x.UpdatedAt = now

	_, err := a.DB.Exec(`
		INSERT INTO appointment_agents (
			id, client_id, name, email, whatsapp, role, color, is_active, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id)
		DO UPDATE SET
			name=excluded.name,
			email=excluded.email,
			whatsapp=excluded.whatsapp,
			role=excluded.role,
			color=excluded.color,
			is_active=excluded.is_active,
			updated_at=excluded.updated_at
	`,
		x.ID,
		x.ClientID,
		x.Name,
		x.Email,
		x.Whatsapp,
		x.Role,
		x.Color,
		boolToInt(x.IsActive),
		x.CreatedAt,
		x.UpdatedAt,
	)

	return x, err
}

func (a *AgendaService) DeleteAgent(id string) error {
	_, err := a.DB.Exec(`DELETE FROM appointment_agents WHERE id=?`, id)
	return err
}

func (a *AgendaService) Metrics(clientID string) (map[string]int, error) {
	out := map[string]int{
		"today":     0,
		"scheduled": 0,
		"confirmed": 0,
		"completed": 0,
		"no_show":   0,
		"cancelled": 0,
	}

	_ = a.DB.QueryRow(`
		SELECT COUNT(*)
		FROM appointments
		WHERE client_id=?
		AND date(start_at)=date('now')
	`, clientID).Scan(&out["today"])

	rows, err := a.DB.Query(`
		SELECT status, COUNT(*)
		FROM appointments
		WHERE (?='' OR client_id=?)
		GROUP BY status
	`, clientID, clientID)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			out[status] = count
		}
	}

	return out, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}