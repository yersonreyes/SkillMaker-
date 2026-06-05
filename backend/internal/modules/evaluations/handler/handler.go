// Package handler contains the Gin HTTP handlers for the evaluations module.
// Handlers are intentionally thin: parse → call service → errors.Is → render.
// No domain logic lives here.
//
// CRITICAL: ErrNotOwner maps to DIFFERENT HTTP statuses depending on the route:
//   - GET routes → 404 via renderReadError  (hides existence, read convention)
//   - POST/PATCH/DELETE routes → 403 via renderWriteError (signals authz failure)
//
// This asymmetry is enforced by TWO separate render helpers, mirroring courses handler.
//
// CRITICAL Gin param convention (ADR-7):
//
//	GET tree uses :id  (POST /courses/:courseId/evaluation → GET /courses/:id/evaluation)
//	POST-create on courses uses :courseId
//	Mixing these on the shared creatorGrp would panic — preserved exactly here.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
)

// Handler holds the service dependency injected at registration time.
type Handler struct {
	svc service.Service
}

// Register mounts the evaluations routes onto a pre-built Gin route group.
// The group must already carry JWT + RequireRole("creador") middleware.
// CRITICAL: param names must match the courses handler convention to avoid Gin tree panics:
//   - GET  /courses/:id/evaluation    (same :id as GET /courses/:id/sections)
//   - POST /courses/:courseId/evaluation (same :courseId as POST /courses/:courseId/sections)
func Register(creatorGrp *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}

	// Evaluation routes.
	creatorGrp.POST("/courses/:courseId/evaluation", h.CreateEvaluation) // :courseId — POST tree
	creatorGrp.GET("/courses/:id/evaluation", h.GetEvaluation)           // :id — GET tree
	creatorGrp.PATCH("/evaluations/:id", h.UpdateEvaluation)

	// Question routes.
	creatorGrp.POST("/evaluations/:id/questions", h.CreateQuestion)
	creatorGrp.PATCH("/questions/:id", h.UpdateQuestion)
	creatorGrp.DELETE("/questions/:id", h.DeleteQuestion)

	// Option routes.
	creatorGrp.POST("/questions/:id/options", h.CreateOption)
	creatorGrp.PATCH("/options/:id", h.UpdateOption)
	creatorGrp.DELETE("/options/:id", h.DeleteOption)
}

// ── Evaluation handlers ────────────────────────────────────────────────────────

// CreateEvaluation godoc
// @Summary     Crea una evaluacion para un curso
// @Description Crea una evaluacion (1-1 por curso). El curso debe estar en borrador o rechazado.
// @Tags        evaluations
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       courseId path   string                       true "UUID del curso"
// @Param       body     body   dto.EvaluationCreateRequest  true "Datos de la evaluacion"
// @Success     201 {object} dto.EvaluationResponse
// @Failure     400 {object} httperr.Error "body invalido"
// @Failure     403 {object} httperr.Error "no es propietario del curso"
// @Failure     404 {object} httperr.Error "curso no encontrado"
// @Failure     409 {object} httperr.Error "ya existe evaluacion o curso no editable"
// @Router      /courses/{courseId}/evaluation [post]
func (h *Handler) CreateEvaluation(c *gin.Context) {
	courseID := c.Param("courseId")
	creadorID := middleware.UserIDFrom(c)

	var req dto.EvaluationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.CreateEvaluation(c.Request.Context(), courseID, creadorID, service.EvaluationCreateRequest{
		NotaMinima:  req.NotaMinima,
		IntentosMax: req.IntentosMax,
	})
	if err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToEvaluation(model))
}

// GetEvaluation godoc
// @Summary     Obtiene la evaluacion de un curso (arbol anidado)
// @Description Retorna la evaluacion con sus preguntas y opciones. Solo el propietario del curso.
//
//	ErrNotOwner → 404 (oculta existencia).
//
// @Tags        evaluations
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del curso"
// @Success     200 {object} dto.EvaluationDetail
// @Failure     404 {object} httperr.Error "evaluacion o curso no encontrado"
// @Router      /courses/{id}/evaluation [get]
func (h *Handler) GetEvaluation(c *gin.Context) {
	courseID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	detail, err := h.svc.GetEvaluation(c.Request.Context(), courseID, creadorID)
	if err != nil {
		h.renderReadError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToEvaluationDetail(detail))
}

// UpdateEvaluation godoc
// @Summary     Actualiza una evaluacion (PATCH parcial)
// @Description Actualiza notaMinima y/o intentosMax. El curso debe estar en borrador o rechazado.
// @Tags        evaluations
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string                       true "UUID de la evaluacion"
// @Param       body body   dto.EvaluationUpdateRequest  true "Campos a actualizar"
// @Success     200 {object} dto.EvaluationResponse
// @Failure     400 {object} httperr.Error
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Failure     409 {object} httperr.Error
// @Router      /evaluations/{id} [patch]
func (h *Handler) UpdateEvaluation(c *gin.Context) {
	evalID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	var req dto.EvaluationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.UpdateEvaluation(c.Request.Context(), evalID, creadorID, service.EvaluationUpdateRequest{
		NotaMinima:  req.NotaMinima,
		IntentosMax: req.IntentosMax,
	})
	if err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToEvaluation(model))
}

// ── Question handlers ──────────────────────────────────────────────────────────

// CreateQuestion godoc
// @Summary     Crea una pregunta en una evaluacion
// @Description Para verdadero_falso auto-crea 2 opciones. Para opcion_multiple empieza vacia.
// @Tags        questions
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string                      true "UUID de la evaluacion"
// @Param       body body   dto.QuestionCreateRequest   true "Datos de la pregunta"
// @Success     201 {object} dto.QuestionDetail
// @Failure     400 {object} httperr.Error "tipo invalido"
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Failure     409 {object} httperr.Error
// @Router      /evaluations/{id}/questions [post]
func (h *Handler) CreateQuestion(c *gin.Context) {
	evalID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	var req dto.QuestionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.CreateQuestion(c.Request.Context(), evalID, creadorID, service.QuestionCreateRequest{
		Enunciado: req.Enunciado,
		Tipo:      req.Tipo,
		Puntaje:   req.Puntaje,
	})
	if err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToQuestionDetail(model))
}

// UpdateQuestion godoc
// @Summary     Actualiza una pregunta (PATCH parcial)
// @Tags        questions
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string                      true "UUID de la pregunta"
// @Param       body body   dto.QuestionUpdateRequest   true "Campos a actualizar"
// @Success     200 {object} dto.QuestionResponse
// @Router      /questions/{id} [patch]
func (h *Handler) UpdateQuestion(c *gin.Context) {
	questionID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	var req dto.QuestionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.UpdateQuestion(c.Request.Context(), questionID, creadorID, service.QuestionUpdateRequest{
		Enunciado: req.Enunciado,
		Puntaje:   req.Puntaje,
		Orden:     req.Orden,
	})
	if err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToQuestion(model))
}

// DeleteQuestion godoc
// @Summary     Elimina una pregunta (y sus opciones en cascada)
// @Tags        questions
// @Security    BearerAuth
// @Param       id path string true "UUID de la pregunta"
// @Success     204
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Router      /questions/{id} [delete]
func (h *Handler) DeleteQuestion(c *gin.Context) {
	questionID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	if err := h.svc.DeleteQuestion(c.Request.Context(), questionID, creadorID); err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ── Option handlers ────────────────────────────────────────────────────────────

// CreateOption godoc
// @Summary     Agrega una opcion a una pregunta opcion_multiple
// @Tags        options
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string                  true "UUID de la pregunta"
// @Param       body body   dto.OptionCreateRequest true "Datos de la opcion"
// @Success     201 {object} dto.OptionResponse
// @Failure     400 {object} httperr.Error "tipo incorrecto (verdadero_falso no permite agregar opciones)"
// @Router      /questions/{id}/options [post]
func (h *Handler) CreateOption(c *gin.Context) {
	questionID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	var req dto.OptionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.CreateOption(c.Request.Context(), questionID, creadorID, service.OptionCreateRequest{
		Texto:    req.Texto,
		Correcta: req.Correcta,
	})
	if err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToOption(model))
}

// UpdateOption godoc
// @Summary     Actualiza una opcion (PATCH parcial)
// @Tags        options
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string                  true "UUID de la opcion"
// @Param       body body   dto.OptionUpdateRequest true "Campos a actualizar"
// @Success     200 {object} dto.OptionResponse
// @Router      /options/{id} [patch]
func (h *Handler) UpdateOption(c *gin.Context) {
	optionID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	var req dto.OptionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.UpdateOption(c.Request.Context(), optionID, creadorID, service.OptionUpdateRequest{
		Texto:    req.Texto,
		Correcta: req.Correcta,
	})
	if err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToOption(model))
}

// DeleteOption godoc
// @Summary     Elimina una opcion
// @Tags        options
// @Security    BearerAuth
// @Param       id path string true "UUID de la opcion"
// @Success     204
// @Router      /options/{id} [delete]
func (h *Handler) DeleteOption(c *gin.Context) {
	optionID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	if err := h.svc.DeleteOption(c.Request.Context(), optionID, creadorID); err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ── Student routes (C3.2) ─────────────────────────────────────────────────────

// RegisterStudent mounts the student attempt routes onto a JWT-only route group.
// Student routes are disjoint from creator routes:
//   - POST /evaluations/:id/attempts shares the :id wildcard segment with the creator
//     POST /evaluations/:id/questions (same param name, no Gin conflict).
//   - /attempts/:id is a brand-new top-level tree.
func RegisterStudent(protectedGrp *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}
	protectedGrp.POST("/evaluations/:id/attempts", h.StartAttempt)
	protectedGrp.GET("/attempts/:id", h.GetAttempt)
	protectedGrp.POST("/attempts/:id/answers", h.SaveAnswer)
	protectedGrp.POST("/attempts/:id/submit", h.SubmitAttempt)
}

// StartAttempt godoc
// @Summary     Inicia un nuevo intento de evaluacion
// @Description Crea un intento para el usuario autenticado en la evaluacion indicada.
// @Tags        attempts
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID de la evaluacion"
// @Success     201 {object} dto.AttemptStartResponse
// @Failure     404 {object} httperr.Error "evaluacion no encontrada"
// @Failure     409 {object} httperr.Error "intento abierto o maximo alcanzado"
// @Router      /evaluations/{id}/attempts [post]
func (h *Handler) StartAttempt(c *gin.Context) {
	evalID := c.Param("id")
	userID := middleware.UserIDFrom(c)

	model, err := h.svc.StartAttempt(c.Request.Context(), evalID, userID)
	if err != nil {
		h.renderAttemptError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToAttemptStart(model))
}

// GetAttempt godoc
// @Summary     Obtiene el estado de un intento
// @Description Retorna el intento con sus preguntas (sin correcta) y las respuestas actuales del estudiante.
// @Tags        attempts
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del intento"
// @Success     200 {object} dto.AttemptStateResponse
// @Failure     404 {object} httperr.Error "intento no encontrado o no pertenece al usuario"
// @Router      /attempts/{id} [get]
func (h *Handler) GetAttempt(c *gin.Context) {
	attemptID := c.Param("id")
	userID := middleware.UserIDFrom(c)

	state, err := h.svc.GetAttempt(c.Request.Context(), attemptID, userID)
	if err != nil {
		h.renderAttemptError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToAttemptState(state))
}

// SaveAnswer godoc
// @Summary     Guarda la respuesta de una pregunta
// @Description Registra o actualiza la respuesta del estudiante para una pregunta del intento.
// @Tags        attempts
// @Accept      json
// @Security    BearerAuth
// @Param       id   path   string              true "UUID del intento"
// @Param       body body   dto.AnswerRequest   true "Respuesta"
// @Success     204
// @Failure     400 {object} httperr.Error "opcion o pregunta invalida"
// @Failure     404 {object} httperr.Error "intento no encontrado"
// @Failure     409 {object} httperr.Error "intento ya finalizado"
// @Router      /attempts/{id}/answers [post]
func (h *Handler) SaveAnswer(c *gin.Context) {
	attemptID := c.Param("id")
	userID := middleware.UserIDFrom(c)

	var req dto.AnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	if err := h.svc.SaveAnswer(c.Request.Context(), attemptID, userID, req.QuestionID, req.OptionID); err != nil {
		h.renderAttemptError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// SubmitAttempt godoc
// @Summary     Finaliza y califica un intento
// @Description Cierra el intento, calcula el puntaje y retorna el resultado.
// @Tags        attempts
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del intento"
// @Success     200 {object} dto.SubmitResponse
// @Failure     404 {object} httperr.Error "intento no encontrado"
// @Failure     409 {object} httperr.Error "intento ya finalizado"
// @Router      /attempts/{id}/submit [post]
func (h *Handler) SubmitAttempt(c *gin.Context) {
	attemptID := c.Param("id")
	userID := middleware.UserIDFrom(c)

	result, err := h.svc.SubmitAttempt(c.Request.Context(), attemptID, userID)
	if err != nil {
		h.renderAttemptError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToSubmit(result))
}

// ── Error render helpers (TWO — deliberate, mirrors courses handler REQ-DIVERGENCE) ────

// renderAttemptError maps student attempt sentinels to HTTP status codes.
// This is a THIRD render helper — intentionally separate from renderReadError and
// renderWriteError (ADR-D). The student sentinel set is disjoint from the creator set.
func (h *Handler) renderAttemptError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAttemptNotFound), errors.Is(err, service.ErrEvaluationNotFound):
		httperr.Render(c, httperr.NotFound("ATTEMPT_NOT_FOUND", "attempt not found"))
	case errors.Is(err, service.ErrMaxAttemptsReached):
		httperr.Render(c, httperr.Conflict("MAX_ATTEMPTS_REACHED", "max attempts reached"))
	case errors.Is(err, service.ErrAttemptAlreadySubmitted):
		httperr.Render(c, httperr.Conflict("ATTEMPT_ALREADY_SUBMITTED", "attempt already submitted"))
	case errors.Is(err, service.ErrInvalidAnswer):
		httperr.Render(c, httperr.BadRequest("INVALID_ANSWER", "invalid answer"))
	default:
		slog.Error("evaluations: unexpected error (attempt)", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}

// renderReadError maps service sentinels to HTTP statuses for READ routes (GET).
// ErrNotOwner → 404 (hides existence of the evaluation from non-owners).
func (h *Handler) renderReadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEvaluationNotFound):
		httperr.Render(c, httperr.NotFound("EVALUATION_NOT_FOUND", "evaluation not found"))
	case errors.Is(err, service.ErrQuestionNotFound):
		httperr.Render(c, httperr.NotFound("QUESTION_NOT_FOUND", "question not found"))
	case errors.Is(err, service.ErrOptionNotFound):
		httperr.Render(c, httperr.NotFound("OPTION_NOT_FOUND", "option not found"))
	case errors.Is(err, service.ErrCourseNotFound):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found"))
	case errors.Is(err, service.ErrNotOwner):
		httperr.Render(c, httperr.NotFound("EVALUATION_NOT_FOUND", "evaluation not found")) // 404: hide existence
	case errors.Is(err, service.ErrCourseNotEditable):
		httperr.Render(c, httperr.Conflict("COURSE_NOT_EDITABLE", "course estado does not permit this edit"))
	default:
		slog.Error("evaluations: unexpected error (read)", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}

// renderWriteError maps service sentinels to HTTP statuses for WRITE routes (POST/PATCH/DELETE).
// ErrNotOwner → 403 (signals authz failure for write operations).
// The ONLY difference from renderReadError is the ErrNotOwner case.
func (h *Handler) renderWriteError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEvaluationNotFound):
		httperr.Render(c, httperr.NotFound("EVALUATION_NOT_FOUND", "evaluation not found"))
	case errors.Is(err, service.ErrQuestionNotFound):
		httperr.Render(c, httperr.NotFound("QUESTION_NOT_FOUND", "question not found"))
	case errors.Is(err, service.ErrOptionNotFound):
		httperr.Render(c, httperr.NotFound("OPTION_NOT_FOUND", "option not found"))
	case errors.Is(err, service.ErrCourseNotFound):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found"))
	case errors.Is(err, service.ErrNotOwner):
		httperr.Render(c, httperr.Forbidden("NOT_OWNER", "you do not own this course")) // 403: authz signal
	case errors.Is(err, service.ErrCourseNotEditable):
		httperr.Render(c, httperr.Conflict("COURSE_NOT_EDITABLE", "course estado does not permit this edit"))
	case errors.Is(err, service.ErrEvaluationExists):
		httperr.Render(c, httperr.Conflict("EVALUATION_EXISTS", "evaluation already exists for this course"))
	case errors.Is(err, service.ErrNoCorrectOption):
		httperr.Render(c, httperr.Conflict("NO_CORRECT_OPTION", "question has no correct option"))
	case errors.Is(err, service.ErrInvalidQuestionType):
		httperr.Render(c, httperr.BadRequest("INVALID_QUESTION_TYPE", "invalid question tipo"))
	default:
		slog.Error("evaluations: unexpected error (write)", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}
