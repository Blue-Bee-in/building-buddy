-- +goose Up
-- +goose StatementBegin

-- Secretary-only audit trail for maintenance request state transitions.
CREATE TABLE maintenance_request_audit_log (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  request_id  uuid        NOT NULL REFERENCES maintenance_requests(id) ON DELETE CASCADE,
  society_id  uuid        NOT NULL,
  actor_id    uuid        REFERENCES users(id),  -- NULL for system events (e.g. pdf_generated)
  action      text        NOT NULL CHECK (action IN ('created','submitted','approved','rejected','pdf_generated')),
  metadata    jsonb,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_maint_audit_request ON maintenance_request_audit_log (request_id, created_at DESC);
CREATE INDEX idx_maint_audit_society ON maintenance_request_audit_log (society_id, created_at DESC);

-- Expo push ticket IDs stored after send, polled 30+ min later to detect DeviceNotRegistered.
CREATE TABLE push_tickets (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  expo_id     text        NOT NULL,
  token       text        NOT NULL,
  notice_id   uuid        NOT NULL REFERENCES notices(id) ON DELETE CASCADE,
  created_at  timestamptz NOT NULL DEFAULT now(),
  polled_at   timestamptz
);
CREATE INDEX idx_push_tickets_pending ON push_tickets (created_at) WHERE polled_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS push_tickets CASCADE;
DROP TABLE IF EXISTS maintenance_request_audit_log CASCADE;
-- +goose StatementEnd
