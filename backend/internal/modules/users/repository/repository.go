package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/domain"
)

// GoogleProfile carries the claims from a Google ID token used to upsert a user.
type GoogleProfile struct {
	GoogleSub string
	Email     string
	Nombre    string
}

// Repository defines the data-access contract for the users module.
type Repository interface {
	// UpsertByGoogleSub inserts or updates a user record keyed on google_sub.
	// After the upsert it ensures the "alumno" role is assigned, in the same
	// transaction.
	UpsertByGoogleSub(ctx context.Context, profile GoogleProfile) (*domain.User, error)

	// GetByID fetches a user by primary key.
	GetByID(ctx context.Context, id string) (*domain.User, error)

	// LoadRoleNames returns the names of all roles assigned to the given user.
	LoadRoleNames(ctx context.Context, userID string) ([]string, error)
}

type gormRepository struct {
	db *gorm.DB
}

// New returns a Repository backed by GORM.
func New(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) UpsertByGoogleSub(ctx context.Context, profile GoogleProfile) (*domain.User, error) {
	user := domain.User{
		ID:        uuid.New().String(),
		GoogleSub: profile.GoogleSub,
		Email:     profile.Email,
		Nombre:    profile.Nombre,
		Activo:    true,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Atomic upsert: INSERT ... ON CONFLICT (google_sub) DO UPDATE
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "google_sub"}},
			DoUpdates: clause.AssignmentColumns([]string{"email", "nombre", "updated_at"}),
		}).Create(&user)
		if result.Error != nil {
			return result.Error
		}

		// If the row already existed the ID in &user will be the newly generated
		// UUID, not the existing one. Re-fetch by google_sub to get the real ID.
		var existing domain.User
		if err := tx.WithContext(ctx).
			Where("google_sub = ?", profile.GoogleSub).
			First(&existing).Error; err != nil {
			return err
		}
		user = existing

		// Ensure the user has the "alumno" role (idempotent).
		if err := tx.Exec(
			`INSERT INTO user_role (user_id, role_id, asignado_en)
			 SELECT ?, r.id, NOW() FROM role r WHERE r.nombre = 'alumno'
			 ON CONFLICT DO NOTHING`,
			user.ID,
		).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *gormRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *gormRepository) LoadRoleNames(ctx context.Context, userID string) ([]string, error) {
	var names []string
	err := r.db.WithContext(ctx).
		Table("role r").
		Select("r.nombre").
		Joins("JOIN user_role ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Pluck("r.nombre", &names).Error
	if err != nil {
		return nil, err
	}
	return names, nil
}

// ErrUserNotFound is returned when a user lookup by ID finds no row.
var ErrUserNotFound = errors.New("user not found")
