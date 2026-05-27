package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	
	"go.mau.fi/whatsmeow/types"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

type AgendaScheduler struct {
	DB      *sql.DB
	Agenda  *AgendaService
	Manager *BotManager
	stopCh  chan struct{}
}

func NewAgendaScheduler(db *sql.DB, agenda *AgendaService, manager *BotManager) *AgendaScheduler {
	return &AgendaScheduler{
		DB:      db,
		Agenda:  agenda,
		Manager: manager,
		stopCh:  make(chan struct{}),
	}
}

func (s *AgendaScheduler) Start() {
	ticker := time.NewTicker(60 * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				s.run()
			case <-s.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *AgendaScheduler) Stop() {
	close(s.stopCh)
}

func (s *AgendaScheduler) run() {
	s.sendReminderWindow(24*time.Hour, "24h")
	s.sendReminderWindow(1*time.Hour, "1h")
	s.sendReminderWindow(15*time.Minute, "15m")
}

func (s *AgendaScheduler) sendReminderWindow(before time.Duration, label string) {
	now := time.Now()
	from := now.Add(before - 90*time.Second)
	to := now.Add(before + 90*time.Second)

	rows, err := s.DB.Query(`
		SELECT
			a.id,
			a.client_id,
			a.bot_id,
			a.contact_name,
			a.contact_phone,
			a.start_at,
			a.timezone,
			st.notify_whatsapp
		FROM appointments a
		LEFT JOIN appointment_settings st
			ON st.client_id = a.client_id
			AND st.bot_id = a.bot_id
		WHERE a.status IN ('scheduled','confirmed')
		AND a.start_at BETWEEN ? AND ?
		AND NOT EXISTS (
			SELECT 1 FROM appointment_notifications n
			WHERE n.appointment_id = a.id
			AND n.channel = ?
			AND n.status = 'sent'
		)
	`, from, to, "reminder_"+label)
	if err != nil {
		return
	}
	defer rows.Close()

	type reminder struct {
		ID             string
		ClientID       string
		BotID          string
		ContactName    string
		ContactPhone   string
		StartAt        time.Time
		Timezone       string
		NotifyWhatsapp string
	}

	items := []reminder{}

	for rows.Next() {
		var x reminder
		if err := rows.Scan(
			&x.ID,
			&x.ClientID,
			&x.BotID,
			&x.ContactName,
			&x.ContactPhone,
			&x.StartAt,
			&x.Timezone,
			&x.NotifyWhatsapp,
		); err == nil {
			items = append(items, x)
		}
	}

	for _, item := range items {
		msg := fmt.Sprintf(
			"⏰ Recordatorio de cita\n\nHola %s 👋\nTu cita está programada para: %s.\n\nTe esperamos.",
			item.ContactName,
			item.StartAt.Format("02/01/2006 15:04"),
		)

		errLead := s.sendWhatsApp(item.BotID, item.ContactPhone, msg)

		if errLead == nil {
			_ = s.Agenda.CreateNotification(item.ClientID, item.ID, "reminder_"+label, msg)

			_, _ = s.DB.Exec(`
				UPDATE appointment_notifications
				SET status='sent', sent_at=?
				WHERE appointment_id=?
				AND channel=?
				AND status='pending'
			`, time.Now(), item.ID, "reminder_"+label)
		}

		if strings.TrimSpace(item.NotifyWhatsapp) != "" {
			adminMsg := fmt.Sprintf(
				"📅 Recordatorio Agenda AI\n\nLa cita con %s será en %s.\nHora: %s",
				item.ContactName,
				label,
				item.StartAt.Format("02/01/2006 15:04"),
			)
			_ = s.sendWhatsApp(item.BotID, item.NotifyWhatsapp, adminMsg)
		}
	}
}

func (s *AgendaScheduler) sendWhatsApp(botID, phone, text string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return fmt.Errorf("empty phone")
	}

	if s.Manager == nil {
		return fmt.Errorf("bot manager not configured")
	}

	rt, ok := s.Manager.runtimes[botID]
	if !ok || rt == nil || rt.Client == nil {
		return fmt.Errorf("bot not connected")
	}

	clean := strings.TrimPrefix(phone, "+")
	clean = strings.ReplaceAll(clean, " ", "")
	clean = strings.ReplaceAll(clean, "-", "")

	jid := types.NewJID(clean, "s.whatsapp.net")

	_, err := rt.Client.SendMessage(
	context.Background(),
	jid,
	&waProto.Message{
		Conversation: proto.String(text),
		},
	)

	return err
}