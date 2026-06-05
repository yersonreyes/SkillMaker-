-- Migration: 0006_add_evaluations (C3.1)
-- evaluation/question/question_option exercised now; attempt/answer SCHEMA-ONLY (C3.2 adds endpoints).
-- 1-1 course<->evaluation via UNIQUE(course_id). nota_minima is whole-% 0..100.

CREATE TABLE evaluation (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id    uuid        NOT NULL UNIQUE REFERENCES course(id) ON DELETE CASCADE,
    nota_minima  integer     NOT NULL DEFAULT 70
        CONSTRAINT ck_evaluation_nota CHECK (nota_minima BETWEEN 0 AND 100),
    intentos_max integer     NOT NULL DEFAULT 0
        CONSTRAINT ck_evaluation_intentos CHECK (intentos_max >= 0),
    created_at   timestamptz NOT NULL DEFAULT now()
);
-- UNIQUE(course_id) already creates an index; no extra idx needed.

CREATE TABLE question (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_id uuid        NOT NULL REFERENCES evaluation(id) ON DELETE CASCADE,
    enunciado     text        NOT NULL,
    tipo          text        NOT NULL
        CONSTRAINT ck_question_tipo CHECK (tipo IN ('opcion_multiple','verdadero_falso')),
    puntaje       integer     NOT NULL DEFAULT 1 CONSTRAINT ck_question_puntaje CHECK (puntaje > 0),
    orden         integer     NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_question_evaluation ON question(evaluation_id);

CREATE TABLE question_option (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id uuid        NOT NULL REFERENCES question(id) ON DELETE CASCADE,
    texto       text        NOT NULL,
    correcta    boolean     NOT NULL DEFAULT false,
    orden       integer     NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_question_option_question ON question_option(question_id);

-- SCHEMA-ONLY (C3.2) --
CREATE TABLE attempt (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid        NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    evaluation_id uuid        NOT NULL REFERENCES evaluation(id) ON DELETE CASCADE,
    numero        integer     NOT NULL DEFAULT 1,
    puntaje       integer     NOT NULL DEFAULT 0,
    aprobado      boolean     NOT NULL DEFAULT false,
    iniciado_en   timestamptz NOT NULL DEFAULT now(),
    finalizado_en timestamptz,
    CONSTRAINT uq_attempt_user_eval_num UNIQUE (user_id, evaluation_id, numero)
);
CREATE INDEX idx_attempt_user       ON attempt(user_id);
CREATE INDEX idx_attempt_evaluation ON attempt(evaluation_id);

CREATE TABLE answer (
    id          uuid    PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id  uuid    NOT NULL REFERENCES attempt(id) ON DELETE CASCADE,
    question_id uuid    NOT NULL REFERENCES question(id) ON DELETE RESTRICT,
    option_id   uuid    NOT NULL REFERENCES question_option(id) ON DELETE RESTRICT,
    correcta    boolean NOT NULL DEFAULT false
);
CREATE INDEX idx_answer_attempt  ON answer(attempt_id);
CREATE INDEX idx_answer_question ON answer(question_id);
