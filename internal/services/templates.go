package services

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type TemplateService struct {
	DB *sql.DB
}

func NewTemplateService(db *sql.DB) *TemplateService {
	return &TemplateService{DB: db}
}

func normalizeTemplateValue(v string) string {
	return strings.TrimSpace(strings.ToLower(v))
}

func (t *TemplateService) List(clientID string) ([]models.Template, error) {
	rows, err := t.DB.Query(`
		SELECT
			id, client_id, name, category, business_type, stage,
			prompt_snippet, message_template, is_default, created_at, updated_at
		FROM templates
		WHERE client_id='' OR client_id=?
		ORDER BY is_default DESC, updated_at DESC
	`, strings.TrimSpace(clientID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Template{}
	for rows.Next() {
		var it models.Template
		if err := rows.Scan(
			&it.ID,
			&it.ClientID,
			&it.Name,
			&it.Category,
			&it.BusinessType,
			&it.Stage,
			&it.PromptSnippet,
			&it.MessageTemplate,
			&it.IsDefault,
			&it.CreatedAt,
			&it.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}

	return out, rows.Err()
}

func (t *TemplateService) Create(template models.Template) (models.Template, error) {
	now := time.Now()
	template.ID = uuid.NewString()
	template.CreatedAt = now
	template.UpdatedAt = now

	template.ClientID = strings.TrimSpace(template.ClientID)
	template.Name = strings.TrimSpace(template.Name)
	template.Category = normalizeTemplateValue(template.Category)
	template.BusinessType = normalizeTemplateValue(template.BusinessType)
	template.Stage = normalizeTemplateValue(template.Stage)
	template.PromptSnippet = strings.TrimSpace(template.PromptSnippet)
	template.MessageTemplate = strings.TrimSpace(template.MessageTemplate)

	if template.Category == "" {
		template.Category = "sales"
	}
	if template.BusinessType == "" {
		template.BusinessType = "general"
	}
	if template.Stage == "" {
		template.Stage = "new"
	}

	_, err := t.DB.Exec(`
		INSERT INTO templates (
			id, client_id, name, category, business_type, stage,
			prompt_snippet, message_template, is_default, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		template.ID,
		template.ClientID,
		template.Name,
		template.Category,
		template.BusinessType,
		template.Stage,
		template.PromptSnippet,
		template.MessageTemplate,
		template.IsDefault,
		now,
		now,
	)
	return template, err
}

func (t *TemplateService) Update(template models.Template) error {
	template.Name = strings.TrimSpace(template.Name)
	template.Category = normalizeTemplateValue(template.Category)
	template.BusinessType = normalizeTemplateValue(template.BusinessType)
	template.Stage = normalizeTemplateValue(template.Stage)
	template.PromptSnippet = strings.TrimSpace(template.PromptSnippet)
	template.MessageTemplate = strings.TrimSpace(template.MessageTemplate)

	if template.Category == "" {
		template.Category = "sales"
	}
	if template.BusinessType == "" {
		template.BusinessType = "general"
	}
	if template.Stage == "" {
		template.Stage = "new"
	}

	_, err := t.DB.Exec(`
		UPDATE templates
		SET
			name=?,
			category=?,
			business_type=?,
			stage=?,
			prompt_snippet=?,
			message_template=?,
			updated_at=?
		WHERE id=?
	`,
		template.Name,
		template.Category,
		template.BusinessType,
		template.Stage,
		template.PromptSnippet,
		template.MessageTemplate,
		time.Now(),
		template.ID,
	)
	return err
}

func (t *TemplateService) Delete(id string) error {
	_, err := t.DB.Exec(`DELETE FROM templates WHERE id=?`, strings.TrimSpace(id))
	return err
}

func (t *TemplateService) GetByID(id string) (models.Template, error) {
	var tpl models.Template

	err := t.DB.QueryRow(`
		SELECT
			id, client_id, name, category, business_type, stage,
			prompt_snippet, message_template, is_default, created_at, updated_at
		FROM templates
		WHERE id=?
	`, strings.TrimSpace(id)).Scan(
		&tpl.ID,
		&tpl.ClientID,
		&tpl.Name,
		&tpl.Category,
		&tpl.BusinessType,
		&tpl.Stage,
		&tpl.PromptSnippet,
		&tpl.MessageTemplate,
		&tpl.IsDefault,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	)

	return tpl, err
}

func (t *TemplateService) FindBestReplyTemplate(clientID, businessType, category, stage string) (models.Template, error) {
	clientID = strings.TrimSpace(clientID)
	businessType = normalizeTemplateValue(businessType)
	category = normalizeTemplateValue(category)
	stage = normalizeTemplateValue(stage)

	if businessType == "" {
		businessType = "general"
	}
	if category == "" {
		category = "sales"
	}
	if stage == "" {
		stage = "new"
	}

	var tpl models.Template

	err := t.DB.QueryRow(`
		SELECT
			id, client_id, name, category, business_type, stage,
			prompt_snippet, message_template, is_default, created_at, updated_at
		FROM templates
		WHERE
			(client_id='' OR client_id=?)
			AND (business_type=? OR business_type='general' OR business_type='')
			AND (category=? OR category='sales' OR category='')
			AND (stage=? OR stage='new' OR stage='')
		ORDER BY
			CASE WHEN client_id=? THEN 0 ELSE 1 END,
			CASE WHEN business_type=? THEN 0 WHEN business_type='general' OR business_type='' THEN 1 ELSE 2 END,
			CASE WHEN category=? THEN 0 WHEN category='sales' OR category='' THEN 1 ELSE 2 END,
			CASE WHEN stage=? THEN 0 WHEN stage='new' OR stage='' THEN 1 ELSE 2 END,
			is_default DESC,
			updated_at DESC
		LIMIT 1
	`,
		clientID,
		businessType,
		category,
		stage,
		clientID,
		businessType,
		category,
		stage,
	).Scan(
		&tpl.ID,
		&tpl.ClientID,
		&tpl.Name,
		&tpl.Category,
		&tpl.BusinessType,
		&tpl.Stage,
		&tpl.PromptSnippet,
		&tpl.MessageTemplate,
		&tpl.IsDefault,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	)

	return tpl, err
}