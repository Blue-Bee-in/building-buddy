-- +goose Up
-- +goose StatementBegin

CREATE TABLE maintenance_requests (
  id               uuid           PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id       uuid           NOT NULL REFERENCES societies(id) ON DELETE CASCADE,
  title            text           NOT NULL,
  description      text,
  contractor_name  text           NOT NULL,
  quoted_amount    numeric(12,2)  NOT NULL CHECK (quoted_amount > 0),
  approved_amount  numeric(12,2),
  status           text           NOT NULL DEFAULT 'draft'
                                  CHECK (status IN ('draft','submitted','approved','rejected')),
  rejection_reason text,
  created_by       uuid           NOT NULL REFERENCES users(id),
  approved_by      uuid           REFERENCES users(id),
  created_at       timestamptz    NOT NULL DEFAULT now(),
  updated_at       timestamptz    NOT NULL DEFAULT now(),
  approved_at      timestamptz
);

CREATE INDEX idx_maintenance_requests_society ON maintenance_requests (society_id, created_at DESC);
CREATE INDEX idx_maintenance_requests_status  ON maintenance_requests (society_id, status);

CREATE TABLE maintenance_request_pdfs (
  id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  request_id    uuid        NOT NULL UNIQUE REFERENCES maintenance_requests(id) ON DELETE CASCADE,
  storage_path  text        NOT NULL,
  generated_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS maintenance_request_pdfs CASCADE;
DROP TABLE IF EXISTS maintenance_requests CASCADE;
-- +goose StatementEnd
