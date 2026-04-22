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

func (a *AuthService) Login(email, password string) (models.User, string, error) {
	var id, clientID, name, hash, role, status string
	var created time.Time
	err := a.DB.QueryRow(`SELECT id, client_id, name, password_hash, role, status, created_at FROM users WHERE email=?`, email).Scan(&id, &clientID, &name, &hash, &role, &status, &created)
	if err != nil { return models.User{}, "", err }
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil { return models.User{}, "", errors.New("invalid credentials") }
	user := models.User{ID:id, ClientID:clientID, Name:name, Email:email, Role:role, Status:status, CreatedAt:created}
	token, err := a.createSession(id); return user, token, err
}

func (a *AuthService) createSession(userID string) (string, error) {
	token := uuid.NewString(); now := time.Now(); exp := now.Add(time.Duration(a.SessionDays) * 24 * time.Hour)
	_, err := a.DB.Exec(`INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`, token, userID, exp, now)
	return token, err
}

func (a *AuthService) GetUser(id string) (models.User, error) {
	var u models.User
	err := a.DB.QueryRow(`SELECT id, client_id, name, email, role, status, created_at FROM users WHERE id=?`, id).Scan(&u.ID, &u.ClientID, &u.Name, &u.Email, &u.Role, &u.Status, &u.CreatedAt)
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
	query := `SELECT id, client_id, name, email, role, status, created_at FROM users`
	args := []any{}
	if clientID != "" { query += ` WHERE client_id=?`; args = append(args, clientID) }
	query += ` ORDER BY created_at DESC`
	rows, err := a.DB.Query(query, args...); if err != nil { return nil, err }
	defer rows.Close()
	out := []models.User{}
	for rows.Next() { var u models.User; if err := rows.Scan(&u.ID,&u.ClientID,&u.Name,&u.Email,&u.Role,&u.Status,&u.CreatedAt); err != nil { return nil, err }; out = append(out,u) }
	return out,nil
}

func (a *AuthService) DeleteUser(id string) error { _, err := a.DB.Exec(`DELETE FROM users WHERE id=?`, id); return err }
