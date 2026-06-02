-- Migration: 0002_add_supervisions
-- Description: supervisor↔employee relation. One supervisor has many employees;
--   an employee has AT MOST ONE supervisor (UNIQUE on empleado_id).
--   Self-supervision forbidden (CHECK). FKs cascade-delete with "user", mirroring user_role.

CREATE TABLE supervision (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    supervisor_id uuid        NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    empleado_id   uuid        NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    creado_en     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_supervision_empleado UNIQUE (empleado_id),
    CONSTRAINT ck_supervision_no_self  CHECK (supervisor_id <> empleado_id)
);

CREATE INDEX idx_supervision_supervisor ON supervision(supervisor_id);
