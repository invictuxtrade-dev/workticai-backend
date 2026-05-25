package services

import (
	"database/sql"
	"errors"
	"fmt"
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
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
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
	StartTime             string    `json:"start_time"`
	EndTime               string    `json:"end_time"`
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

type AvailabilitySlot struct {
	StartAt  time.Time `json:"start_at"`
	EndAt    time.Time `json:"end_at"`
	Label    string    `json:"label"`
	Timezone string    `json:"timezone"`
}

type AutoAppointmentRequest struct {
	ClientID     string `json:"client_id"`
	BotID        string `json:"bot_id"`
	LeadID       int64  `json:"lead_id"`
	ChatJID      string `json:"chat_jid"`
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
	Message      string `json:"message"`
	PreferredAt  string `json:"preferred_at"`
	AISummary    string `json:"ai_summary"`
	LeadScore    int    `json:"lead_score"`
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
			&x.ID,
			&x.ClientID,
			&x.BotID,
			&x.LeadID,
			&x.AgentID,
			&x.Title,
			&x.ContactName,
			&x.ContactPhone,
			&x.ContactEmail,
			&x.Status,
			&x.Source,
			&x.MeetingType,
			&x.MeetingLink,
			&x.Location,
			&x.Notes,
			&x.AISummary,
			&x.LeadScore,
			&x.StartAt,
			&x.EndAt,
			&x.Timezone,
			&x.ReminderSentAt,
			&x.CreatedAt,
			&x.UpdatedAt,
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
	if x.StartAt.IsZero() {
		return Appointment{}, errors.New("start_at required")
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
		x.ID,
		x.ClientID,
		x.BotID,
		x.LeadID,
		x.AgentID,
		x.Title,
		x.ContactName,
		x.ContactPhone,
		x.ContactEmail,
		x.Status,
		x.Source,
		x.MeetingType,
		x.MeetingLink,
		x.Location,
		x.Notes,
		x.AISummary,
		x.LeadScore,
		x.StartAt,
		x.EndAt,
		x.Timezone,
		x.ReminderSentAt,
		x.CreatedAt,
		x.UpdatedAt,
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

	var todayCount int

	_ = a.DB.QueryRow(`
		SELECT COUNT(*)
		FROM appointments
		WHERE (?='' OR client_id=?)
		AND date(start_at)=date('now')
	`, clientID, clientID).Scan(&todayCount)

	out["today"] = todayCount

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

func (a *AgendaService) IsAvailable(clientID, botID string, startAt, endAt time.Time) (bool, string, error) {
	if strings.TrimSpace(clientID) == "" {
		return false, "client_id required", nil
	}
	if strings.TrimSpace(botID) == "" {
		return false, "bot_id required", nil
	}
	if startAt.IsZero() || endAt.IsZero() || !endAt.After(startAt) {
		return false, "invalid appointment time", nil
	}

	settings, err := a.GetSettings(clientID, botID)
	if err != nil {
		return false, "agenda settings not configured", nil
	}

	if !settings.Enabled {
		return false, "agenda ai disabled for this bot", nil
	}

	weekday := strings.ToLower(startAt.Weekday().String()[:3])
	if !strings.Contains(strings.ToLower(settings.AvailableDays), weekday) {
		return false, "day not available", nil
	}

	startClock := startAt.Format("15:04")
	endClock := endAt.Format("15:04")

	if startClock < settings.StartTime || endClock > settings.EndTime {
		return false, "outside available hours", nil
	}

	buffer := time.Duration(settings.BufferMins) * time.Minute
	from := startAt.Add(-buffer)
	to := endAt.Add(buffer)

	var conflicts int
	err = a.DB.QueryRow(`
		SELECT COUNT(*)
		FROM appointments
		WHERE client_id=?
		AND bot_id=?
		AND status NOT IN ('cancelled', 'completed', 'no_show')
		AND start_at < ?
		AND end_at > ?
	`, clientID, botID, to, from).Scan(&conflicts)

	if err != nil {
		return false, "availability check failed", err
	}

	if conflicts > 0 {
		return false, "slot already booked", nil
	}

	return true, "", nil
}

func (a *AgendaService) NextAvailableSlots(clientID, botID string, limit int) ([]AvailabilitySlot, error) {
	if limit <= 0 {
		limit = 5
	}

	settings, err := a.GetSettings(clientID, botID)
	if err != nil {
		return []AvailabilitySlot{}, err
	}

	duration := time.Duration(settings.DurationMins) * time.Minute
	step := duration + time.Duration(settings.BufferMins)*time.Minute

	out := []AvailabilitySlot{}
	now := time.Now()

	for d := 0; d < 14 && len(out) < limit; d++ {
		day := now.AddDate(0, 0, d)
		weekday := strings.ToLower(day.Weekday().String()[:3])

		if !strings.Contains(strings.ToLower(settings.AvailableDays), weekday) {
			continue
		}

		startDay, err := time.ParseInLocation("2006-01-02 15:04", day.Format("2006-01-02")+" "+settings.StartTime, time.Local)
		if err != nil {
			continue
		}

		endDay, err := time.ParseInLocation("2006-01-02 15:04", day.Format("2006-01-02")+" "+settings.EndTime, time.Local)
		if err != nil {
			continue
		}

		for slotStart := startDay; slotStart.Add(duration).Before(endDay) || slotStart.Add(duration).Equal(endDay); slotStart = slotStart.Add(step) {
			if slotStart.Before(now.Add(15 * time.Minute)) {
				continue
			}

			slotEnd := slotStart.Add(duration)

			ok, _, _ := a.IsAvailable(clientID, botID, slotStart, slotEnd)
			if ok {
				out = append(out, AvailabilitySlot{
					StartAt:  slotStart,
					EndAt:    slotEnd,
					Label:    slotStart.Format("Mon 02 Jan 15:04"),
					Timezone: settings.Timezone,
				})
			}

			if len(out) >= limit {
				break
			}
		}
	}

	return out, nil
}

func (a *AgendaService) CreateNotification(clientID, appointmentID, channel, message string) error {
	now := time.Now()

	_, err := a.DB.Exec(`
		INSERT INTO appointment_notifications (
			id, client_id, appointment_id, channel, status, message, error, created_at, sent_at
		)
		VALUES (?, ?, ?, ?, 'pending', ?, '', ?, NULL)
	`,
		uuid.NewString(),
		clientID,
		appointmentID,
		channel,
		message,
		now,
	)

	return err
}

func (a *AgendaService) CreateAutomaticAppointment(req AutoAppointmentRequest) (Appointment, []AvailabilitySlot, error) {
	if strings.TrimSpace(req.ClientID) == "" {
		return Appointment{}, nil, errors.New("client_id required")
	}
	if strings.TrimSpace(req.BotID) == "" {
		return Appointment{}, nil, errors.New("bot_id required")
	}

	settings, err := a.GetSettings(req.ClientID, req.BotID)
	if err != nil || !settings.Enabled {
		return Appointment{}, nil, errors.New("agenda ai disabled or not configured")
	}

	var startAt time.Time

	if strings.TrimSpace(req.PreferredAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.PreferredAt)
		if err == nil {
			startAt = parsed
		}
	}

	if startAt.IsZero() {
		slots, err := a.NextAvailableSlots(req.ClientID, req.BotID, 3)
		if err != nil {
			return Appointment{}, nil, err
		}
		return Appointment{}, slots, errors.New("needs_time_confirmation")
	}

	endAt := startAt.Add(time.Duration(settings.DurationMins) * time.Minute)

	ok, reason, err := a.IsAvailable(req.ClientID, req.BotID, startAt, endAt)
	if err != nil {
		return Appointment{}, nil, err
	}

	if !ok {
		slots, _ := a.NextAvailableSlots(req.ClientID, req.BotID, 3)
		return Appointment{}, slots, fmt.Errorf(reason)
	}

	ap := Appointment{
		ClientID:     req.ClientID,
		BotID:        req.BotID,
		LeadID:       req.LeadID,
		Title:        "Cita agendada por IA",
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		Status:       "scheduled",
		Source:       "whatsapp_ai",
		MeetingType:  settings.Goal,
		Notes:        req.Message,
		AISummary:    req.AISummary,
		LeadScore:    req.LeadScore,
		StartAt:      startAt,
		EndAt:        endAt,
		Timezone:     settings.Timezone,
	}

	ap, err = a.CreateAppointment(ap)
	if err != nil {
		return Appointment{}, nil, err
	}

	_ = a.CreateNotification(req.ClientID, ap.ID, "dashboard", "Nueva cita agendada automáticamente por Agenda AI")

	if strings.TrimSpace(settings.NotifyEmail) != "" {
		_ = a.CreateNotification(req.ClientID, ap.ID, "email", "Nueva cita Agenda AI para "+ap.ContactName)
	}

	if strings.TrimSpace(settings.NotifyWhatsapp) != "" {
		_ = a.CreateNotification(req.ClientID, ap.ID, "whatsapp", "Nueva cita Agenda AI para "+ap.ContactName)
	}

	return ap, nil, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}