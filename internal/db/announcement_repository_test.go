package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"petshop/internal/models"
)

func setupAnnouncementTestDB(t *testing.T) *AnnouncementRepository {
	ResetForTesting()
	err := InitDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(ResetForTesting)
	return NewAnnouncementRepository()
}

func TestNewAnnouncementRepository(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = NewAnnouncementRepository()
	})
}

func TestNewAnnouncementRepositoryWithDB(t *testing.T) {
	ResetForTesting()
	err := InitDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(ResetForTesting)

	repo := NewAnnouncementRepositoryWithDB(GetDB())
	assert.NotNil(t, repo)

	// Verify the repository works by creating an announcement
	announcement := &models.Announcement{
		Title:   "Test Title",
		Content: "Test Content",
		Status:  "active",
	}
	err = repo.Create(announcement)
	assert.NoError(t, err)
	assert.NotZero(t, announcement.ID)
}

func TestAnnouncementRepository_Create(t *testing.T) {
	r := setupAnnouncementTestDB(t)

	tests := []struct {
		name    string
		input   *models.Announcement
		wantErr bool
		check   func(a *models.Announcement)
	}{
		{
			name: "create active announcement",
			input: &models.Announcement{
				Title:   "Welcome",
				Content: "Welcome to our pet shop!",
				Status:  "active",
			},
			wantErr: false,
			check: func(a *models.Announcement) {
				assert.NotZero(t, a.ID)
				assert.Equal(t, "Welcome", a.Title)
				assert.Equal(t, "Welcome to our pet shop!", a.Content)
				assert.Equal(t, "active", a.Status)
				assert.False(t, a.CreatedAt.IsZero())
				assert.False(t, a.UpdatedAt.IsZero())
			},
		},
		{
			name: "create inactive announcement",
			input: &models.Announcement{
				Title:   "Old News",
				Content: "This is an old announcement",
				Status:  "inactive",
			},
			wantErr: false,
			check: func(a *models.Announcement) {
				assert.NotZero(t, a.ID)
				assert.Equal(t, "inactive", a.Status)
			},
		},
		{
			name: "create announcement with empty content",
			input: &models.Announcement{
				Title:   "Empty Content Test",
				Content: "",
				Status:  "active",
			},
			wantErr: false,
			check: func(a *models.Announcement) {
				assert.Equal(t, "", a.Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Create(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(tt.input)
				}
			}
		})
	}
}

func TestAnnouncementRepository_GetAll(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(r *AnnouncementRepository)
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty table",
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "multiple announcements",
			setup: func(r *AnnouncementRepository) {
				_ = r.Create(&models.Announcement{Title: "First", Content: "Content 1", Status: "active"})
				_ = r.Create(&models.Announcement{Title: "Second", Content: "Content 2", Status: "inactive"})
				_ = r.Create(&models.Announcement{Title: "Third", Content: "Content 3", Status: "active"})
			},
			wantLen: 3,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupAnnouncementTestDB(t)
			if tt.setup != nil {
				tt.setup(r)
			}

			announcements, err := r.GetAll()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, announcements, tt.wantLen)
				// Verify descending order by ID
				for i := 1; i < len(announcements); i++ {
					assert.Greater(t, announcements[i-1].ID, announcements[i].ID)
				}
			}
		})
	}
}

func TestAnnouncementRepository_GetAll_Error(t *testing.T) {
	ResetForTesting()
	err := InitDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(ResetForTesting)

	repo := NewAnnouncementRepository()
	Close() // Close db to force error

	_, err = repo.GetAll()
	assert.Error(t, err)
}

func TestAnnouncementRepository_GetByID(t *testing.T) {
	r := setupAnnouncementTestDB(t)

	// Create test announcement
	announcement := &models.Announcement{
		Title:   "Test Announcement",
		Content: "Test Content",
		Status:  "active",
	}
	err := r.Create(announcement)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      int64
		wantErr bool
		check   func(a *models.Announcement)
	}{
		{
			name:    "existing announcement",
			id:      announcement.ID,
			wantErr: false,
			check: func(a *models.Announcement) {
				assert.Equal(t, announcement.ID, a.ID)
				assert.Equal(t, "Test Announcement", a.Title)
				assert.Equal(t, "Test Content", a.Content)
				assert.Equal(t, "active", a.Status)
			},
		},
		{
			name:    "non-existent id",
			id:      9999,
			wantErr: true,
		},
		{
			name:    "non-existent id zero",
			id:      0,
			wantErr: true,
		},
		{
			name:    "non-existent negative id",
			id:      -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.GetByID(tt.id)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(got)
				}
			}
		})
	}
}

func TestAnnouncementRepository_GetActive(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(r *AnnouncementRepository)
		wantLen int
		wantErr bool
	}{
		{
			name:    "no active announcements",
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "only inactive announcements",
			setup: func(r *AnnouncementRepository) {
				_ = r.Create(&models.Announcement{Title: "Inactive 1", Content: "Content", Status: "inactive"})
				_ = r.Create(&models.Announcement{Title: "Inactive 2", Content: "Content", Status: "inactive"})
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "only active announcements",
			setup: func(r *AnnouncementRepository) {
				_ = r.Create(&models.Announcement{Title: "Active 1", Content: "Content", Status: "active"})
				_ = r.Create(&models.Announcement{Title: "Active 2", Content: "Content", Status: "active"})
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "mixed status announcements",
			setup: func(r *AnnouncementRepository) {
				_ = r.Create(&models.Announcement{Title: "Active", Content: "Content", Status: "active"})
				_ = r.Create(&models.Announcement{Title: "Inactive", Content: "Content", Status: "inactive"})
				_ = r.Create(&models.Announcement{Title: "Active 2", Content: "Content", Status: "active"})
			},
			wantLen: 2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupAnnouncementTestDB(t)
			if tt.setup != nil {
				tt.setup(r)
			}

			announcements, err := r.GetActive()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, announcements, tt.wantLen)
				for _, a := range announcements {
					assert.Equal(t, "active", a.Status)
				}
			}
		})
	}
}

func TestAnnouncementRepository_GetActive_Error(t *testing.T) {
	ResetForTesting()
	err := InitDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(ResetForTesting)

	repo := NewAnnouncementRepository()
	Close() // Close db to force error

	_, err = repo.GetActive()
	assert.Error(t, err)
}

func TestAnnouncementRepository_Update(t *testing.T) {
	r := setupAnnouncementTestDB(t)

	// Create test announcement
	announcement := &models.Announcement{
		Title:   "Original Title",
		Content: "Original Content",
		Status:  "active",
	}
	err := r.Create(announcement)
	require.NoError(t, err)
	originalUpdatedAt := announcement.UpdatedAt

	tests := []struct {
		name    string
		input   *models.Announcement
		wantErr bool
		check   func()
	}{
		{
			name: "update existing announcement",
			input: &models.Announcement{
				ID:      announcement.ID,
				Title:   "Updated Title",
				Content: "Updated Content",
				Status:  "inactive",
			},
			wantErr: false,
			check: func() {
				updated, err := r.GetByID(announcement.ID)
				require.NoError(t, err)
				assert.Equal(t, "Updated Title", updated.Title)
				assert.Equal(t, "Updated Content", updated.Content)
				assert.Equal(t, "inactive", updated.Status)
				assert.True(t, updated.UpdatedAt.After(originalUpdatedAt) || updated.UpdatedAt.Equal(originalUpdatedAt))
			},
		},
		{
			name: "update non-existent announcement",
			input: &models.Announcement{
				ID:      9999,
				Title:   "Should Not Exist",
				Content: "Content",
				Status:  "active",
			},
			wantErr: false, // Exec succeeds with 0 rows affected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Update(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Update should update the UpdatedAt field
				if tt.input.ID == announcement.ID {
					assert.False(t, tt.input.UpdatedAt.IsZero())
				}
			}
			if tt.check != nil {
				tt.check()
			}
		})
	}
}

func TestAnnouncementRepository_Delete(t *testing.T) {
	r := setupAnnouncementTestDB(t)

	// Create test announcement
	announcement := &models.Announcement{
		Title:   "To Be Deleted",
		Content: "Delete Me",
		Status:  "active",
	}
	err := r.Create(announcement)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      int64
		wantErr bool
		check   func()
	}{
		{
			name:    "delete existing announcement",
			id:      announcement.ID,
			wantErr: false,
			check: func() {
				_, err := r.GetByID(announcement.ID)
				assert.Error(t, err)
			},
		},
		{
			name:    "delete non-existent announcement",
			id:      9999,
			wantErr: false,
		},
		{
			name:    "delete already deleted announcement",
			id:      announcement.ID,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Delete(tt.id)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.check != nil {
				tt.check()
			}
		})
	}
}

func TestAnnouncementRepository_Integration(t *testing.T) {
	r := setupAnnouncementTestDB(t)

	// Create multiple announcements
	active1 := &models.Announcement{Title: "Active 1", Content: "Content 1", Status: "active"}
	active2 := &models.Announcement{Title: "Active 2", Content: "Content 2", Status: "active"}
	inactive := &models.Announcement{Title: "Inactive", Content: "Content 3", Status: "inactive"}

	require.NoError(t, r.Create(active1))
	require.NoError(t, r.Create(active2))
	require.NoError(t, r.Create(inactive))

	// Test GetAll returns all in descending order
	all, err := r.GetAll()
	require.NoError(t, err)
	assert.Len(t, all, 3)
	assert.Equal(t, inactive.ID, all[0].ID) // Last created should be first

	// Test GetActive returns only active
	active, err := r.GetActive()
	require.NoError(t, err)
	assert.Len(t, active, 2)

	// Update one announcement
	active1.Title = "Updated Active 1"
	require.NoError(t, r.Update(active1))

	// Verify update
	updated, err := r.GetByID(active1.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Active 1", updated.Title)

	// Delete inactive announcement
	require.NoError(t, r.Delete(inactive.ID))

	// Verify deletion
	_, err = r.GetByID(inactive.ID)
	assert.Error(t, err)

	// Verify counts after deletion
	all, err = r.GetAll()
	require.NoError(t, err)
	assert.Len(t, all, 2)

	active, err = r.GetActive()
	require.NoError(t, err)
	assert.Len(t, active, 2)
}

func TestAnnouncementRepository_TimeFields(t *testing.T) {
	r := setupAnnouncementTestDB(t)

	beforeCreate := time.Now()

	announcement := &models.Announcement{
		Title:   "Time Test",
		Content: "Testing time fields",
		Status:  "active",
	}

	err := r.Create(announcement)
	require.NoError(t, err)

	// Verify created_at and updated_at are set
	assert.False(t, announcement.CreatedAt.IsZero())
	assert.False(t, announcement.UpdatedAt.IsZero())
	assert.True(t, announcement.CreatedAt.After(beforeCreate) || announcement.CreatedAt.Equal(beforeCreate))
	assert.True(t, announcement.UpdatedAt.After(beforeCreate) || announcement.UpdatedAt.Equal(beforeCreate))

	// Verify updated_at equals created_at after create
	createdAt := announcement.CreatedAt
	updatedAtAfterCreate := announcement.UpdatedAt
	assert.True(t, updatedAtAfterCreate.After(createdAt) || updatedAtAfterCreate.Equal(createdAt))

	// Update the announcement
	announcement.Title = "Updated Time Test"
	err = r.Update(announcement)
	require.NoError(t, err)

	// Verify updated_at is after created_at after update
	assert.True(t, announcement.UpdatedAt.After(createdAt))
}
