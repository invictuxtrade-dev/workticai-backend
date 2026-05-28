package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type BotRuntime struct {
	BotID   string
	Client  *whatsmeow.Client
	StoreDB *sql.DB
}

type BotManager struct {
	DB      *sql.DB
	AI      *AIService
	Landing *LandingService
	Funnel  *FunnelService
	Billing *BillingService
	Agenda  *AgendaService
	BotsDir string

	mu       sync.Mutex
	runtimes map[string]*BotRuntime
}

func NewBotManager(db *sql.DB, ai *AIService, botsDir string) *BotManager {
	_ = os.MkdirAll(botsDir, 0o755)

	return &BotManager{
		DB:       db,
		AI:       ai,
		Landing:  NewLandingService(ai),
		Funnel:   nil,
		Billing:  NewBillingService(db),
		Agenda:   NewAgendaService(db),
		BotsDir:  botsDir,
		runtimes: map[string]*BotRuntime{},
	}
}

// formatPhoneForWhatsApp - Formatea un número de teléfono para WhatsApp correctamente
func formatPhoneForWhatsApp(phone string) string {
	if phone == "" {
		return ""
	}
	
	// Limpiar el número
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	phone = strings.ReplaceAll(phone, ".", "")
	
	// Eliminar cualquier prefijo duplicado
	for strings.HasPrefix(phone, "+") {
		phone = strings.TrimPrefix(phone, "+")
	}
	
	// Asegurar que solo tenga dígitos
	digits := ""
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits += string(r)
		}
	}
	
	// Si no hay dígitos, retornar vacío
	if digits == "" {
		return ""
	}
	
	// Agregar el prefijo + para WhatsApp
	return "+" + digits
}

// cleanPhoneForStorage - Limpia el teléfono para almacenar en BD (sin formato)
func cleanPhoneForStorage(phone string) string {
	if phone == "" {
		return ""
	}
	
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	phone = strings.ReplaceAll(phone, ".", "")
	phone = strings.TrimPrefix(phone, "+")
	
	// Solo dígitos
	digits := ""
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits += string(r)
		}
	}
	
	return digits
}

// cleanWhatsAppPhone - Limpia número de teléfono de JID de WhatsApp
func cleanWhatsAppPhone(raw string) string {
	if raw == "" {
		return ""
	}
	
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "+")
	raw = strings.Split(raw, "@")[0]
	raw = strings.Split(raw, ":")[0]
	raw = strings.ReplaceAll(raw, " ", "")
	raw = strings.ReplaceAll(raw, "-", "")
	raw = strings.ReplaceAll(raw, "(", "")
	raw = strings.ReplaceAll(raw, ")", "")
	
	onlyDigits := ""
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			onlyDigits += string(r)
		}
	}
	
	return onlyDigits
}

func (m *BotManager) AutoStartBots() error {
	rows, err := m.DB.Query(`SELECT id FROM bots`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var firstErr error

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if err := m.StartBot(id); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (m *BotManager) Metrics(clientID string) (models.Metrics, error) {
	var out models.Metrics
	var err error

	if clientID == "" {
		_ = m.DB.QueryRow(`SELECT COUNT(*) FROM clients`).Scan(&out.Clients)
		_ = m.DB.QueryRow(`SELECT COUNT(*) FROM bots`).Scan(&out.Bots)
		_ = m.DB.QueryRow(`SELECT COUNT(*) FROM leads`).Scan(&out.Leads)
		_ = m.DB.QueryRow(`SELECT COUNT(*) FROM leads WHERE stage='hot'`).Scan(&out.HotLeads)
		_ = m.DB.QueryRow(`SELECT COUNT(*) FROM leads WHERE stage='closed'`).Scan(&out.ClosedLeads)
		err = m.DB.QueryRow(`SELECT COUNT(*) FROM messages WHERE created_at >= datetime('now','-1 day')`).Scan(&out.Messages24h)
		return out, err
	}

	_ = m.DB.QueryRow(`SELECT COUNT(*) FROM clients WHERE id=?`, clientID).Scan(&out.Clients)
	_ = m.DB.QueryRow(`SELECT COUNT(*) FROM bots WHERE client_id=?`, clientID).Scan(&out.Bots)
	_ = m.DB.QueryRow(`SELECT COUNT(*) FROM leads WHERE bot_id IN (SELECT id FROM bots WHERE client_id=?)`, clientID).Scan(&out.Leads)
	_ = m.DB.QueryRow(`SELECT COUNT(*) FROM leads WHERE stage='hot' AND bot_id IN (SELECT id FROM bots WHERE client_id=?)`, clientID).Scan(&out.HotLeads)
	_ = m.DB.QueryRow(`SELECT COUNT(*) FROM leads WHERE stage='closed' AND bot_id IN (SELECT id FROM bots WHERE client_id=?)`, clientID).Scan(&out.ClosedLeads)
	err = m.DB.QueryRow(`SELECT COUNT(*) FROM messages WHERE created_at >= datetime('now','-1 day') AND bot_id IN (SELECT id FROM bots WHERE client_id=?)`, clientID).Scan(&out.Messages24h)

	return out, err
}

func (m *BotManager) ListClients() ([]models.Client, error) {
	rows, err := m.DB.Query(`SELECT id, name, email, phone, plan, status, created_at, updated_at FROM clients ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Client{}
	for rows.Next() {
		var c models.Client
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Plan, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (m *BotManager) CreateClient(name, email, phone, plan string) (models.Client, error) {
	now := time.Now()
	
	// Limpiar y formatear el teléfono para almacenamiento
	cleanPhone := cleanPhoneForStorage(phone)
	
	c := models.Client{
		ID:        uuid.NewString(),
		Name:      strings.TrimSpace(name),
		Email:     strings.TrimSpace(email),
		Phone:     cleanPhone,
		Plan:      strings.TrimSpace(plan),
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if c.Plan == "" {
		c.Plan = "pro"
	}

	_, err := m.DB.Exec(
		`INSERT INTO clients (id, name, email, phone, plan, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.Email, c.Phone, c.Plan, c.Status, c.CreatedAt, c.UpdatedAt,
	)
	return c, err
}

func (m *BotManager) UpdateClient(c models.Client) error {
	// Limpiar el teléfono antes de actualizar
	c.Phone = cleanPhoneForStorage(c.Phone)
	
	_, err := m.DB.Exec(
		`UPDATE clients SET name=?, email=?, phone=?, plan=?, status=?, updated_at=? WHERE id=?`,
		c.Name, c.Email, c.Phone, c.Plan, c.Status, time.Now(), c.ID,
	)
	return err
}

func (m *BotManager) DeleteClient(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("client id required")
	}

	bots, _ := m.ListBots(id)
	for _, b := range bots {
		_ = m.DeleteBot(b.ID)
	}

	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	deletes := []string{
		`DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE client_id=?)`,
		`DELETE FROM users WHERE client_id=?`,
		`DELETE FROM landing_pages WHERE client_id=?`,
		`DELETE FROM funnel_events WHERE client_id=?`,
		`DELETE FROM social_credentials WHERE client_id=?`,
		`DELETE FROM social_campaigns WHERE client_id=?`,
		`DELETE FROM social_posts WHERE client_id=?`,
		`DELETE FROM social_jobs WHERE client_id=?`,
		`DELETE FROM social_logs WHERE client_id=?`,
		`DELETE FROM ai_video_jobs WHERE client_id=?`,
		`DELETE FROM ads_campaigns WHERE client_id=?`,
		`DELETE FROM group_bots WHERE client_id=?`,
		`DELETE FROM facebook_group_targets WHERE client_id=?`,
		`DELETE FROM facebook_group_join_queue WHERE client_id=?`,
		`DELETE FROM facebook_group_activity_logs WHERE client_id=?`,
		`DELETE FROM group_growth_settings WHERE client_id=?`,
		`DELETE FROM subscriptions WHERE client_id=?`,
		`DELETE FROM appointments WHERE client_id=?`,
		`DELETE FROM appointment_settings WHERE client_id=?`,
		`DELETE FROM appointment_agents WHERE client_id=?`,
		`DELETE FROM appointment_notifications WHERE client_id=?`,
		`DELETE FROM appointment_pending WHERE client_id=?`,
		`DELETE FROM clients WHERE id=?`,
	}

	for _, q := range deletes {
		if _, err := tx.Exec(q, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m *BotManager) ListBots(clientID string) ([]models.Bot, error) {
	query := `SELECT id, client_id, name, phone, status, last_qr, created_at, updated_at FROM bots`
	args := []any{}
	if clientID != "" {
		query += ` WHERE client_id=?`
		args = append(args, clientID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := m.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Bot{}
	for rows.Next() {
		var b models.Bot
		if err := rows.Scan(&b.ID, &b.ClientID, &b.Name, &b.Phone, &b.Status, &b.LastQR, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		b.IsActive = b.Status == "connected" || b.Status == "waiting_qr"
		out = append(out, b)
	}
	return out, nil
}

func (m *BotManager) StopBot(id string) error {
	m.mu.Lock()
	rt := m.runtimes[id]
	if rt != nil && rt.Client != nil {
		rt.Client.Disconnect()
		delete(m.runtimes, id)
	}
	m.mu.Unlock()

	_, err := m.DB.Exec(`UPDATE bots SET status='stopped', last_qr='', updated_at=? WHERE id=?`, time.Now(), id)
	return err
}

func (m *BotManager) UpdateBot(id, name string) (models.Bot, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return models.Bot{}, errors.New("name required")
	}

	_, err := m.DB.Exec(`UPDATE bots SET name=?, updated_at=? WHERE id=?`, name, time.Now(), id)
	if err != nil {
		return models.Bot{}, err
	}

	return m.GetBot(id)
}

func (m *BotManager) DeleteBot(id string) error {
	_ = m.StopBot(id)

	_, _ = m.DB.Exec(`DELETE FROM messages WHERE bot_id=?`, id)
	_, _ = m.DB.Exec(`DELETE FROM leads WHERE bot_id=?`, id)
	_, _ = m.DB.Exec(`DELETE FROM bot_configs WHERE bot_id=?`, id)
	_, _ = m.DB.Exec(`DELETE FROM appointments WHERE bot_id=?`, id)
	_, _ = m.DB.Exec(`DELETE FROM appointment_settings WHERE bot_id=?`, id)
	_, _ = m.DB.Exec(`DELETE FROM appointment_pending WHERE bot_id=?`, id)

	_, err := m.DB.Exec(`DELETE FROM bots WHERE id=?`, id)
	return err
}

func (m *BotManager) CreateBot(role, clientID, name string) (models.Bot, error) {
	if strings.TrimSpace(clientID) == "" {
		return models.Bot{}, errors.New("client_id required")
	}

	if m.Billing != nil {
		if err := m.Billing.CheckLimit(role, clientID, "bots"); err != nil {
			return models.Bot{}, err
		}
	}

	now := time.Now()
	b := models.Bot{
		ID:        uuid.NewString(),
		ClientID:  clientID,
		Name:      strings.TrimSpace(name),
		Status:    "created",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if b.Name == "" {
		b.Name = "Bot " + now.Format("20060102150405")
	}

	_, err := m.DB.Exec(
		`INSERT INTO bots (id, client_id, name, phone, status, last_qr, created_at, updated_at) VALUES (?, ?, ?, '', ?, '', ?, ?)`,
		b.ID, b.ClientID, b.Name, b.Status, b.CreatedAt, b.UpdatedAt,
	)
	if err != nil {
		return models.Bot{}, err
	}

	if m.Billing != nil && role != "admin" {
		_ = m.Billing.IncrementUsage(clientID, "bots", 1)
	}

	_, _ = m.UpsertBotConfig(models.BotConfig{
		BotID:             b.ID,
		Tone:              "Profesional y cercano",
		CTAButtonText:     "Quiero avanzar",
		FallbackMessage:   "Gracias por escribirnos. En breve te ayudamos.",
		Temperature:       0.7,
		Model:             "gpt-4o-mini",
		FollowupEnabled:   true,
		FollowupDelayMins: 60,
		ReplyMode:         "manual",
		TemplateID:        "",
	})

	return b, nil
}

func (m *BotManager) GetBot(id string) (models.Bot, error) {
	var b models.Bot
	err := m.DB.QueryRow(
		`SELECT id, client_id, name, phone, status, last_qr, created_at, updated_at FROM bots WHERE id=?`,
		id,
	).Scan(&b.ID, &b.ClientID, &b.Name, &b.Phone, &b.Status, &b.LastQR, &b.CreatedAt, &b.UpdatedAt)

	b.IsActive = b.Status == "connected" || b.Status == "waiting_qr"
	return b, err
}

func (m *BotManager) StartBot(id string) error {
	m.mu.Lock()
	if rt, ok := m.runtimes[id]; ok && rt.Client != nil && rt.Client.IsConnected() {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	ctx := context.Background()
	botDir := m.BotDataDir(id)
	_ = os.MkdirAll(botDir, 0o755)

	storePath := filepath.Join(botDir, "whatsapp.db")
	logger := waLog.Stdout(m.DebugBotLabel(id), "INFO", true)

	container, err := sqlstore.New(ctx, "sqlite3", "file:"+storePath+"?_foreign_keys=on", logger)
	if err != nil {
		return err
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		deviceStore = container.NewDevice()
	}

	client := whatsmeow.NewClient(deviceStore, logger)

	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.QR:
			if len(v.Codes) > 0 {
				_ = m.SetBotQR(id, v.Codes[0])
			}

		case *events.PairSuccess:
			if client.Store.ID != nil {
				phone := client.Store.ID.User
				_ = m.AttachPhoneAndConnected(id, phone)
			}

		case *events.Connected:
			if client.Store.ID != nil {
				phone := client.Store.ID.User
				_ = m.AttachPhoneAndConnected(id, phone)
			}

		case *events.Disconnected:
			_ = m.MarkBotDisconnected(id)

		case *events.Message:
			if v.Info.IsFromMe {
				return
			}

			chatJID := v.Info.Chat.String()
			// Obtener teléfono REAL desde el chat JID
			phone := cleanWhatsAppPhone(chatJID)

			// fallback extremo
			if phone == "" {
				phone = cleanWhatsAppPhone(v.Info.Sender.String())
			}

			if phone == "" {
				phone = cleanWhatsAppPhone(v.Info.Sender.User)
			}

			displayName := v.Info.PushName
			text := extractIncomingText(v.Message)

			if strings.TrimSpace(text) == "" {
				return
			}

			logger.Infof("incoming message bot=%s chat=%s sender=%s text=%s", id, v.Info.Chat.String(), v.Info.Sender.String(), text)

			lead, isNewLead, err := m.EnsureLeadFromInbound(id, chatJID, displayName, phone, text)
			if err != nil {
				logger.Errorf("EnsureLeadFromInbound error: %v", err)
				return
			}

			bot, _ := m.GetBot(id)

			if err := m.SaveInboundMessage(id, chatJID, text); err != nil {
				logger.Errorf("SaveInboundMessage error: %v", err)
			}

			if m.Funnel != nil {
				_ = m.Funnel.TrackEvent(bot.ClientID, id, "", EventMessageReceived, text)
			}

			if m.Funnel != nil && isNewLead {
				_ = m.Funnel.TrackEvent(bot.ClientID, id, "", EventLeadCreated, lead.Phone)
			}

			cfg, _ := m.GetBotConfig(id)

			stage := detectLeadStage(text, lead.Stage)
			lead.Stage = stage
			_ = m.UpdateLeadStage(id, lead.ID, stage)

			if m.Funnel != nil {
				switch stage {
				case "qualified", "interested", "hot":
					_ = m.Funnel.TrackEvent(bot.ClientID, id, "", EventLeadQualified, stage)
				case "closed":
					_ = m.Funnel.TrackEvent(bot.ClientID, id, "", EventConversion, stage)
				}
			}

			// Verificar pendientes de agenda ANTES de detectar nueva intención
			if m.Agenda != nil {
				pending, _ := m.Agenda.GetPending(id, chatJID)

				if pending != nil {
					selected := ""

					switch strings.TrimSpace(strings.ToLower(text)) {
					case "1", "opcion 1", "la 1":
						selected = pending.Slot1
					case "2", "opcion 2", "la 2":
						selected = pending.Slot2
					case "3", "opcion 3", "la 3":
						selected = pending.Slot3
					}

					if selected != "" {
						startAt, parseErr := time.Parse(time.RFC3339, selected)
						if parseErr == nil {
							settings, _ := m.Agenda.GetSettings(bot.ClientID, id)

							duration := settings.DurationMins
							if duration <= 0 {
								duration = 30
							}

							endAt := startAt.Add(time.Duration(duration) * time.Minute)

							targetJID := v.Info.Chat

							contactPhone := phone
							if contactPhone == "" {
								contactPhone = "No disponible"
							}

							_, createErr := m.Agenda.CreateAppointment(Appointment{
								ClientID:     bot.ClientID,
								BotID:        id,
								LeadID:       lead.ID,
								ContactName:  lead.DisplayName,
								ContactPhone: contactPhone,
								Title:        "Cita IA",
								StartAt:      startAt,
								EndAt:        endAt,
								Status:       "scheduled",
								Source:       "ai",
							})

							if createErr == nil {
								_ = m.Agenda.CompletePending(pending.ID)

								if strings.TrimSpace(settings.NotifyWhatsapp) != "" {
									notifyMsg := fmt.Sprintf(
										"📅 Nueva cita agendada por Agenda AI\n\nLead: %s\nTeléfono: %s\nFecha: %s\nHora: %s\nBot: %s",
										lead.DisplayName,
										contactPhone,
										startAt.Format("02/01/2006"),
										startAt.Format("15:04"),
										bot.Name,
									)

									// Formatear el número de teléfono para WhatsApp correctamente
									notifyPhone := formatPhoneForWhatsApp(settings.NotifyWhatsapp)
									
									if notifyPhone == "" {
										logger.Errorf("invalid notification phone number: %s", settings.NotifyWhatsapp)
									} else {
										logger.Infof(
											"sending appointment notification bot=%s to=%s raw=%s",
											id,
											notifyPhone,
											settings.NotifyWhatsapp,
										)

										if err := m.SendText(id, notifyPhone, notifyMsg); err != nil {
											logger.Errorf(
												"error sending appointment notification to %s: %v",
												notifyPhone,
												err,
											)
										} else {
											logger.Infof(
												"appointment notification sent successfully to %s",
												notifyPhone,
											)
										}
									}
								}

								reply := fmt.Sprintf(
									"✅ Perfecto %s.\n\nTu cita quedó agendada para:\n%s",
									lead.DisplayName,
									startAt.Format("02 Jan 2006 03:04 PM"),
								)

								_, sendErr := client.SendMessage(
									context.Background(),
									targetJID,
									&waProto.Message{
										Conversation: proto.String(reply),
									},
								)

								if sendErr == nil {
									_ = m.SaveOutboundMessage(id, chatJID, reply)
								}
								return
							}
						}
					}
				}
			}

			// AGENDA INTENT DESPUÉS DE VERIFICAR PENDIENTES
			if m.Agenda != nil && looksLikeAppointmentIntent(text) {
				agendaReply, handled := m.HandleAgendaIntent(bot, lead, text)

				if handled && strings.TrimSpace(agendaReply) != "" {
					targetJID := v.Info.Chat

					_, sendErr := client.SendMessage(context.Background(), targetJID, &waProto.Message{
						Conversation: proto.String(agendaReply),
					})

					if sendErr != nil {
						logger.Errorf("error sending agenda reply to %s: %v", targetJID.String(), sendErr)
						return
					}

					_ = m.SaveOutboundMessage(id, chatJID, agendaReply)
					_ = m.SetLeadAIResult(id, lead.ID, "appointment_intent", text, agendaReply, nil)
					return
				}
			}

			reply := strings.TrimSpace(cfg.FallbackMessage)
			if reply == "" {
				reply = "Hola 👋 Gracias por escribirnos. En breve te ayudamos."
			}

			templateSvc := NewTemplateService(m.DB)

			var tpl models.Template
			var tplErr error

			if strings.TrimSpace(cfg.TemplateID) != "" {
				tpl, tplErr = templateSvc.GetByID(cfg.TemplateID)
			} else {
				businessType := inferBusinessType(cfg)
				category := detectTemplateCategory(text)
				tpl, tplErr = templateSvc.FindBestReplyTemplate(bot.ClientID, businessType, category, stage)
			}

			mode := strings.TrimSpace(cfg.ReplyMode)
			if mode == "" {
				mode = "manual"
			}

			switch mode {
			case "template_only":
				if tplErr == nil && strings.TrimSpace(tpl.MessageTemplate) != "" {
					reply = strings.TrimSpace(tpl.MessageTemplate)
				}

			case "template_ai":
				var promptParts []string

				if tplErr == nil {
					if strings.TrimSpace(tpl.PromptSnippet) != "" {
						promptParts = append(promptParts, "Usa esta guía de plantilla:\n"+strings.TrimSpace(tpl.PromptSnippet))
					}
					if strings.TrimSpace(tpl.MessageTemplate) != "" {
						promptParts = append(promptParts, "Mensaje base sugerido:\n"+strings.TrimSpace(tpl.MessageTemplate))
					}
				}

				if m.AI != nil {
					aiReply, err := m.AI.GenerateReply(context.Background(), lead, text, cfg, promptParts...)
					if err != nil {
						logger.Errorf("AI GenerateReply error: %v", err)
						if tplErr == nil && strings.TrimSpace(tpl.MessageTemplate) != "" {
							reply = strings.TrimSpace(tpl.MessageTemplate)
						}
					} else if strings.TrimSpace(aiReply) != "" {
						reply = strings.TrimSpace(aiReply)
					} else if tplErr == nil && strings.TrimSpace(tpl.MessageTemplate) != "" {
						reply = strings.TrimSpace(tpl.MessageTemplate)
					}
				} else if tplErr == nil && strings.TrimSpace(tpl.MessageTemplate) != "" {
					reply = strings.TrimSpace(tpl.MessageTemplate)
				}

			case "manual":
				fallthrough
			default:
				if m.AI != nil {
					aiReply, err := m.AI.GenerateReply(context.Background(), lead, text, cfg)
					if err != nil {
						logger.Errorf("AI GenerateReply error: %v", err)
					} else if strings.TrimSpace(aiReply) != "" {
						reply = strings.TrimSpace(aiReply)
					}
				}
			}

			if strings.TrimSpace(reply) == "" {
				reply = "Hola 👋 Gracias por escribirnos. ¿En qué puedo ayudarte?"
			}

			targetJID := v.Info.Chat
			_, sendErr := client.SendMessage(context.Background(), targetJID, &waProto.Message{
				Conversation: proto.String(reply),
			})
			if sendErr != nil {
				logger.Errorf("error sending auto-reply to %s: %v", targetJID.String(), sendErr)
				return
			}

			if err := m.SaveOutboundMessage(id, chatJID, reply); err != nil {
				logger.Errorf("SaveOutboundMessage error: %v", err)
			}

			intent := "inquiry"
			if mode == "template_only" || mode == "template_ai" {
				intent = detectTemplateCategory(text) + "_" + stage
			}

			if err := m.SetLeadAIResult(id, lead.ID, intent, text, reply, nil); err != nil {
				logger.Errorf("SetLeadAIResult error: %v", err)
			}
		}
	})

	m.mu.Lock()
	m.runtimes[id] = &BotRuntime{
		BotID:  id,
		Client: client,
	}
	m.mu.Unlock()

	if err := client.Connect(); err != nil {
		return err
	}

	if client.Store.ID == nil {
		_, _ = m.DB.Exec(`UPDATE bots SET status='waiting_qr', updated_at=? WHERE id=?`, time.Now(), id)
	} else {
		phone := client.Store.ID.User
		_ = m.AttachPhoneAndConnected(id, phone)
	}

	return nil
}

func extractIncomingText(msg *waProto.Message) string {
	if msg == nil {
		return ""
	}
	if t := msg.GetConversation(); t != "" {
		return t
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}

func (m *BotManager) GetQR(id string) (string, error) {
	var qr string
	err := m.DB.QueryRow(`SELECT COALESCE(last_qr,'') FROM bots WHERE id=?`, id).Scan(&qr)
	return qr, err
}

func (m *BotManager) GetBotConfig(botID string) (models.BotConfig, error) {
	var cfg models.BotConfig
	var followEnabled int

	err := m.DB.QueryRow(`
		SELECT bot_id, system_prompt, business_name, business_description, offer, target_audience,
		       tone, cta_button_text, cta_link, fallback_message, human_handoff_phone,
		       temperature, model, followup_enabled, followup_delay_mins, reply_mode, template_id, updated_at
		FROM bot_configs WHERE bot_id=?`,
		botID,
	).Scan(
		&cfg.BotID, &cfg.SystemPrompt, &cfg.BusinessName, &cfg.BusinessDescription,
		&cfg.Offer, &cfg.TargetAudience, &cfg.Tone, &cfg.CTAButtonText, &cfg.CTALink,
		&cfg.FallbackMessage, &cfg.HumanHandoffPhone, &cfg.Temperature, &cfg.Model,
		&followEnabled, &cfg.FollowupDelayMins, &cfg.ReplyMode, &cfg.TemplateID, &cfg.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.BotConfig{
			BotID:             botID,
			Tone:              "Profesional y cercano",
			CTAButtonText:     "Quiero avanzar",
			FallbackMessage:   "Gracias por escribirnos. En breve te ayudamos.",
			Temperature:       0.7,
			Model:             "gpt-4o-mini",
			FollowupEnabled:   true,
			FollowupDelayMins: 60,
			ReplyMode:         "manual",
			TemplateID:        "",
		}, nil
	}
	if err != nil {
		return models.BotConfig{}, err
	}

	cfg.FollowupEnabled = followEnabled == 1
	if strings.TrimSpace(cfg.ReplyMode) == "" {
		cfg.ReplyMode = "manual"
	}
	return cfg, nil
}

func (m *BotManager) UpsertBotConfig(cfg models.BotConfig) (models.BotConfig, error) {
	now := time.Now()

	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	if cfg.Tone == "" {
		cfg.Tone = "Profesional y cercano"
	}
	if cfg.CTAButtonText == "" {
		cfg.CTAButtonText = "Quiero avanzar"
	}
	if cfg.FallbackMessage == "" {
		cfg.FallbackMessage = "Gracias por escribirnos. En breve te ayudamos."
	}
	if cfg.FollowupDelayMins <= 0 {
		cfg.FollowupDelayMins = 60
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}
	if strings.TrimSpace(cfg.ReplyMode) == "" {
		cfg.ReplyMode = "manual"
	}

	followEnabled := 0
	if cfg.FollowupEnabled {
		followEnabled = 1
	}

	_, err := m.DB.Exec(`
		INSERT INTO bot_configs (
			bot_id, system_prompt, business_name, business_description, offer, target_audience,
			tone, cta_button_text, cta_link, fallback_message, human_handoff_phone,
			temperature, model, followup_enabled, followup_delay_mins, reply_mode, template_id, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(bot_id) DO UPDATE SET
			system_prompt=excluded.system_prompt,
			business_name=excluded.business_name,
			business_description=excluded.business_description,
			offer=excluded.offer,
			target_audience=excluded.target_audience,
			tone=excluded.tone,
			cta_button_text=excluded.cta_button_text,
			cta_link=excluded.cta_link,
			fallback_message=excluded.fallback_message,
			human_handoff_phone=excluded.human_handoff_phone,
			temperature=excluded.temperature,
			model=excluded.model,
			followup_enabled=excluded.followup_enabled,
			followup_delay_mins=excluded.followup_delay_mins,
			reply_mode=excluded.reply_mode,
			template_id=excluded.template_id,
			updated_at=excluded.updated_at
	`,
		cfg.BotID, cfg.SystemPrompt, cfg.BusinessName, cfg.BusinessDescription, cfg.Offer, cfg.TargetAudience,
		cfg.Tone, cfg.CTAButtonText, cfg.CTALink, cfg.FallbackMessage, cfg.HumanHandoffPhone,
		cfg.Temperature, cfg.Model, followEnabled, cfg.FollowupDelayMins, cfg.ReplyMode, cfg.TemplateID, now,
	)
	if err != nil {
		return models.BotConfig{}, err
	}

	cfg.UpdatedAt = now
	return cfg, nil
}

// SendText - Envía un mensaje de texto a través del bot de WhatsApp
func (m *BotManager) SendText(botID, number, message string) error {
	if botID == "" {
		return errors.New("bot_id is required")
	}
	
	// Formatear el número para WhatsApp
	formattedNumber := formatPhoneForWhatsApp(number)
	if formattedNumber == "" {
		return fmt.Errorf("invalid phone number: %s", number)
	}
	
	message = strings.TrimSpace(message)
	if message == "" {
		return errors.New("message is required")
	}
	
	m.mu.Lock()
	rt := m.runtimes[botID]
	m.mu.Unlock()
	
	if rt == nil || rt.Client == nil {
		return fmt.Errorf("bot %s not found or not initialized", botID)
	}
	
	if !rt.Client.IsConnected() {
		return fmt.Errorf("bot %s is not connected to WhatsApp", botID)
	}
	
	// Parsear el JID
	jid, err := types.ParseJID(formattedNumber)
	if err != nil {
		return fmt.Errorf("failed to parse JID for %s: %v", formattedNumber, err)
	}
	
	// Enviar el mensaje
	_, err = rt.Client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(message),
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	
	// Guardar el mensaje en la base de datos
	chatJID := jid.String()
	now := time.Now()
	
	// Verificar si existe el lead
	lead, err := m.findLeadByChat(botID, chatJID)
	if err == sql.ErrNoRows {
		// Crear un nuevo lead
		_, err = m.DB.Exec(`
			INSERT INTO leads (
				bot_id, chat_jid, display_name, phone, stage, last_intent, summary, tags,
				last_inbound_text, last_reply_text, followup_count, next_followup_at,
				created_at, updated_at, last_message_at
			) VALUES (?, ?, ?, ?, 'new', '', '', '', '', ?, 0, NULL, ?, ?, ?)
		`, botID, chatJID, number, cleanPhoneForStorage(number), message, now, now, now)
		if err != nil {
			return fmt.Errorf("failed to create lead: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to find lead: %v", err)
	}
	
	// Guardar el mensaje
	_, err = m.DB.Exec(
		`INSERT INTO messages (bot_id, chat_jid, direction, content, created_at) VALUES (?, ?, 'outbound', ?, ?)`,
		botID, chatJID, message, now,
	)
	if err != nil {
		return fmt.Errorf("failed to save message: %v", err)
	}
	
	// Actualizar el lead si existe
	if lead.ID > 0 {
		_, err = m.DB.Exec(`UPDATE leads SET last_reply_text=?, updated_at=?, last_message_at=? WHERE id=?`,
			message, now, now, lead.ID)
		if err != nil {
			return fmt.Errorf("failed to update lead: %v", err)
		}
	}
	
	return nil
}

func (m *BotManager) ListInboxLeads(clientID, botID string) ([]models.Lead, error) {
	query := `
		SELECT
			l.id, l.bot_id, b.client_id, b.name, c.name,
			l.chat_jid, l.display_name, l.phone, l.stage, l.last_intent, l.summary, l.tags,
			l.last_inbound_text, l.last_reply_text, l.followup_count, l.next_followup_at,
			l.created_at, l.updated_at, l.last_message_at
		FROM leads l
		INNER JOIN bots b ON b.id = l.bot_id
		INNER JOIN clients c ON c.id = b.client_id
		WHERE 1=1
	`
	args := []any{}

	if clientID != "" {
		query += ` AND b.client_id=?`
		args = append(args, clientID)
	}
	if botID != "" {
		query += ` AND l.bot_id=?`
		args = append(args, botID)
	}

	query += ` ORDER BY COALESCE(l.last_message_at, l.updated_at) DESC`

	rows, err := m.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Lead{}
	for rows.Next() {
		var l models.Lead
		var nextFollow, lastMsg sql.NullTime
		if err := rows.Scan(
			&l.ID, &l.BotID, &l.ClientID, &l.BotName, &l.ClientName,
			&l.ChatJID, &l.DisplayName, &l.Phone, &l.Stage, &l.LastIntent, &l.Summary, &l.Tags,
			&l.LastInboundText, &l.LastReplyText, &l.FollowupCount, &nextFollow,
			&l.CreatedAt, &l.UpdatedAt, &lastMsg,
		); err != nil {
			return nil, err
		}
		if nextFollow.Valid {
			t := nextFollow.Time
			l.NextFollowupAt = &t
		}
		if lastMsg.Valid {
			t := lastMsg.Time
			l.LastMessageAt = &t
		}
		out = append(out, l)
	}
	return out, nil
}

func (m *BotManager) ListLeads(botID string) ([]models.Lead, error) {
	rows, err := m.DB.Query(`
		SELECT id, bot_id, chat_jid, display_name, phone, stage, last_intent, summary, tags,
		       last_inbound_text, last_reply_text, followup_count, next_followup_at,
		       created_at, updated_at, last_message_at
		FROM leads
		WHERE bot_id=?
		ORDER BY COALESCE(last_message_at, updated_at) DESC
	`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Lead{}
	for rows.Next() {
		var l models.Lead
		var nextFollow, lastMsg sql.NullTime
		if err := rows.Scan(
			&l.ID, &l.BotID, &l.ChatJID, &l.DisplayName, &l.Phone, &l.Stage, &l.LastIntent,
			&l.Summary, &l.Tags, &l.LastInboundText, &l.LastReplyText, &l.FollowupCount,
			&nextFollow, &l.CreatedAt, &l.UpdatedAt, &lastMsg,
		); err != nil {
			return nil, err
		}
		if nextFollow.Valid {
			t := nextFollow.Time
			l.NextFollowupAt = &t
		}
		if lastMsg.Valid {
			t := lastMsg.Time
			l.LastMessageAt = &t
		}
		out = append(out, l)
	}
	return out, nil
}

func (m *BotManager) UpdateLeadStage(botID string, leadID int64, stage string) error {
	_, err := m.DB.Exec(`UPDATE leads SET stage=?, updated_at=? WHERE id=? AND bot_id=?`,
		strings.TrimSpace(stage), time.Now(), leadID, botID)
	return err
}

func (m *BotManager) LeadMessages(botID string, leadID int64) ([]models.Message, error) {
	var chatJID string
	err := m.DB.QueryRow(`SELECT chat_jid FROM leads WHERE id=? AND bot_id=?`, leadID, botID).Scan(&chatJID)
	if err != nil {
		return nil, err
	}

	rows, err := m.DB.Query(`
		SELECT id, bot_id, chat_jid, direction, content, created_at
		FROM messages
		WHERE bot_id=? AND chat_jid=?
		ORDER BY created_at ASC
	`, botID, chatJID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Message{}
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.BotID, &msg.ChatJID, &msg.Direction, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, nil
}

func (m *BotManager) SendToLead(botID string, leadID int64, message string) error {
	var chatJID string
	var phone string
	err := m.DB.QueryRow(`SELECT chat_jid, phone FROM leads WHERE id=? AND bot_id=?`, leadID, botID).Scan(&chatJID, &phone)
	if err != nil {
		return err
	}
	
	// Usar el número de teléfono del lead para enviar el mensaje
	return m.SendText(botID, phone, message)
}

func (m *BotManager) findLeadByChat(botID, chatJID string) (models.Lead, error) {
	var l models.Lead
	err := m.DB.QueryRow(`
		SELECT id, bot_id, chat_jid, display_name, phone, stage, last_intent, summary, tags,
		       last_inbound_text, last_reply_text, followup_count, created_at, updated_at
		FROM leads
		WHERE bot_id=? AND chat_jid=?
	`, botID, chatJID).
		Scan(&l.ID, &l.BotID, &l.ChatJID, &l.DisplayName, &l.Phone, &l.Stage, &l.LastIntent,
			&l.Summary, &l.Tags, &l.LastInboundText, &l.LastReplyText, &l.FollowupCount,
			&l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func (m *BotManager) EnsureLeadFromInbound(botID, chatJID, displayName, phone, incoming string) (models.Lead, bool, error) {
	now := time.Now()
	lead, err := m.findLeadByChat(botID, chatJID)
	if err == sql.ErrNoRows {
		res, err := m.DB.Exec(`
			INSERT INTO leads (
				bot_id, chat_jid, display_name, phone, stage, last_intent, summary, tags,
				last_inbound_text, last_reply_text, followup_count, next_followup_at,
				created_at, updated_at, last_message_at
			) VALUES (?, ?, ?, ?, 'new', '', '', '', ?, '', 0, NULL, ?, ?, ?)
		`, botID, chatJID, displayName, phone, incoming, now, now, now)
		if err != nil {
			return models.Lead{}, false, err
		}
		id, _ := res.LastInsertId()
		createdLead, err := m.getLeadByID(id)
		return createdLead, true, err
	}
	if err != nil {
		return models.Lead{}, false, err
	}

	_, err = m.DB.Exec(`UPDATE leads SET display_name=?, phone=?, last_inbound_text=?, updated_at=?, last_message_at=? WHERE id=?`,
		displayName, phone, incoming, now, now, lead.ID)
	if err != nil {
		return models.Lead{}, false, err
	}
	updatedLead, err := m.getLeadByID(lead.ID)
	return updatedLead, false, err
}

func (m *BotManager) SaveInboundMessage(botID, chatJID, content string) error {
	_, err := m.DB.Exec(
		`INSERT INTO messages (bot_id, chat_jid, direction, content, created_at) VALUES (?, ?, 'inbound', ?, ?)`,
		botID, chatJID, content, time.Now(),
	)
	return err
}

func (m *BotManager) SaveOutboundMessage(botID, chatJID, content string) error {
	_, err := m.DB.Exec(
		`INSERT INTO messages (bot_id, chat_jid, direction, content, created_at) VALUES (?, ?, 'outbound', ?, ?)`,
		botID, chatJID, content, time.Now(),
	)
	return err
}

func (m *BotManager) SetLeadAIResult(botID string, leadID int64, intent, summary, reply string, nextFollowup *time.Time) error {
	_, err := m.DB.Exec(`
		UPDATE leads
		SET last_intent=?, summary=?, last_reply_text=?, next_followup_at=?, updated_at=?, last_message_at=?
		WHERE id=? AND bot_id=?
	`, intent, summary, reply, nextFollowup, time.Now(), time.Now(), leadID, botID)
	return err
}

func (m *BotManager) getLeadByID(id int64) (models.Lead, error) {
	var l models.Lead
	var nextFollow, lastMsg sql.NullTime
	err := m.DB.QueryRow(`
		SELECT id, bot_id, chat_jid, display_name, phone, stage, last_intent, summary, tags,
		       last_inbound_text, last_reply_text, followup_count, next_followup_at,
		       created_at, updated_at, last_message_at
		FROM leads
		WHERE id=?
	`, id).
		Scan(&l.ID, &l.BotID, &l.ChatJID, &l.DisplayName, &l.Phone, &l.Stage, &l.LastIntent,
			&l.Summary, &l.Tags, &l.LastInboundText, &l.LastReplyText, &l.FollowupCount,
			&nextFollow, &l.CreatedAt, &l.UpdatedAt, &lastMsg)
	if err != nil {
		return l, err
	}
	if nextFollow.Valid {
		t := nextFollow.Time
		l.NextFollowupAt = &t
	}
	if lastMsg.Valid {
		t := lastMsg.Time
		l.LastMessageAt = &t
	}
	return l, nil
}

func (m *BotManager) AttachPhoneAndConnected(botID, phone string) error {
	_, err := m.DB.Exec(`UPDATE bots SET phone=?, status='connected', last_qr='', updated_at=? WHERE id=?`,
		phone, time.Now(), botID)
	return err
}

func (m *BotManager) SetBotQR(botID, qr string) error {
	_, err := m.DB.Exec(`UPDATE bots SET last_qr=?, status='waiting_qr', updated_at=? WHERE id=?`,
		qr, time.Now(), botID)
	return err
}

func (m *BotManager) MarkBotDisconnected(botID string) error {
	_, err := m.DB.Exec(`UPDATE bots SET status='disconnected', updated_at=? WHERE id=?`, time.Now(), botID)
	return err
}

func (m *BotManager) BotDataDir(botID string) string {
	return filepath.Join(m.BotsDir, botID)
}

func (m *BotManager) DebugBotLabel(botID string) string {
	return fmt.Sprintf("bot:%s", botID)
}

func (m *BotManager) HandleAgendaIntent(bot models.Bot, lead models.Lead, incoming string) (string, bool) {
	if m.Agenda == nil {
		return "", false
	}

	settings, err := m.Agenda.GetSettings(bot.ClientID, bot.ID)
	if err != nil || !settings.Enabled {
		return "", false
	}

	preferredAt := detectPreferredDateTime(incoming)

	score := 75
	if lead.Stage == "hot" {
		score = 90
	}

	ap, slots, err := m.Agenda.CreateAutomaticAppointment(AutoAppointmentRequest{
		ClientID:     bot.ClientID,
		BotID:        bot.ID,
		LeadID:       lead.ID,
		ChatJID:      lead.ChatJID,
		ContactName:  lead.DisplayName,
		ContactPhone: lead.Phone,
		Message:      incoming,
		PreferredAt:  preferredAt,
		AISummary:    "El lead mostró intención de agendar una cita desde WhatsApp.",
		LeadScore:    score,
	})

	if err != nil {
		if len(slots) >= 3 {
			pendingID := uuid.NewString()
			pending := PendingAppointment{
				ID:        pendingID,
				ClientID:  bot.ClientID,
				BotID:     bot.ID,
				LeadID:    lead.ID,
				ChatJID:   lead.ChatJID,
				Slot1:     slots[0].StartAt.Format(time.RFC3339),
				Slot2:     slots[1].StartAt.Format(time.RFC3339),
				Slot3:     slots[2].StartAt.Format(time.RFC3339),
				Status:    "pending",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(30 * time.Minute),
			}

			_ = m.Agenda.SavePending(pending)

			var b strings.Builder
			b.WriteString("Claro 😊 puedo ayudarte a agendar una cita.\n\n")
			b.WriteString("Tengo estos espacios disponibles:\n\n")

			for i, slot := range slots {
				b.WriteString(fmt.Sprintf("%d. %s\n", i+1, slot.StartAt.Format("02/01/2006 15:04")))
			}

			b.WriteString("\nRespóndeme con el horario que prefieres (1, 2 o 3) y te lo agendo.")
			return b.String(), true
		}

		return "Claro 😊 puedo ayudarte a agendar una cita. ¿Qué día y hora te gustaría?", true
	}

	reply := fmt.Sprintf(
		"Perfecto 😊 tu cita quedó agendada.\n\n📅 Fecha: %s\n⏰ Hora: %s\n\nTe enviaremos un recordatorio antes de la cita.",
		ap.StartAt.Format("02/01/2006"),
		ap.StartAt.Format("15:04"),
	)

	return reply, true
}

func inferBusinessType(cfg models.BotConfig) string {
	text := strings.ToLower(strings.TrimSpace(
		cfg.BusinessName + " " +
			cfg.BusinessDescription + " " +
			cfg.Offer + " " +
			cfg.SystemPrompt,
	))

	switch {
	case strings.Contains(text, "copy trading"),
		strings.Contains(text, "trading"),
		strings.Contains(text, "xauusd"),
		strings.Contains(text, "forex"),
		strings.Contains(text, "oro"):
		return "trading"

	case strings.Contains(text, "mlm"),
		strings.Contains(text, "multinivel"),
		strings.Contains(text, "network marketing"):
		return "mlm"

	case strings.Contains(text, "inmobili"),
		strings.Contains(text, "propiedad"),
		strings.Contains(text, "apartamento"),
		strings.Contains(text, "casa"),
		strings.Contains(text, "lote"),
		strings.Contains(text, "real estate"):
		return "real_estate"

	case strings.Contains(text, "software"),
		strings.Contains(text, "saas"),
		strings.Contains(text, "crm"),
		strings.Contains(text, "plataforma"),
		strings.Contains(text, "app"):
		return "software"

	default:
		return "general"
	}
}

func detectTemplateCategory(incoming string) string {
	text := strings.ToLower(strings.TrimSpace(incoming))

	switch {
	case strings.Contains(text, "soporte"),
		strings.Contains(text, "ayuda"),
		strings.Contains(text, "error"),
		strings.Contains(text, "problema"),
		strings.Contains(text, "no funciona"):
		return "support"

	case strings.Contains(text, "seguimiento"),
		strings.Contains(text, "sigues ahí"),
		strings.Contains(text, "sigues interesado"):
		return "followup"

	default:
		return "sales"
	}
}

func detectLeadStage(incoming string, current string) string {
	text := strings.ToLower(strings.TrimSpace(incoming))
	current = strings.TrimSpace(strings.ToLower(current))

	if strings.Contains(text, "quiero empezar") ||
		strings.Contains(text, "como empiezo") ||
		strings.Contains(text, "cómo empiezo") ||
		strings.Contains(text, "me interesa") ||
		strings.Contains(text, "agendar") ||
		strings.Contains(text, "demo") ||
		strings.Contains(text, "activar") ||
		strings.Contains(text, "quiero entrar") {
		return "hot"
	}

	if strings.Contains(text, "precio") ||
		strings.Contains(text, "cuanto") ||
		strings.Contains(text, "cuánto") ||
		strings.Contains(text, "capital") ||
		strings.Contains(text, "valor") ||
		strings.Contains(text, "costo") ||
		strings.Contains(text, "planes") {
		return "interested"
	}

	if strings.Contains(text, "hola") ||
		strings.Contains(text, "buenas") ||
		strings.Contains(text, "informacion") ||
		strings.Contains(text, "información") ||
		strings.Contains(text, "info") ||
		current == "" {
		return "new"
	}

	if current != "" {
		return current
	}

	return "new"
}

func looksLikeAppointmentIntent(incoming string) bool {
	text := strings.ToLower(strings.TrimSpace(incoming))

	keywords := []string{
		"agenda",
		"agendar",
		"cita",
		"reunión",
		"reunion",
		"llamada",
		"demo",
		"asesoría",
		"asesoria",
		"hablar con alguien",
		"quiero hablar",
		"programar",
		"separar espacio",
		"reservar",
		"mañana",
		"manana",
		"pasado mañana",
		"pasado manana",
		"viernes",
		"lunes",
		"martes",
		"miércoles",
		"miercoles",
		"jueves",
		"sábado",
		"sabado",
		"domingo",
	}

	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}

	return false
}

func detectPreferredDateTime(incoming string) string {
	text := strings.ToLower(strings.TrimSpace(incoming))
	now := time.Now()

	hour := 0
	minute := 0

	for h := 6; h <= 22; h++ {
		if strings.Contains(text, fmt.Sprintf("%d:", h)) ||
			strings.Contains(text, fmt.Sprintf("%d ", h)) ||
			strings.Contains(text, fmt.Sprintf("a las %d", h)) {
			hour = h
			break
		}
	}

	if hour == 0 {
		if strings.Contains(text, "mañana") || strings.Contains(text, "manana") {
			hour = 10
		} else {
			return ""
		}
	}

	if strings.Contains(text, "media") || strings.Contains(text, ":30") {
		minute = 30
	}

	target := now

	switch {
	case strings.Contains(text, "pasado mañana") || strings.Contains(text, "pasado manana"):
		target = now.AddDate(0, 0, 2)

	case strings.Contains(text, "mañana") || strings.Contains(text, "manana"):
		target = now.AddDate(0, 0, 1)

	case strings.Contains(text, "hoy"):
		target = now

	default:
		candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.Local)
		if candidate.Before(now.Add(2 * time.Hour)) {
			target = now.AddDate(0, 0, 1)
		}
	}

	finalTime := time.Date(target.Year(), target.Month(), target.Day(), hour, minute, 0, 0, time.Local)

	return finalTime.Format(time.RFC3339)
}