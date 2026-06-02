package services

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type AuthService struct { DB *sql.DB; SessionDays int }

func NewAuthService(db *sql.DB, sessionDays int) *AuthService { return &AuthService{DB: db, SessionDays: sessionDays} }

func (a *AuthService) HasUsers() (bool, error) {
	var c int
	err := a.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&c)
	return c > 0, err
}

func (a *AuthService) BootstrapAdmin(name, email, password string) (models.User, string, error) {
	has, err := a.HasUsers(); if err != nil { return models.User{}, "", err }
	if has { return models.User{}, "", errors.New("bootstrap already completed") }
	return a.CreateUser("", name, email, password, "admin")
}

func (a *AuthService) CreateUser(clientID, name, email, password, role string) (models.User, string, error) {
	if role == "" { role = "client_admin" }
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil { return models.User{}, "", err }
	id := uuid.NewString(); now := time.Now()
	_, err = a.DB.Exec(`INSERT INTO users (id, client_id, name, email, password_hash, role, status, created_at) VALUES (?, ?, ?, ?, ?, ?, 'active', ?)`, id, clientID, name, email, string(hash), role, now)
	if err != nil { return models.User{}, "", err }
	user, err := a.GetUser(id); if err != nil { return models.User{}, "", err }
	token, err := a.createSession(id); return user, token, err
}

func (a *AuthService) CreateUserWithAgency(clientID, agencyID, name, email, password, role string) (models.User, string, error) {
	if role == "" {
		role = "client_admin"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, "", err
	}

	id := uuid.NewString()
	now := time.Now()

	_, err = a.DB.Exec(`
		INSERT INTO users (
			id, client_id, agency_id, name, email, password_hash, role, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'active', ?)
	`,
		id,
		clientID,
		agencyID,
		name,
		email,
		string(hash),
		role,
		now,
	)

	if err != nil {
		return models.User{}, "", err
	}

	user, err := a.GetUser(id)
	if err != nil {
		return models.User{}, "", err
	}

	token, err := a.createSession(id)
	return user, token, err
}

func (a *AuthService) CreatePendingUserWithAgency(clientID, agencyID, name, email, password, role string) (models.User, string, error) {
	if role == "" {
		role = "client_user"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, "", err
	}

	id := uuid.NewString()
	now := time.Now()

	_, err = a.DB.Exec(`
		INSERT INTO users (
			id, client_id, agency_id, name, email, password_hash, role, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'pending_license', ?)
	`,
		id,
		clientID,
		agencyID,
		name,
		email,
		string(hash),
		role,
		now,
	)

	if err != nil {
		return models.User{}, "", err
	}

	user, err := a.GetUser(id)
	if err != nil {
		return models.User{}, "", err
	}

	return user, "", nil
}

func (a *AuthService) Login(email, password string) (models.User, string, error) {
	var id, clientID, agencyID, name, hash, role, plan, status string
	var created time.Time
	err := a.DB.QueryRow(`
		SELECT
			u.id,
			u.client_id,
			COALESCE(u.agency_id, c.agency_id, ''),
			u.name,
			u.password_hash,
			u.role,
			COALESCE(c.plan, 'free'),
			u.status,
			u.created_at
		FROM users u
		LEFT JOIN clients c ON c.id = u.client_id
		WHERE u.email=?
	`, email).Scan(
		&id,
		&clientID,
		&agencyID,
		&name,
		&hash,
		&role,
		&plan,
		&status,
		&created,
	)
	if err != nil { return models.User{}, "", err }
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil { return models.User{}, "", errors.New("invalid credentials") }
	if agencyID != "" && role == "agency_admin" && status != "active" {
	return models.User{}, "", errors.New("agencia pendiente de contrato y pago")
	}

	if agencyID != "" && role == "client_user" && status != "active" {
		return models.User{}, "", errors.New("usuario pendiente de activación de licencia")
	}
	user := models.User{
	ID: id,
	ClientID: clientID,
	AgencyID: agencyID,
	Name: name,
	Email: email,
	Role: role,
	Plan: plan,
	Status: status,
	CreatedAt: created,
	}
	token, err := a.createSession(id); return user, token, err
}

func (a *AuthService) createSession(userID string) (string, error) {
	token := uuid.NewString(); now := time.Now(); exp := now.Add(time.Duration(a.SessionDays) * 24 * time.Hour)
	_, err := a.DB.Exec(`INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`, token, userID, exp, now)
	return token, err
}

func (a *AuthService) GetUser(id string) (models.User, error) {
	var u models.User
	err := a.DB.QueryRow(`
		SELECT
			u.id,
			u.client_id,
			COALESCE(u.agency_id, c.agency_id, ''),
			u.name,
			u.email,
			u.role,
			COALESCE(c.plan, 'free'),
			u.status,
			u.created_at
		FROM users u
		LEFT JOIN clients c ON c.id = u.client_id
		WHERE u.id=?
	`, id).Scan(
		&u.ID,
		&u.ClientID,
		&u.AgencyID,
		&u.Name,
		&u.Email,
		&u.Role,
		&u.Plan,
		&u.Status,
		&u.CreatedAt,
	)
	return u, err
}

func (a *AuthService) GetUserByToken(token string) (models.User, error) {
	var userID string; var exp time.Time
	err := a.DB.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE token=?`, token).Scan(&userID, &exp)
	if err != nil { return models.User{}, err }
	if time.Now().After(exp) { return models.User{}, errors.New("session expired") }
	return a.GetUser(userID)
}

func (a *AuthService) ListUsers(clientID string) ([]models.User, error) {
	query := `
	SELECT
		u.id,
		u.client_id,
		COALESCE(u.agency_id, c.agency_id, ''),
		u.name,
		u.email,
		u.role,
		COALESCE(c.plan, 'free'),
		u.status,
		u.created_at
	FROM users u
	LEFT JOIN clients c ON c.id = u.client_id`
	args := []any{}
	if clientID != "" { query += ` WHERE client_id=?`; args = append(args, clientID) }
	query += ` ORDER BY u.created_at DESC`
	rows, err := a.DB.Query(query, args...); if err != nil { return nil, err }
	defer rows.Close()
	out := []models.User{}
	for rows.Next() { var u models.User; if err := rows.Scan(
	&u.ID,
	&u.ClientID,
	&u.AgencyID,
	&u.Name,
	&u.Email,
	&u.Role,
	&u.Plan,
	&u.Status,
	&u.CreatedAt,
); err != nil { return nil, err }; out = append(out,u) }
	return out,nil
}

func (a *AuthService) UpdateUser(id, clientID, name, email, role, status, password string) (models.User, error) {
	if id == "" {
		return models.User{}, errors.New("user id required")
	}

	if status == "" {
		status = "active"
	}

	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return models.User{}, err
		}

		_, err = a.DB.Exec(`
			UPDATE users
			SET client_id=?, name=?, email=?, role=?, status=?, password_hash=?
			WHERE id=?
		`, clientID, name, email, role, status, string(hash), id)

		if err != nil {
			return models.User{}, err
		}
	} else {
		_, err := a.DB.Exec(`
			UPDATE users
			SET client_id=?, name=?, email=?, role=?, status=?
			WHERE id=?
		`, clientID, name, email, role, status, id)

		if err != nil {
			return models.User{}, err
		}
	}

	return a.GetUser(id)
}

func (a *AuthService) DeleteUser(id string) error { _, err := a.DB.Exec(`DELETE FROM users WHERE id=?`, id); return err }

func (a *AuthService) ResetAgencyAdminPassword(agencyID, email, newPassword string) (models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}

	_, err = a.DB.Exec(`
		UPDATE users
		SET password_hash=?,
		    status='active'
		WHERE agency_id=? AND email=? AND role='agency_admin'
	`,
		string(hash),
		agencyID,
		email,
	)

	if err != nil {
		return models.User{}, err
	}

	var userID string
	err = a.DB.QueryRow(`
		SELECT id
		FROM users
		WHERE agency_id=? AND email=? AND role='agency_admin'
		LIMIT 1
	`, agencyID, email).Scan(&userID)

	if err != nil {
		return models.User{}, err
	}

	return a.GetUser(userID)
}