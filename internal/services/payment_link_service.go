package services

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PaymentLink struct {
	ID              string     `json:"id"`
	ClientID        string     `json:"client_id"`
	CreatedBy       string     `json:"created_by"`
	Concept         string     `json:"concept"`
	Description     string     `json:"description"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	PaymentMethod   string     `json:"payment_method"`
	WalletAddress   string     `json:"wallet_address"`
	CustomerName    string     `json:"customer_name"`
	CustomerEmail   string     `json:"customer_email"`
	CustomerPhone   string     `json:"customer_phone"`
	TxHash          string     `json:"tx_hash"`
	Status          string     `json:"status"`
	AgencyID     	string 	   `json:"agency_id"`
	PaymentScope 	string 	   `json:"payment_scope"`
	ExpiresAt       *time.Time `json:"expires_at"`
	PaidAt          *time.Time `json:"paid_at"`
	ApprovedAt      *time.Time `json:"approved_at"`
	ApprovedBy      string     `json:"approved_by"`
	RejectionReason string     `json:"rejection_reason"`
	PublicURL       string     `json:"public_url"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type PaymentLinkService struct {
	DB *sql.DB
}

func NewPaymentLinkService(db *sql.DB) *PaymentLinkService {
	return &PaymentLinkService{DB: db}
}

func (s *PaymentLinkService) walletAddress() string {
	var wallet string
	_ = s.DB.QueryRow(`
		SELECT usdt_bep20_wallet
		FROM plan_config
		LIMIT 1
	`).Scan(&wallet)

	return strings.TrimSpace(wallet)
}

func (s *PaymentLinkService) Create(x PaymentLink, publicBaseURL string) (PaymentLink, error) {
	now := time.Now()

	if strings.TrimSpace(x.Concept) == "" {
		return PaymentLink{}, errors.New("concept required")
	}
	if x.Amount <= 0 {
		return PaymentLink{}, errors.New("amount must be greater than 0")
	}

	if x.ID == "" {
		x.ID = uuid.NewString()
	}
	if x.Currency == "" {
		x.Currency = "USDT"
	}
	if x.PaymentMethod == "" {
		x.PaymentMethod = "usdt_bep20"
	}
	if x.Status == "" {
		x.Status = "created"
	}

	x.WalletAddress = s.walletAddress()
	if x.WalletAddress == "" {
		return PaymentLink{}, errors.New("USDT BEP20 wallet is not configured")
	}

	if x.PaymentScope == "" {
	x.PaymentScope = "client"
	}

	x.CreatedAt = now
	x.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO payment_links (
			id, client_id, created_by, concept, description, amount, currency,
			payment_method, wallet_address, customer_name, customer_email,
			customer_phone, tx_hash, status, agency_id, payment_scope, expires_at, paid_at, approved_at,
			approved_by, rejection_reason, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		x.ID,
		x.ClientID,
		x.CreatedBy,
		x.Concept,
		x.Description,
		x.Amount,
		x.Currency,
		x.PaymentMethod,
		x.WalletAddress,
		x.CustomerName,
		x.CustomerEmail,
		x.CustomerPhone,
		x.TxHash,
		x.Status,
		x.AgencyID,
		x.PaymentScope,
		x.ExpiresAt,
		x.PaidAt,
		x.ApprovedAt,
		x.ApprovedBy,
		x.RejectionReason,
		x.CreatedAt,
		x.UpdatedAt,
	)

	if err != nil {
		return PaymentLink{}, err
	}

	x.PublicURL = strings.TrimRight(publicBaseURL, "/") + "/pay/" + x.ID
	return x, nil
}

func (s *PaymentLinkService) List(clientID string) ([]PaymentLink, error) {
	rows, err := s.DB.Query(`
		SELECT id, client_id, created_by, concept, description, amount, currency,
		       payment_method, wallet_address, customer_name, customer_email,
		       customer_phone, tx_hash, status, expires_at, paid_at, approved_at,
		       approved_by, rejection_reason, created_at, updated_at
		FROM payment_links
		WHERE (? = '' OR client_id = ?)
		ORDER BY created_at DESC
	`, clientID, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []PaymentLink{}
	for rows.Next() {
		var x PaymentLink
		if err := rows.Scan(
			&x.ID,
			&x.ClientID,
			&x.CreatedBy,
			&x.Concept,
			&x.Description,
			&x.Amount,
			&x.Currency,
			&x.PaymentMethod,
			&x.WalletAddress,
			&x.CustomerName,
			&x.CustomerEmail,
			&x.CustomerPhone,
			&x.TxHash,
			&x.Status,
			&x.ExpiresAt,
			&x.PaidAt,
			&x.ApprovedAt,
			&x.ApprovedBy,
			&x.RejectionReason,
			&x.CreatedAt,
			&x.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, x)
	}

	return out, nil
}

func (s *PaymentLinkService) Get(id string) (PaymentLink, error) {
	var x PaymentLink

	err := s.DB.QueryRow(`
		SELECT id, client_id, created_by, concept, description, amount, currency,
		       payment_method, wallet_address, customer_name, customer_email,
		       customer_phone, tx_hash, status, expires_at, paid_at, approved_at,
		       approved_by, rejection_reason, created_at, updated_at
		FROM payment_links
		WHERE id=?
		LIMIT 1
	`, id).Scan(
		&x.ID,
		&x.ClientID,
		&x.CreatedBy,
		&x.Concept,
		&x.Description,
		&x.Amount,
		&x.Currency,
		&x.PaymentMethod,
		&x.WalletAddress,
		&x.CustomerName,
		&x.CustomerEmail,
		&x.CustomerPhone,
		&x.TxHash,
		&x.Status,
		&x.ExpiresAt,
		&x.PaidAt,
		&x.ApprovedAt,
		&x.ApprovedBy,
		&x.RejectionReason,
		&x.CreatedAt,
		&x.UpdatedAt,
	)

	return x, err
}

func (s *PaymentLinkService) SubmitTx(id, customerName, customerEmail, customerPhone, txHash string) error {
	txHash = strings.TrimSpace(txHash)
	if txHash == "" {
		return errors.New("tx_hash required")
	}

	now := time.Now()

	_, err := s.DB.Exec(`
		UPDATE payment_links
		SET customer_name=?,
		    customer_email=?,
		    customer_phone=?,
		    tx_hash=?,
		    status='paid_submitted',
		    paid_at=?,
		    updated_at=?
		WHERE id=?
		AND status IN ('created','paid_submitted','rejected')
	`,
		strings.TrimSpace(customerName),
		strings.TrimSpace(customerEmail),
		strings.TrimSpace(customerPhone),
		txHash,
		now,
		now,
		id,
	)

	return err
}

func (s *PaymentLinkService) Approve(id, adminID string) error {
	now := time.Now()

	_, err := s.DB.Exec(`
		UPDATE payment_links
		SET status='approved',
		    approved_at=?,
		    approved_by=?,
		    rejection_reason='',
		    updated_at=?
		WHERE id=?
	`, now, adminID, now, id)

	return err
}

func (s *PaymentLinkService) Reject(id, reason string) error {
	now := time.Now()

	_, err := s.DB.Exec(`
		UPDATE payment_links
		SET status='rejected',
		    rejection_reason=?,
		    updated_at=?
		WHERE id=?
	`, strings.TrimSpace(reason), now, id)

	return err
}