package db

import (
	"database/sql"

	"petshop/internal/models"
)

// CarouselRepository handles carousel database operations
type CarouselRepository struct {
	db *sql.DB
}

// NewCarouselRepository creates a new CarouselRepository
func NewCarouselRepository() *CarouselRepository {
	return &CarouselRepository{db: GetDB()}
}

// NewCarouselRepositoryWithDB creates a new CarouselRepository with a specific database instance
func NewCarouselRepositoryWithDB(db *sql.DB) *CarouselRepository {
	return &CarouselRepository{db: db}
}

// GetAll returns all carousels
func (r *CarouselRepository) GetAll() ([]*models.Carousel, error) {
	rows, err := r.db.Query(`
		SELECT id, image_url, link_url, sort_order, title, status
		FROM carousels ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCarousels(rows)
}

// GetByID returns a carousel by ID
func (r *CarouselRepository) GetByID(id int64) (*models.Carousel, error) {
	c := &models.Carousel{}

	err := r.db.QueryRow(`
		SELECT id, image_url, link_url, sort_order, title, status
		FROM carousels WHERE id = ?`, id).Scan(
		&c.ID, &c.ImageURL, &c.LinkURL, &c.SortOrder, &c.Title, &c.Status)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// GetActive returns all active carousels
func (r *CarouselRepository) GetActive() ([]*models.Carousel, error) {
	rows, err := r.db.Query(`
		SELECT id, image_url, link_url, sort_order, title, status
		FROM carousels WHERE status = 'active' ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCarousels(rows)
}

// Create creates a new carousel
func (r *CarouselRepository) Create(c *models.Carousel) error {
	result, err := r.db.Exec(`
		INSERT INTO carousels (image_url, link_url, sort_order, title, status)
		VALUES (?, ?, ?, ?, ?)`,
		c.ImageURL, c.LinkURL, c.SortOrder, c.Title, c.Status)
	if err != nil {
		return err
	}

	c.ID, _ = result.LastInsertId()
	return nil
}

// Update updates a carousel
func (r *CarouselRepository) Update(c *models.Carousel) error {
	_, err := r.db.Exec(`
		UPDATE carousels SET image_url = ?, link_url = ?, sort_order = ?, title = ?, status = ?
		WHERE id = ?`,
		c.ImageURL, c.LinkURL, c.SortOrder, c.Title, c.Status, c.ID)
	return err
}

// Delete deletes a carousel
func (r *CarouselRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM carousels WHERE id = ?`, id)
	return err
}

// scanCarousels scans carousel rows
func (r *CarouselRepository) scanCarousels(rows *sql.Rows) ([]*models.Carousel, error) {
	var carousels []*models.Carousel
	for rows.Next() {
		c := &models.Carousel{}
		if err := rows.Scan(
			&c.ID, &c.ImageURL, &c.LinkURL, &c.SortOrder, &c.Title, &c.Status); err != nil {
			return nil, err
		}
		carousels = append(carousels, c)
	}
	return carousels, rows.Err()
}
