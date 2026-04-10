-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
-- ENUMS
-- ============================================================
CREATE TYPE society_status      AS ENUM ('draft', 'active', 'suspended');
CREATE TYPE staff_role          AS ENUM ('admin', 'secretary', 'treasurer', 'committee', 'guard');
CREATE TYPE membership_role     AS ENUM ('owner', 'resident');
CREATE TYPE charge_frequency    AS ENUM ('monthly', 'quarterly', 'annually', 'one_time');
CREATE TYPE invoice_status      AS ENUM ('draft', 'sent', 'paid', 'partially_paid', 'overdue', 'waived');
CREATE TYPE payment_method      AS ENUM ('razorpay', 'cash', 'cheque', 'bank_transfer', 'upi_manual');
CREATE TYPE payment_status      AS ENUM ('pending', 'success', 'failed', 'refunded');
CREATE TYPE pass_type           AS ENUM ('visitor', 'delivery', 'frequent_visitor');
CREATE TYPE pass_status         AS ENUM ('active', 'used', 'expired', 'cancelled');
CREATE TYPE entry_type          AS ENUM ('visitor', 'walk_in', 'delivery', 'domestic_help', 'vehicle');
CREATE TYPE domestic_help_type  AS ENUM ('maid', 'cook', 'driver', 'nanny', 'other');
CREATE TYPE vehicle_type        AS ENUM ('2_wheeler', '4_wheeler', 'other');
CREATE TYPE complaint_status    AS ENUM ('open', 'in_progress', 'resolved', 'closed', 'reopened');
CREATE TYPE complaint_category  AS ENUM ('maintenance', 'security', 'billing', 'noise', 'cleanliness', 'other');
CREATE TYPE notice_category     AS ENUM ('general', 'maintenance', 'event', 'emergency');
CREATE TYPE sos_status          AS ENUM ('active', 'cancelled', 'resolved');
CREATE TYPE kyc_status          AS ENUM ('pending', 'under_review', 'approved', 'rejected');

-- ============================================================
-- USERS & AUTH
-- ============================================================

CREATE TABLE users (
  id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  phone           text        NOT NULL UNIQUE,
  name            text        NOT NULL,
  email           text        UNIQUE,
  avatar_url      text,
  is_super_admin  boolean     NOT NULL DEFAULT false,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);

-- OTP records for phone-based login
CREATE TABLE otps (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  phone       text        NOT NULL,
  otp_hash    text        NOT NULL,        -- bcrypt hash of the 6-digit OTP
  expires_at  timestamptz NOT NULL,
  used_at     timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_otps_phone ON otps (phone);

-- Device push tokens (one user can have multiple devices)
CREATE TABLE fcm_tokens (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token       text        NOT NULL UNIQUE,
  device_type text        NOT NULL CHECK (device_type IN ('ios', 'android')),
  app         text        NOT NULL CHECK (app IN ('resident', 'guard')),
  created_at  timestamptz NOT NULL DEFAULT now(),
  last_seen   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_fcm_tokens_user ON fcm_tokens (user_id);

-- ============================================================
-- SOCIETIES
-- ============================================================

CREATE TABLE societies (
  id               uuid           PRIMARY KEY DEFAULT gen_random_uuid(),
  name             text           NOT NULL,
  address          text           NOT NULL,
  city             text           NOT NULL,
  state            text           NOT NULL,
  pincode          text           NOT NULL,
  logo_url         text,
  status           society_status NOT NULL DEFAULT 'draft',
  billing_day      int            NOT NULL DEFAULT 1 CHECK (billing_day BETWEEN 1 AND 28),
  grace_days       int            NOT NULL DEFAULT 7,   -- days after due_date before penalty kicks in
  penalty_percent  numeric(5,2)   NOT NULL DEFAULT 0,   -- % charged on overdue amount per month
  created_at       timestamptz    NOT NULL DEFAULT now(),
  updated_at       timestamptz    NOT NULL DEFAULT now()
);

-- Razorpay Route — each society has its own linked bank account
CREATE TABLE society_bank_details (
  id                           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id                   uuid        NOT NULL UNIQUE REFERENCES societies(id) ON DELETE CASCADE,
  account_name                 text        NOT NULL,
  account_number               text        NOT NULL,
  ifsc                         text        NOT NULL,
  bank_name                    text        NOT NULL,
  razorpay_linked_account_id   text,                   -- populated after KYC approval
  kyc_status                   kyc_status  NOT NULL DEFAULT 'pending',
  kyc_rejection_reason         text,
  created_at                   timestamptz NOT NULL DEFAULT now(),
  updated_at                   timestamptz NOT NULL DEFAULT now()
);

-- Society-level roles: admin/committee members/guards
-- Distinct from flat-level memberships — a person can hold both
CREATE TABLE society_staff (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  society_id  uuid        NOT NULL REFERENCES societies(id) ON DELETE CASCADE,
  role        staff_role  NOT NULL,
  gate_name   text,                                    -- for guards: which gate they cover
  is_active   boolean     NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, society_id, role)
);
CREATE INDEX idx_society_staff_society ON society_staff (society_id);
CREATE INDEX idx_society_staff_user    ON society_staff (user_id);

-- ============================================================
-- FLATS & MEMBERSHIPS
-- ============================================================

CREATE TABLE flats (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id  uuid        NOT NULL REFERENCES societies(id) ON DELETE CASCADE,
  block       text,
  floor       int,
  unit        text        NOT NULL,
  type        text,                                    -- '1BHK', '2BHK', 'Studio', 'Villa', etc.
  area_sqft   numeric(8,2),
  created_at  timestamptz NOT NULL DEFAULT now(),
  UNIQUE (society_id, block, unit)
);
CREATE INDEX idx_flats_society ON flats (society_id);

-- A user can hold owner AND/OR resident role in the same or different flats
-- e.g. owner of H1, resident of H2, owner+resident of H3
CREATE TABLE memberships (
  id            uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       uuid             NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  society_id    uuid             NOT NULL REFERENCES societies(id),  -- denormalised for fast queries
  flat_id       uuid             NOT NULL REFERENCES flats(id) ON DELETE CASCADE,
  role          membership_role  NOT NULL,
  is_primary    boolean          NOT NULL DEFAULT false,  -- primary context when logging in
  move_in_date  date,
  move_out_date date,
  is_active     boolean          NOT NULL DEFAULT true,
  invited_by    uuid             REFERENCES users(id),
  created_at    timestamptz      NOT NULL DEFAULT now(),
  UNIQUE (user_id, flat_id, role)
);
CREATE INDEX idx_memberships_user    ON memberships (user_id);
CREATE INDEX idx_memberships_flat    ON memberships (flat_id);
CREATE INDEX idx_memberships_society ON memberships (society_id);

-- One-time invite links sent via WhatsApp/SMS for resident onboarding
CREATE TABLE invites (
  id          uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id  uuid             NOT NULL REFERENCES societies(id),
  flat_id     uuid             NOT NULL REFERENCES flats(id),
  role        membership_role  NOT NULL,
  phone       text             NOT NULL,
  name        text             NOT NULL,
  email       text,
  token       uuid             NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  invited_by  uuid             NOT NULL REFERENCES users(id),
  expires_at  timestamptz      NOT NULL DEFAULT now() + interval '7 days',
  accepted_at timestamptz,
  created_at  timestamptz      NOT NULL DEFAULT now()
);
CREATE INDEX idx_invites_token ON invites (token);
CREATE INDEX idx_invites_flat  ON invites (flat_id);

-- ============================================================
-- VEHICLES
-- ============================================================

CREATE TABLE vehicles (
  id            uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id    uuid         NOT NULL REFERENCES societies(id),
  flat_id       uuid         NOT NULL REFERENCES flats(id) ON DELETE CASCADE,
  owner_id      uuid         NOT NULL REFERENCES users(id),
  number_plate  text         NOT NULL,
  type          vehicle_type NOT NULL,
  make          text,
  model         text,
  color         text,
  is_active     boolean      NOT NULL DEFAULT true,
  created_at    timestamptz  NOT NULL DEFAULT now(),
  UNIQUE (society_id, number_plate)
);
CREATE INDEX idx_vehicles_flat  ON vehicles (flat_id);

-- ============================================================
-- DOMESTIC HELP
-- ============================================================

CREATE TABLE domestic_help (
  id              uuid               PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id      uuid               NOT NULL REFERENCES societies(id),
  flat_id         uuid               NOT NULL REFERENCES flats(id) ON DELETE CASCADE,
  registered_by   uuid               NOT NULL REFERENCES users(id),
  name            text               NOT NULL,
  phone           text,
  photo_url       text,
  id_proof_url    text,
  type            domestic_help_type NOT NULL,
  working_days    int[]              NOT NULL DEFAULT '{1,2,3,4,5}',  -- 0=Sun … 6=Sat
  entry_time      time,
  exit_time       time,
  is_active       boolean            NOT NULL DEFAULT true,
  is_blacklisted  boolean            NOT NULL DEFAULT false,
  blacklist_note  text,
  created_at      timestamptz        NOT NULL DEFAULT now(),
  updated_at      timestamptz        NOT NULL DEFAULT now()
);
CREATE INDEX idx_domestic_help_flat    ON domestic_help (flat_id);
CREATE INDEX idx_domestic_help_society ON domestic_help (society_id);

-- Society-wide blacklist — applies across all flats regardless of registration
CREATE TABLE domestic_help_blacklist (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id  uuid        NOT NULL REFERENCES societies(id),
  phone       text        NOT NULL,
  name        text        NOT NULL,
  reason      text        NOT NULL,
  blocked_by  uuid        NOT NULL REFERENCES users(id),
  created_at  timestamptz NOT NULL DEFAULT now(),
  UNIQUE (society_id, phone)
);

-- ============================================================
-- BILLING
-- ============================================================

-- Reusable billing components per society (maintenance, sinking fund, water, parking, etc.)
CREATE TABLE charge_heads (
  id            uuid              PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id    uuid              NOT NULL REFERENCES societies(id) ON DELETE CASCADE,
  name          text              NOT NULL,
  description   text,
  amount        numeric(10,2)     NOT NULL,
  frequency     charge_frequency  NOT NULL,
  is_active     boolean           NOT NULL DEFAULT true,
  applies_from  date              NOT NULL DEFAULT current_date,
  created_at    timestamptz       NOT NULL DEFAULT now(),
  updated_at    timestamptz       NOT NULL DEFAULT now()
);
CREATE INDEX idx_charge_heads_society ON charge_heads (society_id);

-- Override default charge amount for a specific flat (e.g. commercial flat, extra parking)
CREATE TABLE flat_charge_overrides (
  id              uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
  flat_id         uuid          NOT NULL REFERENCES flats(id) ON DELETE CASCADE,
  charge_head_id  uuid          NOT NULL REFERENCES charge_heads(id) ON DELETE CASCADE,
  amount          numeric(10,2) NOT NULL,
  reason          text,
  created_at      timestamptz   NOT NULL DEFAULT now(),
  UNIQUE (flat_id, charge_head_id)
);

CREATE TABLE invoices (
  id                 uuid           PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id         uuid           NOT NULL REFERENCES societies(id),
  flat_id            uuid           NOT NULL REFERENCES flats(id),
  period             text           NOT NULL,               -- 'YYYY-MM'
  total_amount       numeric(10,2)  NOT NULL,
  paid_amount        numeric(10,2)  NOT NULL DEFAULT 0,
  status             invoice_status NOT NULL DEFAULT 'draft',
  due_date           date           NOT NULL,
  pdf_url            text,
  razorpay_order_id  text           UNIQUE,
  paid_at            timestamptz,
  created_at         timestamptz    NOT NULL DEFAULT now(),
  updated_at         timestamptz    NOT NULL DEFAULT now(),
  UNIQUE (flat_id, period)
);
CREATE INDEX idx_invoices_flat    ON invoices (flat_id);
CREATE INDEX idx_invoices_society ON invoices (society_id);
CREATE INDEX idx_invoices_status  ON invoices (status);

CREATE TABLE invoice_line_items (
  id              uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
  invoice_id      uuid          NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
  charge_head_id  uuid          REFERENCES charge_heads(id),   -- nullable for penalty/misc lines
  description     text          NOT NULL,
  amount          numeric(10,2) NOT NULL
);
CREATE INDEX idx_line_items_invoice ON invoice_line_items (invoice_id);

CREATE TABLE payments (
  id                    uuid           PRIMARY KEY DEFAULT gen_random_uuid(),
  invoice_id            uuid           NOT NULL REFERENCES invoices(id),
  society_id            uuid           NOT NULL REFERENCES societies(id),
  flat_id               uuid           NOT NULL REFERENCES flats(id),
  amount                numeric(10,2)  NOT NULL,
  method                payment_method NOT NULL,
  status                payment_status NOT NULL DEFAULT 'pending',
  razorpay_payment_id   text           UNIQUE,
  razorpay_order_id     text,
  recorded_by           uuid           REFERENCES users(id),   -- admin user for manual payments
  notes                 text,
  paid_at               timestamptz,
  created_at            timestamptz    NOT NULL DEFAULT now()
);
CREATE INDEX idx_payments_invoice ON payments (invoice_id);
CREATE INDEX idx_payments_society ON payments (society_id);

-- Full webhook audit trail; kept even for failed/suspicious events
CREATE TABLE razorpay_events (
  id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id      text        NOT NULL UNIQUE,   -- Razorpay's x-razorpay-event-id header
  event_type    text        NOT NULL,
  payload       jsonb       NOT NULL,
  is_verified   boolean     NOT NULL DEFAULT false,
  processed_at  timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now()
);

-- ============================================================
-- VISITORS & GATE
-- ============================================================

-- QR/OTP passes created by residents for expected visitors and deliveries
CREATE TABLE visitor_passes (
  id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id      uuid        NOT NULL REFERENCES societies(id),
  flat_id         uuid        NOT NULL REFERENCES flats(id),
  created_by      uuid        NOT NULL REFERENCES users(id),
  visitor_name    text        NOT NULL,
  visitor_phone   text,
  purpose         text,
  type            pass_type   NOT NULL DEFAULT 'visitor',
  token           text        NOT NULL UNIQUE DEFAULT encode(gen_random_bytes(12), 'hex'),
  otp             text,                                    -- 6-digit, stored plain (short-lived)
  valid_from      timestamptz NOT NULL DEFAULT now(),
  valid_until     timestamptz NOT NULL,
  status          pass_status NOT NULL DEFAULT 'active',
  created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_visitor_passes_flat    ON visitor_passes (flat_id);
CREATE INDEX idx_visitor_passes_token   ON visitor_passes (token);
CREATE INDEX idx_visitor_passes_society ON visitor_passes (society_id);

-- All entry/exit events: visitors, walk-ins, deliveries, domestic help, vehicles
CREATE TABLE entry_log (
  id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id        uuid        NOT NULL REFERENCES societies(id),
  flat_id           uuid        REFERENCES flats(id),           -- destination flat
  entry_type        entry_type  NOT NULL,
  pass_id           uuid        REFERENCES visitor_passes(id),  -- set for pre-approved entries
  domestic_help_id  uuid        REFERENCES domestic_help(id),   -- set for domestic help entries
  vehicle_id        uuid        REFERENCES vehicles(id),        -- set for vehicle entries
  person_name       text        NOT NULL,
  person_phone      text,
  vehicle_number    text,
  photo_url         text,
  entry_at          timestamptz NOT NULL DEFAULT now(),
  exit_at           timestamptz,
  guard_id          uuid        REFERENCES users(id),
  approved_by       uuid        REFERENCES users(id),           -- resident who approved a walk-in
  notes             text,
  created_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_entry_log_society  ON entry_log (society_id);
CREATE INDEX idx_entry_log_flat     ON entry_log (flat_id);
CREATE INDEX idx_entry_log_entry_at ON entry_log (entry_at DESC);

-- ============================================================
-- HELPDESK & NOTICES
-- ============================================================

CREATE TABLE complaints (
  id            uuid               PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id    uuid               NOT NULL REFERENCES societies(id),
  flat_id       uuid               NOT NULL REFERENCES flats(id),
  raised_by     uuid               NOT NULL REFERENCES users(id),
  category      complaint_category NOT NULL,
  title         text               NOT NULL,
  description   text,
  photo_urls    text[]             NOT NULL DEFAULT '{}',
  status        complaint_status   NOT NULL DEFAULT 'open',
  assigned_to   uuid               REFERENCES users(id),
  sla_hours     int                NOT NULL DEFAULT 48,
  due_at        timestamptz,
  escalated_at  timestamptz,
  resolved_at   timestamptz,
  closed_at     timestamptz,
  parent_id     uuid               REFERENCES complaints(id),   -- link recurring/related issues
  created_at    timestamptz        NOT NULL DEFAULT now(),
  updated_at    timestamptz        NOT NULL DEFAULT now()
);
CREATE INDEX idx_complaints_society ON complaints (society_id);
CREATE INDEX idx_complaints_flat    ON complaints (flat_id);
CREATE INDEX idx_complaints_status  ON complaints (status);

CREATE TABLE complaint_comments (
  id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  complaint_id  uuid        NOT NULL REFERENCES complaints(id) ON DELETE CASCADE,
  author_id     uuid        NOT NULL REFERENCES users(id),
  body          text        NOT NULL,
  photo_urls    text[]      NOT NULL DEFAULT '{}',
  created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_complaint_comments_complaint ON complaint_comments (complaint_id);

-- Society announcements (pinned notices, event announcements, emergency broadcasts)
CREATE TABLE notices (
  id            uuid            PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id    uuid            NOT NULL REFERENCES societies(id) ON DELETE CASCADE,
  created_by    uuid            NOT NULL REFERENCES users(id),
  title         text            NOT NULL,
  body          text            NOT NULL,
  category      notice_category NOT NULL DEFAULT 'general',
  target_flat   uuid            REFERENCES flats(id),     -- null = broadcast to all residents
  is_pinned     boolean         NOT NULL DEFAULT false,
  published_at  timestamptz,
  expires_at    timestamptz,
  created_at    timestamptz     NOT NULL DEFAULT now()
);
CREATE INDEX idx_notices_society ON notices (society_id);

-- ============================================================
-- EMERGENCY SOS
-- ============================================================

CREATE TABLE sos_alerts (
  id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  society_id     uuid        NOT NULL REFERENCES societies(id),
  flat_id        uuid        NOT NULL REFERENCES flats(id),
  raised_by      uuid        NOT NULL REFERENCES users(id),
  status         sos_status  NOT NULL DEFAULT 'active',
  latitude       numeric(10,7),
  longitude      numeric(10,7),
  cancelled_at   timestamptz,
  resolved_at    timestamptz,
  response_time  interval,   -- resolved_at - created_at; stored for analytics
  notes          text,
  created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_sos_society ON sos_alerts (society_id, created_at DESC);

-- ============================================================
-- NOTIFICATIONS
-- ============================================================

CREATE TABLE notifications (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  society_id  uuid        REFERENCES societies(id),
  title       text        NOT NULL,
  body        text        NOT NULL,
  type        text        NOT NULL,   -- 'visitor_entry', 'invoice_due', 'complaint_update', 'sos', 'notice'
  ref_id      uuid,                   -- FK to the related entity (polymorphic, no constraint)
  is_read     boolean     NOT NULL DEFAULT false,
  read_at     timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_notifications_user ON notifications (user_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS sos_alerts CASCADE;
DROP TABLE IF EXISTS complaint_comments CASCADE;
DROP TABLE IF EXISTS complaints CASCADE;
DROP TABLE IF EXISTS notices CASCADE;
DROP TABLE IF EXISTS entry_log CASCADE;
DROP TABLE IF EXISTS visitor_passes CASCADE;
DROP TABLE IF EXISTS razorpay_events CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS invoice_line_items CASCADE;
DROP TABLE IF EXISTS invoices CASCADE;
DROP TABLE IF EXISTS flat_charge_overrides CASCADE;
DROP TABLE IF EXISTS charge_heads CASCADE;
DROP TABLE IF EXISTS domestic_help_blacklist CASCADE;
DROP TABLE IF EXISTS domestic_help CASCADE;
DROP TABLE IF EXISTS vehicles CASCADE;
DROP TABLE IF EXISTS invites CASCADE;
DROP TABLE IF EXISTS memberships CASCADE;
DROP TABLE IF EXISTS flats CASCADE;
DROP TABLE IF EXISTS society_staff CASCADE;
DROP TABLE IF EXISTS society_bank_details CASCADE;
DROP TABLE IF EXISTS societies CASCADE;
DROP TABLE IF EXISTS fcm_tokens CASCADE;
DROP TABLE IF EXISTS otps CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TYPE IF EXISTS sos_status CASCADE;
DROP TYPE IF EXISTS notice_category CASCADE;
DROP TYPE IF EXISTS complaint_category CASCADE;
DROP TYPE IF EXISTS complaint_status CASCADE;
DROP TYPE IF EXISTS vehicle_type CASCADE;
DROP TYPE IF EXISTS domestic_help_type CASCADE;
DROP TYPE IF EXISTS entry_type CASCADE;
DROP TYPE IF EXISTS pass_status CASCADE;
DROP TYPE IF EXISTS pass_type CASCADE;
DROP TYPE IF EXISTS payment_status CASCADE;
DROP TYPE IF EXISTS payment_method CASCADE;
DROP TYPE IF EXISTS invoice_status CASCADE;
DROP TYPE IF EXISTS charge_frequency CASCADE;
DROP TYPE IF EXISTS membership_role CASCADE;
DROP TYPE IF EXISTS staff_role CASCADE;
DROP TYPE IF EXISTS kyc_status CASCADE;
DROP TYPE IF EXISTS society_status CASCADE;
-- +goose StatementEnd
