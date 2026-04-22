package services

import "database/sql"

const (
	EventLandingView     = "landing_view"
	EventWhatsAppClick   = "whatsapp_click"
	EventMessageReceived = "message_received"
	EventLeadCreated     = "lead_created"
	EventLeadQualified   = "lead_qualified"
	EventConversion      = "conversion"
)

type FunnelService struct {
	DB *sql.DB
}

func NewFunnelService(db *sql.DB) *FunnelService {
	return &FunnelService{DB: db}
}

func (f *FunnelService) TrackEvent(clientID, botID, landingID, eventType, metadata string) error {
	_, err := f.DB.Exec(`
		INSERT INTO funnel_events (client_id, bot_id, landing_id, event_type, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, clientID, botID, landingID, eventType, metadata)
	return err
}

func (f *FunnelService) Metrics(clientID string) (map[string]int, error) {
	rows, err := f.DB.Query(`
		SELECT event_type, COUNT(*)
		FROM funnel_events
		WHERE (? = '' OR client_id = ?)
		GROUP BY event_type
	`, clientID, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int{
		EventLandingView:     0,
		EventWhatsAppClick:   0,
		EventMessageReceived: 0,
		EventLeadCreated:     0,
		EventLeadQualified:   0,
		EventConversion:      0,
	}

	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, err
		}
		out[eventType] = count
	}

	return out, nil
}