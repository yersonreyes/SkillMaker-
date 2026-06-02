package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// GoogleProfile carries the claims from a Google ID token used to upsert a user.
type GoogleProfile struct {
	GoogleSub string
	Email     string
	Nombre    string
}

// ListFilters carries optional filter criteria for listing users.
// Zero-value means unfiltered for that field.
type ListFilters struct {
	Q      string // ILIKE substring match on nombre OR email; "" = any
	Role   string // exact role name (one of 4 fixed); "" = any
	Active *bool  // nil = any; non-nil filters user.activo
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

	// List returns a paginated, filtered page of users.
	List(ctx context.Context, f ListFilters, p pagination.Params) (pagination.Page[domain.User], error)

	// SetActive sets the activo flag on the given user.
	SetActive(ctx context.Context, id string, active bool) error

	// AddRoles assigns role names to a user (idempotent — ON CONFLICT DO NOTHING).
	AddRoles(ctx context.Context, userID string, roleNames []string) error

	// RemoveRoles removes role names from a user (idempotent — absent roles are no-op).
	RemoveRoles(ctx context.Context, userID string, roleNames []string) error

	// CountActiveAdmins returns the number of users with role "administrador" and activo=true.
	CountActiveAdmins(ctx context.Context) (int64, error)

	// ExistsUser reports whether a user with the given ID exists.
	ExistsUser(ctx context.Context, id string) (bool, error)

	// CreateSupervision creates a supervisor→employee relation.
	// Returns ErrSupervisionExists if the employee already has a supervisor.
	CreateSupervision(ctx context.Context, supervisorID, empleadoID string) (*domain.Supervision, error)

	// ListSupervisions returns all supervision relations.
	ListSupervisions(ctx context.Context) ([]domain.Supervision, error)

	// DeleteSupervision removes a supervision relation by its primary key.
	// Returns ErrSupervisionNotFound if the relation does not exist.
	DeleteSupervision(ctx context.Context, id string) error

	// ListEmployeesBySupervisor returns all users supervised by the given supervisor.
	ListEmployeesBySupervisor(ctx context.Context, supervisorID string) ([]domain.User, error)
}

// ── Sentinels ─────────────────────────────────────────────────────────────────

// ErrUserNotFound is returned when a user lookup by ID finds no row.
var ErrUserNotFound = errors.New("user not found")

// ErrSupervisionExists is returned when an employee already has a supervisor
// (UNIQUE constraint violation on empleado_id).
var ErrSupervisionExists = errors.New("employee already has a supervisor")

// ErrSupervisionNotFound is returned when a supervision relation does not exist.
var ErrSupervisionNotFound = errors.New("supervision relation not found")

// ── gormRepository ────────────────────────────────────────────────────────────

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

func (r *gormRepository) List(ctx context.Context, f ListFilters, p pagination.Params) (pagination.Page[domain.User], error) {
	base := r.db.WithContext(ctx).Model(&domain.User{})

	if f.Q != "" {
		like := "%" + f.Q + "%"
		base = base.Where(`"user".nombre ILIKE ? OR "user".email ILIKE ?`, like, like)
	}

	if f.Active != nil {
		base = base.Where(`"user".activo = ?`, *f.Active)
	}

	if f.Role != "" {
		// EXISTS subquery avoids row duplication that a JOIN would cause.
		base = base.Where(`EXISTS (
			SELECT 1 FROM user_role ur JOIN role rl ON rl.id = ur.role_id
			WHERE ur.user_id = "user".id AND rl.nombre = ?)`, f.Role)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return pagination.Page[domain.User]{}, err
	}

	var users []domain.User
	err := base.
		Preload("Roles").
		Order(`"user".created_at DESC`).
		Offset(p.Offset()).Limit(p.Limit()).
		Find(&users).Error
	if err != nil {
		return pagination.Page[domain.User]{}, err
	}

	return pagination.NewPage(users, total, p), nil
}

func (r *gormRepository) SetActive(ctx context.Context, id string, active bool) error {
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", id).
		Update("activo", active)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *gormRepository) AddRoles(ctx context.Context, userID string, roleNames []string) error {
	if len(roleNames) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Exec(
		`INSERT INTO user_role (user_id, role_id, asignado_en)
		 SELECT ?, r.id, NOW() FROM role r WHERE r.nombre = ANY(?)
		 ON CONFLICT DO NOTHING`,
		userID, pq.Array(roleNames),
	).Error
}

func (r *gormRepository) RemoveRoles(ctx context.Context, userID string, roleNames []string) error {
	if len(roleNames) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Exec(
		`DELETE FROM user_role
		 WHERE user_id = ? AND role_id IN (SELECT id FROM role WHERE nombre = ANY(?))`,
		userID, pq.Array(roleNames),
	).Error
}

func (r *gormRepository) CountActiveAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table(`"user" u`).
		Select("COUNT(DISTINCT u.id)").
		Joins("JOIN user_role ur ON ur.user_id = u.id").
		Joins("JOIN role rl ON rl.id = ur.role_id").
		Where("rl.nombre = ? AND u.activo = ?", "administrador", true).
		Scan(&count).Error
	return count, err
}

func (r *gormRepository) ExistsUser(ctx context.Context, id string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", id).
		Count(&count).Error
	return count > 0, err
}

func (r *gormRepository) CreateSupervision(ctx context.Context, supervisorID, empleadoID string) (*domain.Supervision, error) {
	sv := domain.Supervision{
		ID:           uuid.New().String(),
		SupervisorID: supervisorID,
		EmpleadoID:   empleadoID,
	}

	if err := r.db.WithContext(ctx).Create(&sv).Error; err != nil {
		if isPgUniqueViolation(err) {
			return nil, ErrSupervisionExists
		}

		return nil, err
	}

	return &sv, nil
}

func (r *gormRepository) ListSupervisions(ctx context.Context) ([]domain.Supervision, error) {
	var supervisions []domain.Supervision
	err := r.db.WithContext(ctx).Order("creado_en DESC").Find(&supervisions).Error
	return supervisions, err
}

func (r *gormRepository) DeleteSupervision(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Supervision{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrSupervisionNotFound
	}

	return nil
}

func (r *gormRepository) ListEmployeesBySupervisor(ctx context.Context, supervisorID string) ([]domain.User, error) {
	var supervisions []domain.Supervision
	if err := r.db.WithContext(ctx).
		Where("supervisor_id = ?", supervisorID).
		Find(&supervisions).Error; err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(supervisions))
	for _, sv := range supervisions {
		ids = append(ids, sv.EmpleadoID)
	}

	if len(ids) == 0 {
		return []domain.User{}, nil
	}

	var users []domain.User
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, err
}

// isPgUniqueViolation reports whether err is a Postgres UNIQUE violation (23505).
// We avoid importing pgconn directly by inspecting the error string — keeps the
// import graph clean. A compile-time type assertion would require adding pgconn
// to go.mod as a direct dependency.
func isPgUniqueViolation(err error) bool {
	type pgcoder interface{ SQLState() string }
	var pg pgcoder
	if errors.As(err, &pg) {
		return pg.SQLState() == "23505"
	}

	return false
}
