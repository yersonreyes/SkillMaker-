// Package service implements the authentication business logic:
// Google ID token validation, JWT issuance, refresh token rotation, and logout.
// All external dependencies are injected via interfaces — no *gorm.DB here.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/api/idtoken"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users"
)

// Sentinel errors used by handlers to build HTTP responses.
var (
	ErrInvalidGoogleToken  = errors.New("token de google invalido")
	ErrUnauthorizedDomain  = errors.New("cuenta no pertenece al dominio corporativo")
	ErrInvalidRefreshToken = errors.New("refresh token invalido o expirado")
	ErrRefreshTokenReused  = errors.New("refresh token reutilizado: sesiones revocadas")
)

// Config holds the runtime configuration injected at startup.
type Config struct {
	JWTSecret             string
	JWTExpiresIn          time.Duration
	RefreshTokenExpiresIn time.Duration
	GoogleClientID        string
	GoogleHostedDomain    string
}

// Service is the public interface of the auth module.
// Callers (handlers) only depend on this interface.
type Service interface {
	LoginWithGoogle(ctx context.Context, idTokenStr string) (dto.LoginResponse, error)
	Refresh(ctx context.Context, refreshTokenPlain string) (dto.LoginResponse, error)
	Logout(ctx context.Context, refreshTokenPlain string) error
}

type service struct {
	cfg   Config
	users users.Service
	repo  repository.Repository
}

// New constructs a Service with all required dependencies injected.
func New(cfg Config, u users.Service, r repository.Repository) Service {
	return &service{cfg: cfg, users: u, repo: r}
}

// LoginWithGoogle validates a Google ID token, enforces the hosted-domain
// constraint (RT-13), upserts the user, and issues a JWT + refresh token pair.
func (s *service) LoginWithGoogle(ctx context.Context, idTokenStr string) (dto.LoginResponse, error) {
	payload, err := idtoken.Validate(ctx, idTokenStr, s.cfg.GoogleClientID)
	if err != nil {
		return dto.LoginResponse{}, ErrInvalidGoogleToken
	}

	// RT-13: only allow accounts from the configured Google Workspace domain.
	hd, _ := payload.Claims["hd"].(string)
	if hd != s.cfg.GoogleHostedDomain {
		return dto.LoginResponse{}, ErrUnauthorizedDomain
	}

	sub, _ := payload.Claims["sub"].(string)
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)

	u, err := s.users.UpsertFromGoogle(ctx, users.GoogleProfile{
		GoogleSub: sub,
		Email:     email,
		Nombre:    name,
	})
	if err != nil {
		return dto.LoginResponse{}, err
	}

	accessToken, exp, err := s.issueJWT(*u)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	refreshToken, err := s.issueRefreshToken(ctx, u.ID, "")
	if err != nil {
		return dto.LoginResponse{}, err
	}

	return dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    exp,
		User: dto.UserDTO{
			ID:     u.ID,
			Email:  u.Email,
			Nombre: u.Nombre,
			Roles:  u.Roles,
		},
	}, nil
}

// Refresh rotates a refresh token: validates the presented token, detects
// OWASP replay attacks (a used token re-presented = revoke all), marks the
// old token as used+revoked, and issues a fresh JWT + refresh token pair.
func (s *service) Refresh(ctx context.Context, refreshTokenPlain string) (dto.LoginResponse, error) {
	sum := sha256.Sum256([]byte(refreshTokenPlain))
	hash := hex.EncodeToString(sum[:])

	rt, err := s.repo.FindByHash(ctx, hash)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	if rt == nil {
		return dto.LoginResponse{}, ErrInvalidRefreshToken
	}
	if rt.RevokedAt != nil || time.Now().UTC().After(rt.ExpiresAt) {
		return dto.LoginResponse{}, ErrInvalidRefreshToken
	}

	// ── OWASP reuse detection ─────────────────────────────────────────────────
	// A token that was already used (UsedAt != nil) being presented again is a
	// strong signal that it was stolen. The safe response is to revoke ALL active
	// refresh tokens for this user, forcing them to re-authenticate via Google.
	if rt.UsedAt != nil {
		_ = s.repo.RevokeAllForUser(ctx, rt.UserID)
		return dto.LoginResponse{}, ErrRefreshTokenReused
	}

	// ── Rotation ──────────────────────────────────────────────────────────────
	now := time.Now().UTC()
	if err := s.repo.MarkUsed(ctx, rt.ID, now); err != nil {
		return dto.LoginResponse{}, err
	}
	if err := s.repo.Revoke(ctx, rt.ID, now); err != nil {
		return dto.LoginResponse{}, err
	}

	u, err := s.users.GetByID(ctx, rt.UserID)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	accessToken, exp, err := s.issueJWT(*u)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	newRefresh, err := s.issueRefreshToken(ctx, u.ID, rt.ID)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	return dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		ExpiresAt:    exp,
		User:         dto.UserDTO{ID: u.ID, Email: u.Email, Nombre: u.Nombre, Roles: u.Roles},
	}, nil
}

// Logout revokes the given refresh token. The operation is idempotent: calling
// it with an already-revoked or non-existent token returns nil (HTTP 204).
func (s *service) Logout(ctx context.Context, refreshTokenPlain string) error {
	if refreshTokenPlain == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(refreshTokenPlain))
	hash := hex.EncodeToString(sum[:])

	rt, err := s.repo.FindByHash(ctx, hash)
	if err != nil || rt == nil {
		return nil // idempotent
	}
	return s.repo.Revoke(ctx, rt.ID, time.Now().UTC())
}

// ── Private helpers ───────────────────────────────────────────────────────────

// issueJWT creates a signed HS256 JWT for the given user with the claims
// required by the middleware (sub, email, nombre, roles, exp, iat).
func (s *service) issueJWT(u users.UserSummary) (string, time.Time, error) {
	exp := time.Now().UTC().Add(s.cfg.JWTExpiresIn)
	claims := jwt.MapClaims{
		"sub":    u.ID,
		"email":  u.Email,
		"nombre": u.Nombre,
		"roles":  u.Roles,
		"exp":    exp.Unix(),
		"iat":    time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	return signed, exp, err
}

// issueRefreshToken generates a cryptographically random 32-byte token,
// encodes it as base64url (opaque to the client), persists only its
// SHA-256 hash, and returns the plaintext. The plaintext is never stored —
// only the client holds it.
func (s *service) issueRefreshToken(ctx context.Context, userID string, parentID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	tokenPlain := base64.RawURLEncoding.EncodeToString(raw)

	sum := sha256.Sum256([]byte(tokenPlain))
	tokenHash := hex.EncodeToString(sum[:])

	var parent *string
	if parentID != "" {
		parent = &parentID
	}

	rt := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: tokenHash,
		ParentID:  parent,
		ExpiresAt: time.Now().UTC().Add(s.cfg.RefreshTokenExpiresIn),
	}
	if err := s.repo.Insert(ctx, rt); err != nil {
		return "", err
	}
	return tokenPlain, nil
}
