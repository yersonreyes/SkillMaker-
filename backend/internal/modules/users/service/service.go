package service

import (
	"context"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/repository"
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

// Service is the public interface of the users module.
// Other modules depend on this interface — never on the concrete struct.
type Service interface {
	// UpsertFromGoogle creates or updates a user from a Google profile and
	// returns a populated UserSummary including role names.
	UpsertFromGoogle(ctx context.Context, profile GoogleProfile) (*UserSummary, error)

	// GetByID fetches a user by primary key and returns a UserSummary.
	GetByID(ctx context.Context, id string) (*UserSummary, error)
}

type serviceImpl struct {
	repo repository.Repository
}

// New creates a Service backed by the given Repository.
func New(repo repository.Repository) Service {
	return &serviceImpl{repo: repo}
}

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
