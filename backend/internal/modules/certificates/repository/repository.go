// Package repository contains the data-access layer for the certificates module.
// Mirrors the courses gormRepository pattern exactly: interface + sentinel + gormRepository + New(db).
package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/domain"
	certService "github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/service"
)

// ErrCertificateNotFound is returned when a certificate lookup finds no matching row.
var ErrCertificateNotFound = errors.New("certificate not found")

// BadgeWithGrant is re-exported from service to avoid an import cycle
// (service declares the interface; repo returns the type).
type BadgeWithGrant = certService.BadgeWithGrant

// RankingRow is re-exported from service for the same reason.
type RankingRow = certService.RankingRow

// Repository defines the data-access contract for the certificates module.
type Repository interface {
	GetByUserCourse(ctx context.Context, userID, courseID string) (*domain.Certificate, error)
	Create(ctx context.Context, cert *domain.Certificate) error
	GetByID(ctx context.Context, certID string) (*domain.Certificate, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Certificate, error)
	CountByUser(ctx context.Context, userID string) (int64, error)
	ListBadgesByUser(ctx context.Context, userID string) ([]BadgeWithGrant, error)
	ListBadgesUpToThreshold(ctx context.Context, count int64) ([]domain.Badge, error)
	AwardBadge(ctx context.Context, userID, badgeID string) error
	Ranking(ctx context.Context, n int) ([]RankingRow, error)
}

type gormRepository struct {
	db *gorm.DB
}

// New constructs a GORM-backed Repository.
func New(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// GetByUserCourse finds a certificate by (user_id, course_id).
// Returns ErrCertificateNotFound if no row matches.
func (r *gormRepository) GetByUserCourse(ctx context.Context, userID, courseID string) (*domain.Certificate, error) {
	var cert domain.Certificate
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		First(&cert).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCertificateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// Create persists a new certificate. Uses ON CONFLICT (user_id, course_id) DO NOTHING
// as a double-guard against concurrent IssueOnPass calls.
func (r *gormRepository) Create(ctx context.Context, cert *domain.Certificate) error {
	result := r.db.WithContext(ctx).
		Exec(
			`INSERT INTO certificate (id, user_id, course_id, codigo, storage_key, emitido_en)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT (user_id, course_id) DO NOTHING`,
			cert.ID, cert.UserID, cert.CourseID, cert.Codigo, cert.StorageKey, cert.EmitidoEn,
		)
	return result.Error
}

// GetByID fetches a certificate by primary key.
// Returns ErrCertificateNotFound when no row matches.
func (r *gormRepository) GetByID(ctx context.Context, certID string) (*domain.Certificate, error) {
	var cert domain.Certificate
	err := r.db.WithContext(ctx).First(&cert, "id = ?", certID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCertificateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// ListByUser returns all certificates for userID ordered by emitido_en DESC.
func (r *gormRepository) ListByUser(ctx context.Context, userID string) ([]domain.Certificate, error) {
	var certs []domain.Certificate
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("emitido_en DESC").
		Find(&certs).Error
	return certs, err
}

// CountByUser returns the total number of certificates for userID.
func (r *gormRepository) CountByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Certificate{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// badgeScanRow is the intermediate scan target for ListBadgesByUser.
type badgeScanRow struct {
	ID          string    `gorm:"column:id"`
	Nombre      string    `gorm:"column:nombre"`
	Descripcion string    `gorm:"column:descripcion"`
	OtorgadoEn  time.Time `gorm:"column:otorgado_en"`
}

// ListBadgesByUser returns the earned badges for userID, joined with badge metadata,
// ordered by umbral ASC.
func (r *gormRepository) ListBadgesByUser(ctx context.Context, userID string) ([]BadgeWithGrant, error) {
	var rows []badgeScanRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT b.id, b.nombre, b.descripcion, ub.otorgado_en
		 FROM user_badge ub
		 JOIN badge b ON b.id = ub.badge_id
		 WHERE ub.user_id = ?
		 ORDER BY b.umbral ASC`,
		userID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]BadgeWithGrant, 0, len(rows))
	for _, rw := range rows {
		out = append(out, BadgeWithGrant{
			ID:          rw.ID,
			Nombre:      rw.Nombre,
			Descripcion: rw.Descripcion,
			OtorgadoEn:  rw.OtorgadoEn,
		})
	}
	return out, nil
}

// ListBadgesUpToThreshold returns all badges with umbral <= count, ordered by umbral ASC.
func (r *gormRepository) ListBadgesUpToThreshold(ctx context.Context, count int64) ([]domain.Badge, error) {
	var badges []domain.Badge
	err := r.db.WithContext(ctx).
		Where("umbral <= ?", count).
		Order("umbral ASC").
		Find(&badges).Error
	return badges, err
}

// AwardBadge inserts a user_badge row. Idempotent: ON CONFLICT (user_id, badge_id) DO NOTHING.
func (r *gormRepository) AwardBadge(ctx context.Context, userID, badgeID string) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO user_badge (user_id, badge_id)
		 VALUES (?, ?)
		 ON CONFLICT (user_id, badge_id) DO NOTHING`,
		userID, badgeID,
	).Error
}

// Ranking returns the top-n users by certificate count (0-cert users excluded).
// Ordered DESC by total, then ASC by nombre (ties), then ASC by user_id (stable tiebreaker).
func (r *gormRepository) Ranking(ctx context.Context, n int) ([]RankingRow, error) {
	var rows []RankingRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT c.user_id AS user_id, u.nombre AS nombre, COUNT(*) AS total
		 FROM certificate c
		 JOIN "user" u ON u.id = c.user_id
		 GROUP BY c.user_id, u.nombre
		 ORDER BY total DESC, u.nombre ASC, c.user_id ASC
		 LIMIT ?`,
		n,
	).Scan(&rows).Error
	return rows, err
}
