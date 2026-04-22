package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type SocialScheduler struct {
	DB      *sql.DB
	Social  *SocialService
	stopCh  chan struct{}
}

func NewSocialScheduler(db *sql.DB, social *SocialService) *SocialScheduler {
	return &SocialScheduler{
		DB:     db,
		Social: social,
		stopCh: make(chan struct{}),
	}
}

func (s *SocialScheduler) Start() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.runPending()
			case <-s.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *SocialScheduler) Stop() {
	close(s.stopCh)
}

func (s *SocialScheduler) CreateJob(clientID, campaignID, postID, jobType string, runAt time.Time, recurringMinutes int, daysOfWeek string) error {
	now := time.Now()
	_, err := s.DB.Exec(`
		INSERT INTO social_jobs (
			id, client_id, campaign_id, post_id, job_type, run_at, recurring_minutes, days_of_week,
			status, last_error, last_run_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending', '', NULL, ?, ?)
	`,
		uuid.NewString(), clientID, campaignID, postID, jobType, runAt, recurringMinutes, daysOfWeek, now, now,
	)
	return err
}

func (s *SocialScheduler) runPending() {
	rows, err := s.DB.Query(`
		SELECT id, post_id, job_type, run_at, recurring_minutes, days_of_week
		FROM social_jobs
		WHERE status='pending' AND run_at <= ?
		ORDER BY run_at ASC
	`, time.Now())
	if err != nil {
		return
	}
	defer rows.Close()

	type job struct {
		ID               string
		PostID           string
		JobType          string
		RunAt            time.Time
		RecurringMinutes int
		DaysOfWeek       string
	}

	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.ID, &j.PostID, &j.JobType, &j.RunAt, &j.RecurringMinutes, &j.DaysOfWeek); err == nil {
			jobs = append(jobs, j)
		}
	}

	for _, j := range jobs {
		now := time.Now()
		_, _ = s.DB.Exec(`UPDATE social_jobs SET status='running', last_run_at=?, updated_at=? WHERE id=?`, now, now, j.ID)

		err := s.Social.PublishNow(context.Background(), j.PostID)
		if err != nil {
			_, _ = s.DB.Exec(`UPDATE social_jobs SET status='error', last_error=?, updated_at=? WHERE id=?`, err.Error(), time.Now(), j.ID)
			continue
		}

		if j.JobType == "recurring_publish" && j.RecurringMinutes > 0 {
			nextRun := time.Now().Add(time.Duration(j.RecurringMinutes) * time.Minute)
			_, _ = s.DB.Exec(`UPDATE social_jobs SET status='pending', run_at=?, updated_at=? WHERE id=?`, nextRun, time.Now(), j.ID)
		} else {
			_, _ = s.DB.Exec(`UPDATE social_jobs SET status='done', updated_at=? WHERE id=?`, time.Now(), j.ID)
		}
	}
}