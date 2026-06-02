package service

import (
	"context"
	"errors"
	"time"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// GoogleProfile carries the Google ID token claims needed to create or update
// a user record. It is the input to UpsertFromGoogle.
type GoogleProfile struct {
	GoogleSub string
	Email     string
	Nombre    string
}

// UserSummary is the read model returned by the service to its callers.
// It contains only the data that cross-module consumers (e.g. auth) need.
type UserSummary struct {
	ID     string
	Email  string
	Nombre string
	Roles  []string
}

// UserDetailModel is the richer read model used by the users management API.
// It is returned by GetDetail, List, PatchRoles, and SetActive.
type UserDetailModel struct {
	ID        string
	Email     string
	Nombre    string
	Activo    bool
	Roles     []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SupervisionModel is the read model for a supervision relation.
type SupervisionModel struct {
	ID           string
	SupervisorID string
	EmpleadoID   string
	CreadoEn     time.Time
}

// ListFilters mirrors repository.ListFilters at the service layer so callers
// only import the service package.
type ListFilters struct {
	Q      string // ILIKE substring on nombre/email
	Role   string // exact role name; "" = any
	Active *bool  // nil = any
}

// Service is the public interface of the users module.
// Other modules depend on this interface — never on the concrete struct.
type Service interface {
	// UpsertFromGoogle creates or updates a user from a Google profile and
	// returns a populated UserSummary including role names.
	UpsertFromGoogle(ctx context.Context, profile GoogleProfile) (*UserSummary, error)

	// GetByID fetches a user by primary key and returns a UserSummary.
	GetByID(ctx context.Context, id string) (*UserSummary, error)

	// List returns a paginated, filtered page of users with their detail.
	List(ctx context.Context, f ListFilters, p pagination.Params) (pagination.Page[UserDetailModel], error)

	// GetDetail returns the full UserDetailModel for a user.
	// Used by GET /users/:id and GET /users/me.
	GetDetail(ctx context.Context, id string) (*UserDetailModel, error)

	// PatchRoles applies a role delta (add, remove) to a user.
	// Validates role names, detects add∩remove conflicts, enforces the
	// last-admin invariant before removing "administrador".
	PatchRoles(ctx context.Context, id string, add, remove []string) (*UserDetailModel, error)

	// SetActive sets the activo flag on a user.
	// Enforces the last-admin invariant when active=false.
	SetActive(ctx context.Context, id string, active bool) (*UserDetailModel, error)

	// CreateSupervision creates a supervisor→employee relation.
	CreateSupervision(ctx context.Context, supervisorID, empleadoID string) (*SupervisionModel, error)

	// ListSupervisions returns all supervision relations.
	ListSupervisions(ctx context.Context) ([]SupervisionModel, error)

	// DeleteSupervision removes a supervision relation by its primary key.
	DeleteSupervision(ctx context.Context, id string) error
}

// ── concrete implementation ────────────────────────────────────────────────────

type serviceImpl struct {
	repo repository.Repository
}

// New creates a Service backed by the given Repository.
func New(repo repository.Repository) Service {
	return &serviceImpl{repo: repo}
}

// ── existing methods (unchanged) ──────────────────────────────────────────────

func (s *serviceImpl) UpsertFromGoogle(ctx context.Context, profile GoogleProfile) (*UserSummary, error) {
	user, err := s.repo.UpsertByGoogleSub(ctx, repository.GoogleProfile{
		GoogleSub: profile.GoogleSub,
		Email:     profile.Email,
		Nombre:    profile.Nombre,
	})
	if err != nil {
		return nil, err
	}

	roles, err := s.repo.LoadRoleNames(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &UserSummary{
		ID:     user.ID,
		Email:  user.Email,
		Nombre: user.Nombre,
		Roles:  roles,
	}, nil
}

func (s *serviceImpl) GetByID(ctx context.Context, id string) (*UserSummary, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapUserNotFound(err)
	}

	roles, err := s.repo.LoadRoleNames(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &UserSummary{
		ID:     user.ID,
		Email:  user.Email,
		Nombre: user.Nombre,
		Roles:  roles,
	}, nil
}

// ── new C1.1 methods ──────────────────────────────────────────────────────────

func (s *serviceImpl) GetDetail(ctx context.Context, id string) (*UserDetailModel, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapUserNotFound(err)
	}

	roles, err := s.repo.LoadRoleNames(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return toDetailModel(user, roles), nil
}

func (s *serviceImpl) List(ctx context.Context, f ListFilters, p pagination.Params) (pagination.Page[UserDetailModel], error) {
	repoPage, err := s.repo.List(ctx, toRepoFilters(f), p)
	if err != nil {
		return pagination.Page[UserDetailModel]{}, err
	}

	items := make([]UserDetailModel, 0, len(repoPage.Items))
	for i := range repoPage.Items {
		u := &repoPage.Items[i]
		roles := roleNames(u.Roles)
		items = append(items, *toDetailModel(u, roles))
	}

	return pagination.Page[UserDetailModel]{
		Items:      items,
		Page:       repoPage.Page,
		Size:       repoPage.Size,
		Total:      repoPage.Total,
		TotalPages: repoPage.TotalPages,
	}, nil
}

func (s *serviceImpl) PatchRoles(ctx context.Context, id string, add, remove []string) (*UserDetailModel, error) {
	// 1. Validate all names.
	for _, name := range add {
		if !isValidRole(name) {
			return nil, ErrInvalidRole
		}
	}

	for _, name := range remove {
		if !isValidRole(name) {
			return nil, ErrInvalidRole
		}
	}

	// 2. Detect add∩remove conflict.
	addSet := toSet(add)
	for _, name := range remove {
		if _, exists := addSet[name]; exists {
			return nil, ErrAddRemoveConflict
		}
	}

	// 3. Last-admin guard when removing "administrador".
	for _, name := range remove {
		if name == "administrador" {
			if err := s.ensureNotLastAdmin(ctx, id); err != nil {
				return nil, err
			}

			break
		}
	}

	// 4. Apply delta (both calls are idempotent in the repo layer).
	if err := s.repo.AddRoles(ctx, id, add); err != nil {
		return nil, err
	}

	if err := s.repo.RemoveRoles(ctx, id, remove); err != nil {
		return nil, err
	}

	// 5. Return refreshed detail.
	return s.GetDetail(ctx, id)
}

func (s *serviceImpl) SetActive(ctx context.Context, id string, active bool) (*UserDetailModel, error) {
	// Last-admin guard when deactivating.
	if !active {
		if err := s.ensureNotLastAdmin(ctx, id); err != nil {
			return nil, err
		}
	}

	if err := s.repo.SetActive(ctx, id, active); err != nil {
		return nil, wrapUserNotFound(err)
	}

	return s.GetDetail(ctx, id)
}

func (s *serviceImpl) CreateSupervision(ctx context.Context, supervisorID, empleadoID string) (*SupervisionModel, error) {
	if supervisorID == empleadoID {
		return nil, ErrSelfSupervision
	}

	// Pre-check both users exist.
	if ok, err := s.repo.ExistsUser(ctx, supervisorID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrUserNotFound
	}

	if ok, err := s.repo.ExistsUser(ctx, empleadoID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrUserNotFound
	}

	sv, err := s.repo.CreateSupervision(ctx, supervisorID, empleadoID)
	if err != nil {
		if errors.Is(err, repository.ErrSupervisionExists) {
			return nil, ErrSupervisionExists
		}

		return nil, err
	}

	return &SupervisionModel{
		ID:           sv.ID,
		SupervisorID: sv.SupervisorID,
		EmpleadoID:   sv.EmpleadoID,
		CreadoEn:     sv.CreadoEn,
	}, nil
}

func (s *serviceImpl) ListSupervisions(ctx context.Context) ([]SupervisionModel, error) {
	svs, err := s.repo.ListSupervisions(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]SupervisionModel, 0, len(svs))
	for _, sv := range svs {
		out = append(out, SupervisionModel{
			ID:           sv.ID,
			SupervisorID: sv.SupervisorID,
			EmpleadoID:   sv.EmpleadoID,
			CreadoEn:     sv.CreadoEn,
		})
	}

	return out, nil
}

func (s *serviceImpl) DeleteSupervision(ctx context.Context, id string) error {
	err := s.repo.DeleteSupervision(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrSupervisionNotFound) {
			return ErrUserNotFound // supervision not found is a 404
		}

		return err
	}

	return nil
}

// ── private helpers ────────────────────────────────────────────────────────────

// ensureNotLastAdmin returns ErrLastAdmin if the target user is the last active
// administrator. It is called before removing the "administrador" role and before
// deactivating a user — covering both mutation paths (spec U5, design D3).
func (s *serviceImpl) ensureNotLastAdmin(ctx context.Context, targetID string) error {
	detail, err := s.GetDetail(ctx, targetID)
	if err != nil {
		return err
	}

	// Target is not an active admin → no risk.
	if !contains(detail.Roles, "administrador") || !detail.Activo {
		return nil
	}

	n, err := s.repo.CountActiveAdmins(ctx)
	if err != nil {
		return err
	}

	if n <= 1 {
		return ErrLastAdmin
	}

	return nil
}

// toDetailModel converts a GORM domain.User + preloaded role names into a UserDetailModel.
func toDetailModel(u *domain.User, roles []string) *UserDetailModel {
	return &UserDetailModel{
		ID:        u.ID,
		Email:     u.Email,
		Nombre:    u.Nombre,
		Activo:    u.Activo,
		Roles:     roles,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// toRepoFilters converts service-level ListFilters to repository-level ListFilters.
func toRepoFilters(f ListFilters) repository.ListFilters {
	return repository.ListFilters{
		Q:      f.Q,
		Role:   f.Role,
		Active: f.Active,
	}
}

// roleNames extracts the Nombre field from a slice of domain.Role.
func roleNames(roles []domain.Role) []string {
	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, r.Nombre)
	}

	return names
}

// toSet converts a string slice to a set (map[string]struct{}).
func toSet(ss []string) map[string]struct{} {
	m := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		m[s] = struct{}{}
	}

	return m
}

// contains reports whether s contains the target string.
func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}

	return false
}

// wrapUserNotFound converts repository.ErrUserNotFound to service.ErrUserNotFound.
func wrapUserNotFound(err error) error {
	if errors.Is(err, repository.ErrUserNotFound) {
		return ErrUserNotFound
	}

	return err
}
