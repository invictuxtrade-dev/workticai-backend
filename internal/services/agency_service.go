package services

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Agency struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Email               string     `json:"email"`
	Phone               string     `json:"phone"`
	Status              string     `json:"status"`
	LogoURL             string     `json:"logo_url"`
	BrandColor          string     `json:"brand_color"`
	SecondaryColor      string     `json:"secondary_color"`
	LoginBackground     string     `json:"login_background"`
	FaviconURL          string     `json:"favicon_url"`
	LoginTitle          string     `json:"login_title"`
	LoginSubtitle       string     `json:"login_subtitle"`
	BrandName           string     `json:"brand_name"`
	CustomDomain        string     `json:"custom_domain"`
	Subdomain           string     `json:"subdomain"`
	ContractTitle       string     `json:"contract_title"`
	ContractBody        string     `json:"contract_body"`
	ContractStatus      string     `json:"contract_status"`
	ContractSignedBy    string     `json:"contract_signed_by"`
	ContractSignedEmail string     `json:"contract_signed_email"`
	ContractSignature   string     `json:"contract_signature"`
	ContractSignedIP    string     `json:"contract_signed_ip"`
	ContractSignedAt    *time.Time `json:"contract_signed_at"`
	Notes               string     `json:"notes"`
	MonthlyFee          float64    `json:"monthly_fee"`
	PlanEquivalent      string     `json:"plan_equivalent"`
	StartsAt            *time.Time `json:"starts_at"`
	ExpiresAt           *time.Time `json:"expires_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type AgencyPlanPrice struct {
	ID           string    `json:"id"`
	AgencyID     string    `json:"agency_id"`
	PlanSlug     string    `json:"plan_slug"`
	NormalPrice  float64   `json:"normal_price"`
	AgencyPrice  float64   `json:"agency_price"`
	BillingCycle string    `json:"billing_cycle"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AgencyService struct {
	DB *sql.DB
}

func NewAgencyService(db *sql.DB) *AgencyService {
	return &AgencyService{DB: db}
}

func normalizeAgency(a Agency) Agency {
	a.Name = strings.TrimSpace(a.Name)
	a.Email = strings.TrimSpace(a.Email)
	a.Phone = strings.TrimSpace(a.Phone)
	a.LogoURL = strings.TrimSpace(a.LogoURL)
	a.BrandColor = strings.TrimSpace(a.BrandColor)
	a.SecondaryColor = strings.TrimSpace(a.SecondaryColor)
	a.LoginBackground = strings.TrimSpace(a.LoginBackground)
	a.FaviconURL = strings.TrimSpace(a.FaviconURL)
	a.LoginTitle = strings.TrimSpace(a.LoginTitle)
	a.LoginSubtitle = strings.TrimSpace(a.LoginSubtitle)
	a.BrandName = strings.TrimSpace(a.BrandName)
	a.CustomDomain = strings.TrimSpace(a.CustomDomain)
	a.Subdomain = strings.TrimSpace(a.Subdomain)
	a.ContractTitle = strings.TrimSpace(a.ContractTitle)
	a.Notes = strings.TrimSpace(a.Notes)
	a.PlanEquivalent = strings.TrimSpace(a.PlanEquivalent)

	if a.Status == "" {
		a.Status = "pending"
	}
	if a.BrandColor == "" {
		a.BrandColor = "#7430e2"
	}
	if a.SecondaryColor == "" {
		a.SecondaryColor = "#0f172a"
	}
	if a.BrandName == "" {
		a.BrandName = a.Name
	}
	if a.PlanEquivalent == "" {
		a.PlanEquivalent = "business"
	}
	if a.ContractStatus == "" {
		a.ContractStatus = "draft"
	}
	if a.ContractTitle == "" {
		a.ContractTitle = "Contrato Comercial de Agencia Worktic AI"
	}
	if a.LoginTitle == "" {
		a.LoginTitle = "Bienvenido"
	}
	if a.LoginSubtitle == "" {
		a.LoginSubtitle = "Plataforma de automatización comercial"
	}
	if a.ContractBody == "" {
		a.ContractBody = DefaultAgencyContract(a.Name)
	}

	return a
}

func (s *AgencyService) Create(a Agency) (Agency, error) {
	now := time.Now()

	a = normalizeAgency(a)

	if a.Name == "" {
		return Agency{}, errors.New("agency name required")
	}

	if a.ID == "" {
		a.ID = uuid.NewString()
	}

	a.CreatedAt = now
	a.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO agencies (
			id, name, email, phone, status,
			logo_url, brand_color, secondary_color, login_background, favicon_url,
			login_title, login_subtitle, brand_name,
			custom_domain, subdomain,
			contract_title, contract_body, contract_status,
			contract_signed_by, contract_signed_email, contract_signature, contract_signed_ip,
			contract_signed_at, notes, monthly_fee, plan_equivalent, starts_at, expires_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		a.ID, a.Name, a.Email, a.Phone, a.Status,
		a.LogoURL, a.BrandColor, a.SecondaryColor, a.LoginBackground, a.FaviconURL,
		a.LoginTitle, a.LoginSubtitle, a.BrandName,
		a.CustomDomain, a.Subdomain,
		a.ContractTitle, a.ContractBody, a.ContractStatus,
		a.ContractSignedBy, a.ContractSignedEmail, a.ContractSignature, a.ContractSignedIP,
		a.ContractSignedAt, a.Notes, a.MonthlyFee, a.PlanEquivalent, a.StartsAt, a.ExpiresAt,
		a.CreatedAt, a.UpdatedAt,
	)

	return a, err
}

func scanAgency(scanner interface {
	Scan(dest ...any) error
}) (Agency, error) {
	var a Agency

	err := scanner.Scan(
		&a.ID,
		&a.Name,
		&a.Email,
		&a.Phone,
		&a.Status,
		&a.LogoURL,
		&a.BrandColor,
		&a.SecondaryColor,
		&a.LoginBackground,
		&a.FaviconURL,
		&a.LoginTitle,
		&a.LoginSubtitle,
		&a.BrandName,
		&a.CustomDomain,
		&a.Subdomain,
		&a.ContractTitle,
		&a.ContractBody,
		&a.ContractStatus,
		&a.ContractSignedBy,
		&a.ContractSignedEmail,
		&a.ContractSignature,
		&a.ContractSignedIP,
		&a.ContractSignedAt,
		&a.Notes,
		&a.MonthlyFee,
		&a.PlanEquivalent,
		&a.StartsAt,
		&a.ExpiresAt,
		&a.CreatedAt,
		&a.UpdatedAt,
	)

	return a, err
}

const agencySelectFields = `
	id, name, email, phone, status,
	logo_url, brand_color, secondary_color, login_background, favicon_url,
	login_title, login_subtitle, brand_name,
	custom_domain, subdomain,
	contract_title, contract_body, contract_status,
	contract_signed_by, contract_signed_email, contract_signature, contract_signed_ip,
	contract_signed_at, notes, monthly_fee, plan_equivalent, starts_at, expires_at,
	created_at, updated_at
`

func (s *AgencyService) List() ([]Agency, error) {
	rows, err := s.DB.Query(`
		SELECT ` + agencySelectFields + `
		FROM agencies
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Agency{}

	for rows.Next() {
		a, err := scanAgency(rows)
		if err != nil {
			return nil, err
		}

		out = append(out, a)
	}

	return out, rows.Err()
}

func (s *AgencyService) Get(id string) (Agency, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Agency{}, errors.New("agency id required")
	}

	row := s.DB.QueryRow(`
		SELECT `+agencySelectFields+`
		FROM agencies
		WHERE id=?
		LIMIT 1
	`, id)

	return scanAgency(row)
}

func (s *AgencyService) GetBySubdomain(slug string) (Agency, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return Agency{}, errors.New("subdomain required")
	}

	row := s.DB.QueryRow(`
		SELECT `+agencySelectFields+`
		FROM agencies
		WHERE lower(subdomain)=lower(?)
		LIMIT 1
	`, slug)

	return scanAgency(row)
}

func (s *AgencyService) Update(a Agency) error {
	a = normalizeAgency(a)

	if strings.TrimSpace(a.ID) == "" {
		return errors.New("agency id required")
	}

	a.UpdatedAt = time.Now()

	_, err := s.DB.Exec(`
		UPDATE agencies SET
			name=?,
			email=?,
			phone=?,
			status=?,
			logo_url=?,
			brand_color=?,
			secondary_color=?,
			login_background=?,
			favicon_url=?,
			login_title=?,
			login_subtitle=?,
			brand_name=?,
			custom_domain=?,
			subdomain=?,
			contract_title=?,
			contract_body=?,
			contract_status=?,
			notes=?,
			monthly_fee=?,
			plan_equivalent=?,
			starts_at=?,
			expires_at=?,
			updated_at=?
		WHERE id=?
	`,
		a.Name,
		a.Email,
		a.Phone,
		a.Status,
		a.LogoURL,
		a.BrandColor,
		a.SecondaryColor,
		a.LoginBackground,
		a.FaviconURL,
		a.LoginTitle,
		a.LoginSubtitle,
		a.BrandName,
		a.CustomDomain,
		a.Subdomain,
		a.ContractTitle,
		a.ContractBody,
		a.ContractStatus,
		a.Notes,
		a.MonthlyFee,
		a.PlanEquivalent,
		a.StartsAt,
		a.ExpiresAt,
		a.UpdatedAt,
		a.ID,
	)

	return err
}

func (s *AgencyService) Delete(id string) error {
	_, err := s.DB.Exec(`DELETE FROM agencies WHERE id=?`, strings.TrimSpace(id))
	return err
}

func (s *AgencyService) Activate(id string, months int) error {
	if months <= 0 {
		months = 1
	}

	now := time.Now()
	expires := now.AddDate(0, months, 0)

	_, err := s.DB.Exec(`
		UPDATE agencies
		SET status='active',
		    starts_at=?,
		    expires_at=?,
		    updated_at=?
		WHERE id=?
	`, now, expires, now, strings.TrimSpace(id))

	return err
}

func (s *AgencyService) Suspend(id string) error {
	now := time.Now()

	_, err := s.DB.Exec(`
		UPDATE agencies
		SET status='suspended',
		    updated_at=?
		WHERE id=?
	`, now, strings.TrimSpace(id))

	return err
}

func (s *AgencyService) IsActive(id string) bool {
	a, err := s.Get(id)
	if err != nil {
		return false
	}

	if a.Status != "active" {
		return false
	}

	if a.ExpiresAt == nil {
		return false
	}

	return a.ExpiresAt.After(time.Now())
}

func (s *AgencyService) SignContract(id, signedBy, signedEmail, signature, ip string) error {
	now := time.Now()

	if strings.TrimSpace(signature) == "" {
		return errors.New("signature required")
	}

	_, err := s.DB.Exec(`
		UPDATE agencies
		SET contract_status='signed',
		    contract_signed_by=?,
		    contract_signed_email=?,
		    contract_signature=?,
		    contract_signed_ip=?,
		    contract_signed_at=?,
		    updated_at=?
		WHERE id=?
	`,
		strings.TrimSpace(signedBy),
		strings.TrimSpace(signedEmail),
		strings.TrimSpace(signature),
		strings.TrimSpace(ip),
		now,
		now,
		strings.TrimSpace(id),
	)

	return err
}

func (s *AgencyService) SavePrices(agencyID string, prices []AgencyPlanPrice) error {
	agencyID = strings.TrimSpace(agencyID)
	if agencyID == "" {
		return errors.New("agency id required")
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM agency_plan_prices WHERE agency_id=?`, agencyID); err != nil {
		return err
	}

	now := time.Now()

	for _, p := range prices {
		p.PlanSlug = strings.TrimSpace(p.PlanSlug)
		p.BillingCycle = strings.TrimSpace(p.BillingCycle)

		if p.ID == "" {
			p.ID = uuid.NewString()
		}
		if p.BillingCycle == "" {
			p.BillingCycle = "monthly"
		}
		if p.PlanSlug == "" {
			continue
		}

		enabled := 0
		if p.Enabled {
			enabled = 1
		}

		if _, err := tx.Exec(`
			INSERT INTO agency_plan_prices (
				id, agency_id, plan_slug, normal_price, agency_price,
				billing_cycle, enabled, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			p.ID,
			agencyID,
			p.PlanSlug,
			p.NormalPrice,
			p.AgencyPrice,
			p.BillingCycle,
			enabled,
			now,
			now,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *AgencyService) Prices(agencyID string) ([]AgencyPlanPrice, error) {
	rows, err := s.DB.Query(`
		SELECT id, agency_id, plan_slug, normal_price, agency_price,
		       billing_cycle, enabled, created_at, updated_at
		FROM agency_plan_prices
		WHERE agency_id=?
		ORDER BY plan_slug ASC
	`, strings.TrimSpace(agencyID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AgencyPlanPrice{}

	for rows.Next() {
		var p AgencyPlanPrice
		var enabled int

		if err := rows.Scan(
			&p.ID,
			&p.AgencyID,
			&p.PlanSlug,
			&p.NormalPrice,
			&p.AgencyPrice,
			&p.BillingCycle,
			&enabled,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, err
		}

		p.Enabled = enabled == 1
		out = append(out, p)
	}

	return out, rows.Err()
}

func (s *AgencyService) AgencyPrice(agencyID, planSlug, billingCycle string) (float64, bool) {
	if !s.IsActive(agencyID) {
		return 0, false
	}

	var price float64
	var enabled int

	err := s.DB.QueryRow(`
		SELECT agency_price, enabled
		FROM agency_plan_prices
		WHERE agency_id=? AND plan_slug=? AND billing_cycle=?
		LIMIT 1
	`,
		strings.TrimSpace(agencyID),
		strings.TrimSpace(planSlug),
		strings.TrimSpace(billingCycle),
	).Scan(&price, &enabled)

	if err != nil || enabled != 1 || price <= 0 {
		return 0, false
	}

	return price, true
}

func DefaultAgencyContract(agencyName string) string {
	return `CONTRATO COMERCIAL DE AGENCIA WORKTIC AI

Entre Worktic S.A.S., identificada con NIT 900814197-0, en adelante WORKTIC, y la agencia ` + agencyName + `, en adelante LA AGENCIA, se celebra el presente acuerdo comercial para el uso, promoción y comercialización de la plataforma Worktic AI.

1. OBJETO
WORKTIC autoriza a LA AGENCIA a comercializar planes de Worktic AI bajo condiciones comerciales especiales, manteniendo la operación dentro del ecosistema oficial app.workticai.com.

2. CONTROL DE PLATAFORMA
LA AGENCIA reconoce que Worktic AI, su software, infraestructura, código fuente, automatizaciones, inteligencia artificial, diseños, flujos, módulos, base tecnológica y propiedad intelectual pertenecen exclusivamente a WORKTIC.

3. USO DE MARCA
LA AGENCIA podrá configurar elementos de branding propios dentro de la plataforma, como logo, color, nombre comercial y textos visibles, sin que esto implique transferencia de propiedad intelectual ni independencia tecnológica fuera de Worktic AI.

4. LICENCIA MENSUAL DE AGENCIA
LA AGENCIA deberá mantener activa su licencia mensual equivalente al plan Business o superior. Si la licencia vence, LA AGENCIA no podrá seguir ofreciendo precios especiales de agencia hasta renovar.

5. CLIENTES DE LA AGENCIA
Los clientes vinculados a LA AGENCIA podrán adquirir planes con precios preferenciales mientras la agencia esté activa. Si la agencia no renueva, los clientes existentes conservarán su licencia hasta su fecha de vencimiento, pero futuras renovaciones se realizarán con precios normales de Worktic AI.

6. PAGOS Y VALIDACIÓN
Los pagos de licencia de agencia podrán realizarse mediante links de pago generados por WORKTIC. La activación estará sujeta a validación y aprobación administrativa.

7. PROHIBICIONES
LA AGENCIA no podrá copiar, clonar, revender fuera de plataforma, sublicenciar sin autorización, reclamar autoría, competir usando la tecnología de Worktic AI, ni inducir a clientes a operar fuera del ecosistema oficial.

8. TERMINACIÓN
WORKTIC podrá suspender o terminar el convenio por incumplimiento, uso indebido, falta de pago, daño reputacional, intento de copia, fraude o violación de propiedad intelectual.

9. FIRMA ELECTRÓNICA
LA AGENCIA acepta que la firma electrónica, nombre, correo, IP, fecha y hora registrados en la plataforma constituyen aceptación válida del presente acuerdo.

Al firmar, LA AGENCIA declara haber leído, entendido y aceptado todas las condiciones del presente contrato.`
}