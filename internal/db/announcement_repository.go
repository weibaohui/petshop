package db

import (
	"database/sql"
	"time"

	"petshop/internal/models"
)

// AnnouncementRepository handles announcement database operations
type AnnouncementRepository struct {
	db *sql.DB
}

// NewAnnouncementRepository creates a new AnnouncementRepository
func NewAnnouncementRepository() *AnnouncementRepository {
	return &AnnouncementRepository{db: GetDB()}
}

// NewAnnouncementRepositoryWithDB creates a new AnnouncementRepository with a specific database instance
func NewAnnouncementRepositoryWithDB(db *sql.DB) *AnnouncementRepository {
	return &AnnouncementRepository{db: db}
}

// GetAll returns all announcements
func (r *AnnouncementRepository) GetAll() ([]*models.Announcement, error) {
	rows, err := r.db.Query(`
		SELECT id, title, content, status, created_at, updated_at
		FROM announcements ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanAnnouncements(rows)
}

// GetByID returns an announcement by ID
func (r *AnnouncementRepository) GetByID(id int64) (*models.Announcement, error) {
	a := &models.Announcement{}

	err := r.db.QueryRow(`
		SELECT id, title, content, status, created_at, updated_at
		FROM announcements WHERE id = ?`, id).Scan(
		&a.ID, &a.Title, &a.Content, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// GetActive returns all active announcements
func (r *AnnouncementRepository) GetActive() ([]*models.Announcement, error) {
	rows, err := r.db.Query(`
		SELECT id, title, content, status, created_at, updated_at
		FROM announcements WHERE status = 'active' ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanAnnouncements(rows)
}

// Create creates a new announcement
func (r *AnnouncementRepository) Create(a *models.Announcement) error {
	now := time.Now()

	result, err := r.db.Exec(`
		INSERT INTO announcements (title, content, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		a.Title, a.Content, a.Status, now, now)
	if err != nil {
		return err
	}

	a.ID, _ = result.LastInsertId()
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

// Update updates an announcement
func (r *AnnouncementRepository) Update(a *models.Announcement) error {
	now := time.Now()

	_, err := r.db.Exec(`
		UPDATE announcements SET title = ?, content = ?, status = ?, updated_at = ?
		WHERE id = ?`,
		a.Title, a.Content, a.Status, now, a.ID)
	if err != nil {
		return err
	}

	a.UpdatedAt = now
	return nil
}

// Delete deletes an announcement
func (r *AnnouncementRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM announcements WHERE id = ?`, id)
	return err
}

// scanAnnouncements scans announcement rows
func (r *AnnouncementRepository) scanAnnouncements(rows *sql.Rows) ([]*models.Announcement, error) {
	var announcements []*models.Announcement
	for rows.Next() {
		a := &models.Announcement{}
		if err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		announcements = append(announcements, a)
	}
	return announcements, rows.Err()
}
