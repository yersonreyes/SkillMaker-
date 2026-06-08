package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── Mock repository ────────────────────────────────────────────────────────────

type mockRepo struct {
	GetByUserCourseFn      func(ctx context.Context, userID, courseID string) (*domain.Certificate, error)
	CreateFn               func(ctx context.Context, cert *domain.Certificate) error
	GetByIDFn              func(ctx context.Context, certID string) (*domain.Certificate, error)
	GetByCodigoFn          func(ctx context.Context, codigo string) (*domain.Certificate, error)
	ListByUserFn           func(ctx context.Context, userID string) ([]domain.Certificate, error)
	CountByUserFn          func(ctx context.Context, userID string) (int64, error)
	ListBadgesByUserFn     func(ctx context.Context, userID string) ([]BadgeWithGrant, error)
	ListBadgesUpToThreshFn func(ctx context.Context, count int64) ([]domain.Badge, error)
	AwardBadgeFn           func(ctx context.Context, userID, badgeID string) error
	RankingFn              func(ctx context.Context, n int) ([]RankingRow, error)
	createCalls            []string // tracks certIDs created
	awardCalls             []string // tracks badgeIDs awarded
}

func (m *mockRepo) GetByUserCourse(ctx context.Context, userID, courseID string) (*domain.Certificate, error) {
	if m.GetByUserCourseFn != nil {
		return m.GetByUserCourseFn(ctx, userID, courseID)
	}
	return nil, ErrCertificateNotFound
}

func (m *mockRepo) Create(ctx context.Context, cert *domain.Certificate) error {
	m.createCalls = append(m.createCalls, cert.ID)
	if m.CreateFn != nil {
		return m.CreateFn(ctx, cert)
	}
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, certID string) (*domain.Certificate, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, certID)
	}
	return nil, ErrCertificateNotFound
}

func (m *mockRepo) GetByCodigo(ctx context.Context, codigo string) (*domain.Certificate, error) {
	if m.GetByCodigoFn != nil {
		return m.GetByCodigoFn(ctx, codigo)
	}
	return nil, ErrCertificateNotFound
}

func (m *mockRepo) ListByUser(ctx context.Context, userID string) ([]domain.Certificate, error) {
	if m.ListByUserFn != nil {
		return m.ListByUserFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockRepo) CountByUser(ctx context.Context, userID string) (int64, error) {
	if m.CountByUserFn != nil {
		return m.CountByUserFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockRepo) ListBadgesByUser(ctx context.Context, userID string) ([]BadgeWithGrant, error) {
	if m.ListBadgesByUserFn != nil {
		return m.ListBadgesByUserFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockRepo) ListBadgesUpToThreshold(ctx context.Context, count int64) ([]domain.Badge, error) {
	if m.ListBadgesUpToThreshFn != nil {
		return m.ListBadgesUpToThreshFn(ctx, count)
	}
	return nil, nil
}

func (m *mockRepo) AwardBadge(ctx context.Context, userID, badgeID string) error {
	m.awardCalls = append(m.awardCalls, badgeID)
	if m.AwardBadgeFn != nil {
		return m.AwardBadgeFn(ctx, userID, badgeID)
	}
	return nil
}

func (m *mockRepo) Ranking(ctx context.Context, n int) ([]RankingRow, error) {
	if m.RankingFn != nil {
		return m.RankingFn(ctx, n)
	}
	return nil, nil
}

// ── Mock storage ────────────────────────────────────────────────────────────────

type mockStore struct {
	PutObjectFn     func(ctx context.Context, key string, r io.Reader, size int64, ct string) error
	PresignGetURLFn func(ctx context.Context, key string, ttl time.Duration) (string, error)
	putObjectCalls  []string
}

func (m *mockStore) PutObject(ctx context.Context, key string, r io.Reader, size int64, ct string) error {
	m.putObjectCalls = append(m.putObjectCalls, key)
	if m.PutObjectFn != nil {
		return m.PutObjectFn(ctx, key, r, size, ct)
	}
	return nil
}

func (m *mockStore) PresignPutURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}

func (m *mockStore) PresignGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if m.PresignGetURLFn != nil {
		return m.PresignGetURLFn(ctx, key, ttl)
	}
	return "https://presigned/" + key, nil
}

func (m *mockStore) Delete(_ context.Context, _ string) error { return nil }
func (m *mockStore) Ping(_ context.Context) error             { return nil }

var _ storage.Client = &mockStore{}

// ── Mock seams ──────────────────────────────────────────────────────────────────

type mockUserNames struct {
	nombre string
	err    error
}

func (m *mockUserNames) GetUserNombre(_ context.Context, _ string) (string, error) {
	return m.nombre, m.err
}

type mockCourseTitulos struct {
	titulo string
	err    error
}

func (m *mockCourseTitulos) GetCourseTitulo(_ context.Context, _ string) (string, error) {
	return m.titulo, m.err
}

// ── helpers ────────────────────────────────────────────────────────────────────

func newTestService(repo *mockRepo, store *mockStore) Service {
	return &serviceImpl{
		repo:          repo,
		store:         store,
		userNames:     &mockUserNames{nombre: "Test User"},
		courseTitulos: &mockCourseTitulos{titulo: "Test Course"},
		renderPDF:     func(_, _ string, _ time.Time, _ string) ([]byte, error) { return []byte("PDF"), nil },
		presignTTL:    15 * time.Minute,
	}
}

// ── TestIssueOnPass_HappyPath ──────────────────────────────────────────────────

func TestIssueOnPass_HappyPath(t *testing.T) {
	repo := &mockRepo{}
	store := &mockStore{}
	svc := newTestService(repo, store)

	userID := uuid.New().String()
	courseID := uuid.New().String()

	err := svc.IssueOnPass(context.Background(), userID, courseID)

	require.NoError(t, err, "IssueOnPass happy path must return nil")
	assert.Len(t, store.putObjectCalls, 1, "PutObject must be called once")
	assert.Len(t, repo.createCalls, 1, "Create must be called once")
	// PutObject key must use certificates/{certID}.pdf format.
	assert.Contains(t, store.putObjectCalls[0], "certificates/", "storage key must be in certificates/ prefix")
	assert.Contains(t, store.putObjectCalls[0], ".pdf", "storage key must end in .pdf")
}

// ── TestIssueOnPass_Idempotent ─────────────────────────────────────────────────

func TestIssueOnPass_Idempotent(t *testing.T) {
	existing := &domain.Certificate{
		ID:     uuid.New().String(),
		UserID: "u1", CourseID: "c1",
	}
	repo := &mockRepo{
		GetByUserCourseFn: func(_ context.Context, _, _ string) (*domain.Certificate, error) {
			return existing, nil // cert already exists
		},
	}
	store := &mockStore{}
	svc := newTestService(repo, store)

	err := svc.IssueOnPass(context.Background(), "u1", "c1")

	require.NoError(t, err, "IssueOnPass must return nil when cert already exists (idempotent)")
	assert.Empty(t, store.putObjectCalls, "PutObject must NOT be called when cert already exists")
	assert.Empty(t, repo.createCalls, "Create must NOT be called when cert already exists")
}

// ── TestIssueOnPass_StorageFailure ─────────────────────────────────────────────

func TestIssueOnPass_StorageFailure(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	repo := &mockRepo{}
	store := &mockStore{
		PutObjectFn: func(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
			return storageErr
		},
	}
	svc := newTestService(repo, store)

	err := svc.IssueOnPass(context.Background(), "u1", "c1")

	require.Error(t, err, "IssueOnPass must return error when PutObject fails")
	assert.ErrorIs(t, err, storageErr)
}

// ── TestIssueOnPass_EvaluateBadgesErrorIsSwallowed ─────────────────────────────

func TestIssueOnPass_EvaluateBadgesErrorIsSwallowed(t *testing.T) {
	badgeErr := errors.New("badge DB down")
	repo := &mockRepo{
		CountByUserFn: func(_ context.Context, _ string) (int64, error) {
			return 0, badgeErr
		},
	}
	store := &mockStore{}
	svc := newTestService(repo, store)

	err := svc.IssueOnPass(context.Background(), "u1", "c1")

	require.NoError(t, err,
		"IssueOnPass must return nil when EvaluateBadges fails (non-fatal — swallowed)")
}

// ── TestEvaluateBadges_Thresholds ─────────────────────────────────────────────

func TestEvaluateBadges_Thresholds(t *testing.T) {
	t.Run("1 cert awards Primer badge only", func(t *testing.T) {
		primerID := uuid.New().String()
		repo := &mockRepo{
			CountByUserFn: func(_ context.Context, _ string) (int64, error) { return 1, nil },
			ListBadgesUpToThreshFn: func(_ context.Context, count int64) ([]domain.Badge, error) {
				assert.Equal(t, int64(1), count)
				return []domain.Badge{{ID: primerID, Nombre: "Primer curso completado", Umbral: 1}}, nil
			},
		}
		store := &mockStore{}
		svc := newTestService(repo, store)

		err := svc.EvaluateBadges(context.Background(), "u1")
		require.NoError(t, err)
		assert.Equal(t, []string{primerID}, repo.awardCalls)
	})

	t.Run("5 certs awards Primer + 5 cursos badges", func(t *testing.T) {
		primerID := uuid.New().String()
		cincoID := uuid.New().String()
		repo := &mockRepo{
			CountByUserFn: func(_ context.Context, _ string) (int64, error) { return 5, nil },
			ListBadgesUpToThreshFn: func(_ context.Context, count int64) ([]domain.Badge, error) {
				assert.Equal(t, int64(5), count)
				return []domain.Badge{
					{ID: primerID, Nombre: "Primer curso completado", Umbral: 1},
					{ID: cincoID, Nombre: "5 cursos completados", Umbral: 5},
				}, nil
			},
		}
		store := &mockStore{}
		svc := newTestService(repo, store)

		err := svc.EvaluateBadges(context.Background(), "u1")
		require.NoError(t, err)
		assert.Contains(t, repo.awardCalls, primerID)
		assert.Contains(t, repo.awardCalls, cincoID)
	})

	t.Run("10 certs awards all 3 badges", func(t *testing.T) {
		primerID := uuid.New().String()
		cincoID := uuid.New().String()
		diezID := uuid.New().String()
		repo := &mockRepo{
			CountByUserFn: func(_ context.Context, _ string) (int64, error) { return 10, nil },
			ListBadgesUpToThreshFn: func(_ context.Context, count int64) ([]domain.Badge, error) {
				assert.Equal(t, int64(10), count)
				return []domain.Badge{
					{ID: primerID, Nombre: "Primer curso completado", Umbral: 1},
					{ID: cincoID, Nombre: "5 cursos completados", Umbral: 5},
					{ID: diezID, Nombre: "10 cursos completados", Umbral: 10},
				}, nil
			},
		}
		store := &mockStore{}
		svc := newTestService(repo, store)

		err := svc.EvaluateBadges(context.Background(), "u1")
		require.NoError(t, err)
		assert.Len(t, repo.awardCalls, 3, "must award 3 badges for 10 certs")
	})
}

// ── TestGetCertificate_OwnerScoped ────────────────────────────────────────────

func TestGetCertificate_OwnerScoped(t *testing.T) {
	ownerID := uuid.New().String()
	otherID := uuid.New().String()
	certID := uuid.New().String()

	cert := &domain.Certificate{ID: certID, UserID: ownerID, CourseID: uuid.New().String()}
	repo := &mockRepo{
		GetByIDFn: func(_ context.Context, id string) (*domain.Certificate, error) {
			if id == certID {
				return cert, nil
			}
			return nil, ErrCertificateNotFound
		},
	}
	store := &mockStore{}
	svc := newTestService(repo, store)

	t.Run("owner gets cert", func(t *testing.T) {
		got, err := svc.GetCertificate(context.Background(), certID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, certID, got.ID)
	})

	t.Run("non-owner gets ErrCertificateNotFound (anti-enum)", func(t *testing.T) {
		_, err := svc.GetCertificate(context.Background(), certID, otherID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCertificateNotFound,
			"non-owner must get ErrCertificateNotFound (not ErrNotOwner) — anti-enumeration")
	})

	t.Run("missing cert gets ErrCertificateNotFound", func(t *testing.T) {
		_, err := svc.GetCertificate(context.Background(), uuid.New().String(), ownerID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCertificateNotFound)
	})
}

// ── TestGetDownloadURL_OwnerScoped ────────────────────────────────────────────

func TestGetDownloadURL_OwnerScoped(t *testing.T) {
	ownerID := uuid.New().String()
	otherID := uuid.New().String()
	certID := uuid.New().String()

	cert := &domain.Certificate{ID: certID, UserID: ownerID, StorageKey: "certificates/test.pdf"}
	repo := &mockRepo{
		GetByIDFn: func(_ context.Context, id string) (*domain.Certificate, error) {
			if id == certID {
				return cert, nil
			}
			return nil, ErrCertificateNotFound
		},
	}
	store := &mockStore{}
	svc := newTestService(repo, store)

	t.Run("owner gets download URL", func(t *testing.T) {
		result, err := svc.GetDownloadURL(context.Background(), certID, ownerID)
		require.NoError(t, err)
		assert.NotEmpty(t, result.URL, "download URL must not be empty")
		assert.False(t, result.ExpiresAt.IsZero(), "expiresAt must be set")
	})

	t.Run("non-owner gets ErrCertificateNotFound", func(t *testing.T) {
		_, err := svc.GetDownloadURL(context.Background(), certID, otherID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCertificateNotFound)
	})

	t.Run("cert with empty storage_key returns ErrNoPDF", func(t *testing.T) {
		noPDFCert := &domain.Certificate{ID: certID, UserID: ownerID, StorageKey: ""}
		repo2 := &mockRepo{
			GetByIDFn: func(_ context.Context, _ string) (*domain.Certificate, error) {
				return noPDFCert, nil
			},
		}
		svc2 := newTestService(repo2, store)
		_, err := svc2.GetDownloadURL(context.Background(), certID, ownerID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoPDF, "empty storage_key must return ErrNoPDF")
	})
}

// ── TestUserNameReaderFailure_NonFatalToAttempt ───────────────────────────────

func TestUserNameReaderFailure_IsReturnedFromIssueOnPass(t *testing.T) {
	userErr := errors.New("users service down")
	repo := &mockRepo{}
	store := &mockStore{}
	svc := &serviceImpl{
		repo:          repo,
		store:         store,
		userNames:     &mockUserNames{err: userErr},
		courseTitulos: &mockCourseTitulos{titulo: "Test"},
		renderPDF:     func(_, _ string, _ time.Time, _ string) ([]byte, error) { return []byte("PDF"), nil },
		presignTTL:    15 * time.Minute,
	}

	err := svc.IssueOnPass(context.Background(), "u1", "c1")
	// The error is returned from IssueOnPass — evaluations.SubmitAttempt swallows it.
	require.Error(t, err)
	assert.ErrorIs(t, err, userErr)
}

// ── Notifier seam tests (notifications-inapp) ─────────────────────────────────

// TestIssueOnPass_FirstIssue_FiresNotifierOnce verifies that IssueOnPass fires
// the Notifier exactly once on the FIRST issue path.
func TestIssueOnPass_FirstIssue_FiresNotifierOnce(t *testing.T) {
	repo := &mockRepo{}
	store := &mockStore{}
	notifier := &testutil.MockNotifier{}

	svc := &serviceImpl{
		repo:          repo,
		store:         store,
		userNames:     &mockUserNames{nombre: "Test User"},
		courseTitulos: &mockCourseTitulos{titulo: "Test Course"},
		renderPDF:     func(_, _ string, _ time.Time, _ string) ([]byte, error) { return []byte("PDF"), nil },
		presignTTL:    15 * time.Minute,
		notifier:      notifier,
	}

	userID := uuid.New().String()
	courseID := uuid.New().String()

	// The certID is generated inside IssueOnPass; we accept any non-empty string.
	notifier.On("Notify", mock.Anything, userID, "certificado_emitido", "Certificado emitido", "Test Course", mock.MatchedBy(func(s string) bool {
		return s != "" // certID must be non-empty
	})).Return(nil).Once()

	err := svc.IssueOnPass(context.Background(), userID, courseID)
	require.NoError(t, err, "IssueOnPass first issue must return nil")
	notifier.AssertExpectations(t)
	notifier.AssertNumberOfCalls(t, "Notify", 1)
}

// TestIssueOnPass_IdempotentReissue_NotifierNotCalled verifies that when the cert
// already exists (idempotent re-call), the Notifier is NOT called.
func TestIssueOnPass_IdempotentReissue_NotifierNotCalled(t *testing.T) {
	existing := &domain.Certificate{
		ID:       uuid.New().String(),
		UserID:   "u1",
		CourseID: "c1",
	}
	repo := &mockRepo{
		GetByUserCourseFn: func(_ context.Context, _, _ string) (*domain.Certificate, error) {
			return existing, nil // already issued
		},
	}
	store := &mockStore{}
	notifier := &testutil.MockNotifier{}

	svc := &serviceImpl{
		repo:          repo,
		store:         store,
		userNames:     &mockUserNames{nombre: "Test User"},
		courseTitulos: &mockCourseTitulos{titulo: "Test Course"},
		renderPDF:     func(_, _ string, _ time.Time, _ string) ([]byte, error) { return []byte("PDF"), nil },
		presignTTL:    15 * time.Minute,
		notifier:      notifier,
	}

	err := svc.IssueOnPass(context.Background(), "u1", "c1")
	require.NoError(t, err, "idempotent re-issue must return nil")
	// Notifier must NOT be called on the idempotent re-issue path.
	notifier.AssertNotCalled(t, "Notify")
}

// TestIssueOnPass_NotifierFails_StillReturnsNil verifies NON-FATAL: a failing
// Notifier does not break IssueOnPass.
func TestIssueOnPass_NotifierFails_StillReturnsNil(t *testing.T) {
	repo := &mockRepo{}
	store := &mockStore{}
	notifier := &testutil.MockNotifier{}

	svc := &serviceImpl{
		repo:          repo,
		store:         store,
		userNames:     &mockUserNames{nombre: "Test User"},
		courseTitulos: &mockCourseTitulos{titulo: "Test Course"},
		renderPDF:     func(_, _ string, _ time.Time, _ string) ([]byte, error) { return []byte("PDF"), nil },
		presignTTL:    15 * time.Minute,
		notifier:      notifier,
	}

	notifier.On("Notify", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("notification service down"))

	err := svc.IssueOnPass(context.Background(), "u1", "c1")
	assert.NoError(t, err, "NON-FATAL: IssueOnPass must return nil when Notifier fails")
}

// ── VerifyCertificate (PUBLIC, by code) ─────────────────────────────────────────

// TestVerifyCertificate_HappyPath composes holder name + course title via the seams.
func TestVerifyCertificate_HappyPath(t *testing.T) {
	emitido := time.Now()
	repo := &mockRepo{
		GetByCodigoFn: func(_ context.Context, codigo string) (*domain.Certificate, error) {
			assert.Equal(t, "ABCD1234EFG12", codigo)
			return &domain.Certificate{
				ID: uuid.New().String(), UserID: "u1", CourseID: "c1",
				Codigo: "ABCD1234EFG12", StorageKey: "certificates/x.pdf", EmitidoEn: emitido,
			}, nil
		},
	}
	svc := newTestService(repo, &mockStore{})

	res, err := svc.VerifyCertificate(context.Background(), "ABCD1234EFG12")
	require.NoError(t, err)
	assert.Equal(t, "ABCD1234EFG12", res.Codigo)
	assert.Equal(t, "Test User", res.HolderNombre, "holder name composed via UserNameReader seam")
	assert.Equal(t, "Test Course", res.CourseTitulo, "course title composed via CourseTituloReader seam")
	assert.Equal(t, emitido, res.EmitidoEn)
}

// TestVerifyCertificate_NotFound returns ErrCertificateNotFound for an unknown code.
func TestVerifyCertificate_NotFound(t *testing.T) {
	repo := &mockRepo{
		GetByCodigoFn: func(_ context.Context, _ string) (*domain.Certificate, error) {
			return nil, ErrCertificateNotFound
		},
	}
	svc := newTestService(repo, &mockStore{})

	res, err := svc.VerifyCertificate(context.Background(), "UNKNOWN")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, ErrCertificateNotFound)
}
