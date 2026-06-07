CREATE TABLE notification (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    tipo       text        NOT NULL CHECK (tipo IN ('curso_aprobado','curso_rechazado','certificado_emitido')),
    titulo     text        NOT NULL,
    cuerpo     text        NOT NULL DEFAULT '',
    leida      boolean     NOT NULL DEFAULT false,
    ref_id     uuid        NULL,
    creado_en  timestamptz NOT NULL DEFAULT now()
);

-- Partial index for fast unread count per user.
CREATE INDEX idx_notification_user_unread ON notification(user_id) WHERE leida = false;

-- Composite index for list query ordered by creado_en DESC.
CREATE INDEX idx_notification_user_created ON notification(user_id, creado_en DESC);
