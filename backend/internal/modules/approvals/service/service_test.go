// Package service — white-box unit tests for the approvals service.
// No build tag: runs with standard `make backend-test`.
//
// Strategy: inject MockCourseStateManager + MockEvaluationValidator (from testutil)
// + a local mockApprovalsRepo so no real DB is needed.
//
// TDD: tests are written BEFORE the service implementation (Strict TDD per C4.1).
//
// Covered invariants:
//   - SubmitToReview: happy path + each guard in order (owner→transition→content→eval)
//   - ORDER probe: non-owner on no-content course → ErrNotOwner (owner checked FIRST)
//   - Approve: happy + ErrNotInReview (no Create/SetEstado on wrong estado)
//   - Two-write order: Create called before SetEstado (InOrder assertion)
//   - Reject: happy + ErrCommentRequired (asserts NO Create/SetEstado called) + ErrNotInReview
//   - ListPending: delegates to ListByEstado("en_revision")
//   - ListHistory: admin sees any, owner sees own, non-owner non-admin → ErrNotOwner
package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── mockApprovalsRepo ─────────────────────────────────────────────────────────

type mockApprovalsRepo struct {
	mock.Mock
	createCalls []time.Time // records call times for InOrder assertions
}

func (m *mockApprovalsRepo) Create(ctx context.Context, a *domain.Approval) error {
	m.createCalls = append(m.createCalls, time.Now())
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *mockApprovalsRepo) ListByCourse(ctx context.Context, courseID string) ([]domain.Approval, error) {
	args := m.Called(ctx, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Approval), args.Error(1)
}

// ── helper ────────────────────────────────────────────────────────────────────

func newSvc(repo Repository, csm CourseStateManager, ev EvaluationValidator) *serviceImpl {
	return New(repo, csm, ev).(*serviceImpl)
}

// ── SubmitToReview tests ──────────────────────────────────────────────────────

// TestSubmitToReview_Happy verifies the happy path: owner, borrador, hasContent, eval ready.
func TestSubmitToReview_Happy(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	csm.On("HasContent", mock.Anything, courseID, ownerID).Return(true, nil)
	ev.On("ValidateSubmitReady", mock.Anything, courseID).Return(nil)
	csm.On("SetEstado", mock.Anything, courseID, "en_revision").Return(nil)

	err := svc.SubmitToReview(context.Background(), courseID, ownerID)
	assert.NoError(t, err, "happy path submit must succeed")
	csm.AssertExpectations(t)
	ev.AssertExpectations(t)
}

// TestSubmitToReview_NotOwner_ReturnsErrNotOwner verifies owner check is FIRST.
func TestSubmitToReview_NotOwner_ReturnsErrNotOwner(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()
	callerID := uuid.New().String() // different caller

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)

	err := svc.SubmitToReview(context.Background(), courseID, callerID)
	assert.ErrorIs(t, err, ErrNotOwner, "non-owner must get ErrNotOwner")
	// HasContent must NOT be called before ownership is checked.
	csm.AssertNotCalled(t, "HasContent")
	ev.AssertNotCalled(t, "ValidateSubmitReady")
}

// TestSubmitToReview_InvalidTransition_AprobadoBlocked verifies aprobado cannot resubmit.
func TestSubmitToReview_InvalidTransition_AprobadoBlocked(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "aprobado", nil)

	err := svc.SubmitToReview(context.Background(), courseID, ownerID)
	assert.ErrorIs(t, err, ErrCourseNotSubmittable,
		"aprobado course must return ErrCourseNotSubmittable")
	csm.AssertNotCalled(t, "HasContent")
	ev.AssertNotCalled(t, "ValidateSubmitReady")
}

// TestSubmitToReview_AlreadyEnRevision_Blocked verifies en_revision cannot resubmit.
func TestSubmitToReview_AlreadyEnRevision_Blocked(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "en_revision", nil)

	err := svc.SubmitToReview(context.Background(), courseID, ownerID)
	assert.ErrorIs(t, err, ErrCourseNotSubmittable,
		"en_revision course must return ErrCourseNotSubmittable")
}

// TestSubmitToReview_NoContent_ReturnsErrNoContent verifies content check.
func TestSubmitToReview_NoContent_ReturnsErrNoContent(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	csm.On("HasContent", mock.Anything, courseID, ownerID).Return(false, nil)

	err := svc.SubmitToReview(context.Background(), courseID, ownerID)
	assert.ErrorIs(t, err, ErrNoContent, "course without content must return ErrNoContent")
	ev.AssertNotCalled(t, "ValidateSubmitReady")
}

// TestSubmitToReview_RechazadoAllowed verifies rechazado can resubmit.
func TestSubmitToReview_RechazadoAllowed(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "rechazado", nil)
	csm.On("HasContent", mock.Anything, courseID, ownerID).Return(true, nil)
	ev.On("ValidateSubmitReady", mock.Anything, courseID).Return(nil)
	csm.On("SetEstado", mock.Anything, courseID, "en_revision").Return(nil)

	err := svc.SubmitToReview(context.Background(), courseID, ownerID)
	assert.NoError(t, err, "rechazado course can resubmit")
	csm.AssertExpectations(t)
}

// TestSubmitToReview_EvalNotFound_Surfaced verifies eval sentinel propagation.
func TestSubmitToReview_EvalNotFound_Surfaced(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	csm.On("HasContent", mock.Anything, courseID, ownerID).Return(true, nil)
	ev.On("ValidateSubmitReady", mock.Anything, courseID).Return(testutil.ErrEvaluationNotFoundSentinel)

	err := svc.SubmitToReview(context.Background(), courseID, ownerID)
	assert.ErrorIs(t, err, testutil.ErrEvaluationNotFoundSentinel,
		"ErrEvaluationNotFound must be surfaced verbatim from the evaluations seam")
	csm.AssertNotCalled(t, "SetEstado")
}

// TestSubmitToReview_EvalIncomplete_Surfaced verifies that when the EvaluationValidator
// returns the evaluation-incomplete sentinel (evaluations.ErrNoCorrectOption), that error
// is surfaced verbatim via errors.Is and SetEstado is NOT called (W-1 verify warning).
//
// Uses the REAL evaluations.ErrNoCorrectOption sentinel (re-exported from evaluations facade)
// so the test exercises the true errors.Is chain — not a test-only proxy.
func TestSubmitToReview_EvalIncomplete_Surfaced(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	csm.On("HasContent", mock.Anything, courseID, ownerID).Return(true, nil)
	// Return the REAL evaluations.ErrNoCorrectOption sentinel so errors.Is works on the full chain.
	ev.On("ValidateSubmitReady", mock.Anything, courseID).Return(evaluations.ErrNoCorrectOption)

	err := svc.SubmitToReview(context.Background(), courseID, ownerID)
	assert.ErrorIs(t, err, evaluations.ErrNoCorrectOption,
		"evaluation-incomplete sentinel must be surfaced verbatim by SubmitToReview")
	csm.AssertNotCalled(t, "SetEstado",
		"SetEstado must NOT be called when evaluation is incomplete")
}

// TestSubmitToReview_ORDER_Probe_NonOwnerOnNoContent_ReturnsErrNotOwner verifies
// the ownership check runs BEFORE content check (D5 ordering).
// A non-owner on a course with no content → must still get ErrNotOwner (owner first).
func TestSubmitToReview_ORDER_Probe_NonOwnerOnNoContent_ReturnsErrNotOwner(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()
	callerID := uuid.New().String() // not the owner

	// Caller is not owner; course has no content (but we should never reach that check).
	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)

	err := svc.SubmitToReview(context.Background(), courseID, callerID)
	assert.ErrorIs(t, err, ErrNotOwner,
		"ORDER PROBE: non-owner on no-content course must get ErrNotOwner (owner before content)")
	csm.AssertNotCalled(t, "HasContent",
		"HasContent must NOT be called before ownership is verified")
}

// ── Approve tests ─────────────────────────────────────────────────────────────

// TestApprove_Happy verifies approve creates audit row then sets estado.
func TestApprove_Happy(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	adminID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return("owner", "en_revision", nil)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(a *domain.Approval) bool {
		return a.CourseID == courseID && a.AdminID == adminID && a.Resultado == "aprobado"
	})).Return(nil)
	csm.On("SetEstado", mock.Anything, courseID, "aprobado").Return(nil)

	err := svc.Approve(context.Background(), courseID, adminID, "")
	assert.NoError(t, err, "approve on en_revision must succeed")
	csm.AssertExpectations(t)
	repo.AssertExpectations(t)
}

// TestApprove_NotInReview_ReturnsErrNotInReview verifies estado guard.
func TestApprove_NotInReview_ReturnsErrNotInReview(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	adminID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return("owner", "borrador", nil)

	err := svc.Approve(context.Background(), courseID, adminID, "")
	assert.ErrorIs(t, err, ErrNotInReview,
		"approve on borrador must return ErrNotInReview")
	repo.AssertNotCalled(t, "Create")
	csm.AssertNotCalled(t, "SetEstado")
}

// TestApprove_TwoWriteOrder_CreateBeforeSetEstado verifies history-first ordering (D6/R1).
func TestApprove_TwoWriteOrder_CreateBeforeSetEstado(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	adminID := uuid.New().String()

	var callOrder []string
	csm.On("GetCourseOwnership", mock.Anything, courseID).Return("owner", "en_revision", nil)
	repo.On("Create", mock.Anything, mock.Anything).Run(func(_ mock.Arguments) {
		callOrder = append(callOrder, "Create")
	}).Return(nil)
	csm.On("SetEstado", mock.Anything, courseID, "aprobado").Run(func(_ mock.Arguments) {
		callOrder = append(callOrder, "SetEstado")
	}).Return(nil)

	err := svc.Approve(context.Background(), courseID, adminID, "comment")
	require.NoError(t, err)
	require.Len(t, callOrder, 2, "must have exactly 2 writes (Create + SetEstado)")
	assert.Equal(t, "Create", callOrder[0], "Create must be called BEFORE SetEstado (two-write ordering R1)")
	assert.Equal(t, "SetEstado", callOrder[1], "SetEstado must be called AFTER Create (two-write ordering R1)")
}

// ── Reject tests ──────────────────────────────────────────────────────────────

// TestReject_Happy verifies reject with a non-empty comment works.
func TestReject_Happy(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	adminID := uuid.New().String()
	comentario := "Needs more examples"

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return("owner", "en_revision", nil)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(a *domain.Approval) bool {
		return a.Resultado == "rechazado" && a.Comentario == comentario
	})).Return(nil)
	csm.On("SetEstado", mock.Anything, courseID, "rechazado").Return(nil)

	err := svc.Reject(context.Background(), courseID, adminID, comentario)
	assert.NoError(t, err, "reject with non-empty comment must succeed")
	csm.AssertExpectations(t)
	repo.AssertExpectations(t)
}

// TestReject_EmptyComment_ReturnsErrCommentRequired verifies comment check is FIRST (SEC-7).
func TestReject_EmptyComment_ReturnsErrCommentRequired(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	adminID := uuid.New().String()

	// Empty comment — must return ErrCommentRequired WITHOUT any DB/seam calls.
	err := svc.Reject(context.Background(), courseID, adminID, "")
	assert.ErrorIs(t, err, ErrCommentRequired,
		"empty comment must return ErrCommentRequired (SEC-7)")
	csm.AssertNotCalled(t, "GetCourseOwnership")
	repo.AssertNotCalled(t, "Create")
	csm.AssertNotCalled(t, "SetEstado")
}

// TestReject_WhitespaceComment_ReturnsErrCommentRequired verifies whitespace-only comment is rejected.
func TestReject_WhitespaceComment_ReturnsErrCommentRequired(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	err := svc.Reject(context.Background(), uuid.New().String(), uuid.New().String(), "   ")
	assert.ErrorIs(t, err, ErrCommentRequired,
		"whitespace-only comment must return ErrCommentRequired")
	repo.AssertNotCalled(t, "Create")
}

// TestReject_NotInReview_ReturnsErrNotInReview verifies estado guard fires after comment check.
func TestReject_NotInReview_ReturnsErrNotInReview(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	adminID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return("owner", "aprobado", nil)

	err := svc.Reject(context.Background(), courseID, adminID, "valid comment")
	assert.ErrorIs(t, err, ErrNotInReview,
		"reject on non-en_revision course must return ErrNotInReview")
	repo.AssertNotCalled(t, "Create")
	csm.AssertNotCalled(t, "SetEstado")
}

// ── ListPending tests ─────────────────────────────────────────────────────────

// TestListPending_DelegatesToListByEstado verifies delegation to ListByEstado("en_revision").
func TestListPending_DelegatesToListByEstado(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	pending := []courses.CourseSummary{
		{ID: uuid.New().String(), Titulo: "Pending 1", Estado: "en_revision"},
	}
	csm.On("ListByEstado", mock.Anything, "en_revision").Return(pending, nil)

	result, err := svc.ListPending(context.Background())
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, pending[0].ID, result[0].ID)
	csm.AssertExpectations(t)
}

// ── ListHistory tests ─────────────────────────────────────────────────────────

// TestListHistory_Admin_SeesAnyHistory verifies admin can see any course's history.
func TestListHistory_Admin_SeesAnyHistory(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()
	adminID := uuid.New().String() // different from owner

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "aprobado", nil)
	expectedRows := []domain.Approval{{ID: uuid.New().String(), CourseID: courseID, Resultado: "aprobado"}}
	repo.On("ListByCourse", mock.Anything, courseID).Return(expectedRows, nil)

	rows, err := svc.ListHistory(context.Background(), courseID, adminID, true /* isAdmin */)
	require.NoError(t, err)
	assert.Len(t, rows, 1, "admin must see all history rows")
	csm.AssertExpectations(t)
}

// TestListHistory_Owner_SeesOwnHistory verifies owner can see their course's history.
func TestListHistory_Owner_SeesOwnHistory(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "rechazado", nil)
	expectedRows := []domain.Approval{{ID: uuid.New().String(), CourseID: courseID, Resultado: "rechazado"}}
	repo.On("ListByCourse", mock.Anything, courseID).Return(expectedRows, nil)

	rows, err := svc.ListHistory(context.Background(), courseID, ownerID, false /* isAdmin */)
	require.NoError(t, err)
	assert.Len(t, rows, 1, "owner must see their own course's history")
}

// TestListHistory_NonOwnerNonAdmin_ReturnsErrNotOwner verifies access control.
func TestListHistory_NonOwnerNonAdmin_ReturnsErrNotOwner(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	ownerID := uuid.New().String()
	callerID := uuid.New().String() // not owner, not admin

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "rechazado", nil)

	_, err := svc.ListHistory(context.Background(), courseID, callerID, false /* isAdmin */)
	assert.ErrorIs(t, err, ErrNotOwner,
		"non-owner non-admin must get ErrNotOwner for history")
	repo.AssertNotCalled(t, "ListByCourse")
}

// ── Notifier seam tests (notifications-inapp) ─────────────────────────────────

// TestApprove_FiresNotifierWithCreatorID verifies that after a successful Approve,
// the Notifier is called with the course creator's ID and tipo='curso_aprobado'.
func TestApprove_FiresNotifierWithCreatorID(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	notifier := &testutil.MockNotifier{}

	svc := New(repo, csm, ev, WithNotifier(notifier)).(*serviceImpl)

	courseID := uuid.New().String()
	adminID := uuid.New().String()
	creatorID := uuid.New().String()
	titulo := "My Course"

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(creatorID, "en_revision", nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil)
	csm.On("SetEstado", mock.Anything, courseID, "aprobado").Return(nil)
	csm.On("GetCourseTitulo", mock.Anything, courseID).Return(titulo, nil)
	notifier.On("Notify", mock.Anything, creatorID, "curso_aprobado", "Curso aprobado", titulo, courseID).Return(nil)

	err := svc.Approve(context.Background(), courseID, adminID, "")
	require.NoError(t, err)
	notifier.AssertExpectations(t)
}

// TestReject_FiresNotifierWithComentario verifies that after a successful Reject,
// the Notifier is called with tipo='curso_rechazado' and cuerpo containing the comentario.
func TestReject_FiresNotifierWithComentario(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	notifier := &testutil.MockNotifier{}

	svc := New(repo, csm, ev, WithNotifier(notifier)).(*serviceImpl)

	courseID := uuid.New().String()
	adminID := uuid.New().String()
	creatorID := uuid.New().String()
	comentario := "Falta bibliografía"

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return(creatorID, "en_revision", nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil)
	csm.On("SetEstado", mock.Anything, courseID, "rechazado").Return(nil)
	csm.On("GetCourseTitulo", mock.Anything, courseID).Return("My Course", nil)
	notifier.On("Notify", mock.Anything, creatorID, "curso_rechazado", "Curso rechazado", mock.MatchedBy(func(cuerpo string) bool {
		return strings.Contains(cuerpo, comentario)
	}), courseID).Return(nil)

	err := svc.Reject(context.Background(), courseID, adminID, comentario)
	require.NoError(t, err)
	notifier.AssertExpectations(t)
}

// TestApprove_NotifierFails_StillReturnsNil verifies NON-FATAL: a failing Notifier
// does not break Approve — it must still return nil.
func TestApprove_NotifierFails_StillReturnsNil(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	notifier := &testutil.MockNotifier{}

	svc := New(repo, csm, ev, WithNotifier(notifier)).(*serviceImpl)

	courseID := uuid.New().String()
	adminID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return("creator", "en_revision", nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil)
	csm.On("SetEstado", mock.Anything, courseID, "aprobado").Return(nil)
	csm.On("GetCourseTitulo", mock.Anything, courseID).Return("", nil)
	notifier.On("Notify", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("notification service down"))

	// NON-FATAL: Approve must return nil even when Notifier fails.
	err := svc.Approve(context.Background(), courseID, adminID, "")
	assert.NoError(t, err, "NON-FATAL: Approve must return nil when Notifier fails")
}

// TestReject_NotifierFails_StillReturnsNil verifies NON-FATAL for Reject.
func TestReject_NotifierFails_StillReturnsNil(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}
	notifier := &testutil.MockNotifier{}

	svc := New(repo, csm, ev, WithNotifier(notifier)).(*serviceImpl)

	courseID := uuid.New().String()
	adminID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return("creator", "en_revision", nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil)
	csm.On("SetEstado", mock.Anything, courseID, "rechazado").Return(nil)
	csm.On("GetCourseTitulo", mock.Anything, courseID).Return("", nil)
	notifier.On("Notify", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("notification service down"))

	err := svc.Reject(context.Background(), courseID, adminID, "valid comment")
	assert.NoError(t, err, "NON-FATAL: Reject must return nil when Notifier fails")
}

// TestApprove_NilNotifier_NoPanic verifies that omitting WithNotifier causes no panic.
func TestApprove_NilNotifier_NoPanic(t *testing.T) {
	repo := &mockApprovalsRepo{}
	csm := &testutil.MockCourseStateManager{}
	ev := &testutil.MockEvaluationValidator{}

	// No WithNotifier → notifier is nil.
	svc := newSvc(repo, csm, ev)

	courseID := uuid.New().String()
	adminID := uuid.New().String()

	csm.On("GetCourseOwnership", mock.Anything, courseID).Return("creator", "en_revision", nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil)
	csm.On("SetEstado", mock.Anything, courseID, "aprobado").Return(nil)

	assert.NotPanics(t, func() {
		err := svc.Approve(context.Background(), courseID, adminID, "")
		assert.NoError(t, err)
	}, "nil Notifier must not panic")
}
