CREATE TABLE video_progress (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid        NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    video_id        uuid        NOT NULL REFERENCES video(id)   ON DELETE CASCADE,
    completado      boolean     NOT NULL DEFAULT false,
    last_position_s integer     NOT NULL DEFAULT 0,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_video_progress_user_video UNIQUE (user_id, video_id)
);
CREATE INDEX idx_video_progress_user  ON video_progress(user_id);
CREATE INDEX idx_video_progress_video ON video_progress(video_id);
