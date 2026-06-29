-- 000017_design_problem_progress.up.sql
-- Per-user progress on HLD design problems (FR-DESIGN-004). Mirrors
-- user_problem_progress (000011) so design problems track status, confidence,
-- attempts and time the same way DSA problems do — enabling them to feed the
-- dashboard/analytics and the revision engine later.
--
-- design_problem_id references design_problems(id) (000007). user_id keeps its
-- hard FK to users(id) for the user-owned cascade.

BEGIN;

CREATE TABLE IF NOT EXISTS user_design_problem_progress (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    design_problem_id   UUID NOT NULL,
    status              progress_status NOT NULL DEFAULT 'not_started',
    confidence          SMALLINT NULL CHECK (confidence BETWEEN 1 AND 5),
    attempts            INTEGER NOT NULL DEFAULT 0,
    time_spent_minutes  INTEGER NOT NULL DEFAULT 0,
    notes               TEXT NULL,
    last_attempt_at     TIMESTAMPTZ NULL,
    first_completed_at  TIMESTAMPTZ NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ NULL
);

-- One live progress row per (user, design problem).
CREATE UNIQUE INDEX IF NOT EXISTS uq_udpp_user_problem
    ON user_design_problem_progress (user_id, design_problem_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_udpp_user_status
    ON user_design_problem_progress (user_id, status);

DROP TRIGGER IF EXISTS trg_udpp_updated_at ON user_design_problem_progress;
CREATE TRIGGER trg_udpp_updated_at
    BEFORE UPDATE ON user_design_problem_progress
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
