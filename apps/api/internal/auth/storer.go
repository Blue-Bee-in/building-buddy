package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	ab "github.com/volatiletech/authboss/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStorer implements:
//   - authboss.ServerStorer
//   - authboss.CreatingServerStorer  (register module)
//   - authboss.RecoveringServerStorer (recover module)
type PGStorer struct {
	db *pgxpool.Pool
}

func NewPGStorer(db *pgxpool.Pool) *PGStorer { return &PGStorer{db: db} }

// ── column list used in every SELECT ─────────────────────────────────────────
const userCols = `
	id, phone, name, email, avatar_url, is_super_admin, is_confirmed,
	password_hash, attempt_count, last_attempt, locked_until,
	recover_selector, recover_verifier, recover_expiry, otps,
	created_at, updated_at`

func scanUser(row pgx.Row) (*User, error) {
	u := &User{}
	var lastAttempt, lockedUntil, recoverExpiry *time.Time
	err := row.Scan(
		&u.ID, &u.Phone, &u.Name, &u.Email, &u.AvatarURL,
		&u.IsSuperAdmin, &u.IsConfirmed,
		&u.PasswordHash, &u.AttemptCount, &lastAttempt, &lockedUntil,
		&u.RecoverSelector, &u.RecoverVerifier, &recoverExpiry, &u.OTPs,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if lastAttempt != nil {
		u.LastAttempt = *lastAttempt
	}
	if lockedUntil != nil {
		u.LockedUntil = *lockedUntil
	}
	if recoverExpiry != nil {
		u.RecoverExpiry = *recoverExpiry
	}
	return u, nil
}

// ── authboss.ServerStorer ─────────────────────────────────────────────────────

// Load retrieves a user by PID (phone number).
func (s *PGStorer) Load(ctx context.Context, key string) (ab.User, error) {
	u, err := scanUser(s.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE phone = $1`, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ab.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Save updates an existing user's mutable fields.
func (s *PGStorer) Save(ctx context.Context, user ab.User) error {
	u := user.(*User)
	var lockedUntil *time.Time
	if !u.LockedUntil.IsZero() {
		lockedUntil = &u.LockedUntil
	}
	var lastAttempt *time.Time
	if !u.LastAttempt.IsZero() {
		lastAttempt = &u.LastAttempt
	}
	var recoverExpiry *time.Time
	if !u.RecoverExpiry.IsZero() {
		recoverExpiry = &u.RecoverExpiry
	}
	_, err := s.db.Exec(ctx, `
		UPDATE users SET
			name             = $2,
			email            = $3,
			password_hash    = $4,
			attempt_count    = $5,
			last_attempt     = $6,
			locked_until     = $7,
			recover_selector = $8,
			recover_verifier = $9,
			recover_expiry   = $10,
			otps             = $11,
			updated_at       = now()
		WHERE phone = $1`,
		u.Phone, u.Name, u.Email, u.PasswordHash,
		u.AttemptCount, lastAttempt, lockedUntil,
		u.RecoverSelector, u.RecoverVerifier, recoverExpiry,
		u.OTPs,
	)
	return err
}

// ── authboss.CreatingServerStorer ─────────────────────────────────────────────

// New returns a blank User for the register module to populate.
func (s *PGStorer) New(_ context.Context) ab.User { return &User{} }

// Create inserts a new user (called by the register module after New).
func (s *PGStorer) Create(ctx context.Context, user ab.User) error {
	u := user.(*User)
	return s.db.QueryRow(ctx, `
		INSERT INTO users (phone, name, email, password_hash, otps)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		u.Phone, u.Name, u.Email, u.PasswordHash, u.OTPs,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

// ── authboss.RecoveringServerStorer ──────────────────────────────────────────

// LoadByRecoverSelector looks up a user by their recovery selector token.
func (s *PGStorer) LoadByRecoverSelector(ctx context.Context, selector string) (ab.User, error) {
	u, err := scanUser(s.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE recover_selector = $1`, selector))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ab.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// SaveRecover persists only the recovery-specific fields.
func (s *PGStorer) SaveRecover(ctx context.Context, user ab.User) error {
	u := user.(*User)
	var recoverExpiry *time.Time
	if !u.RecoverExpiry.IsZero() {
		recoverExpiry = &u.RecoverExpiry
	}
	_, err := s.db.Exec(ctx, `
		UPDATE users SET
			recover_selector = $2,
			recover_verifier = $3,
			recover_expiry   = $4,
			updated_at       = now()
		WHERE phone = $1`,
		u.Phone, u.RecoverSelector, u.RecoverVerifier, recoverExpiry,
	)
	return err
}

// ── Helpers used by event hooks & custom handlers ─────────────────────────────

func (s *PGStorer) FindIDByPhone(ctx context.Context, phone string) (string, error) {
	var id string
	err := s.db.QueryRow(ctx,
		`SELECT id FROM users WHERE phone = $1`, phone).Scan(&id)
	return id, err
}

func (s *PGStorer) MarkConfirmed(ctx context.Context, phone string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE users SET is_confirmed = true, updated_at = now() WHERE phone = $1`, phone)
	return err
}

func (s *PGStorer) LoadMemberships(ctx context.Context, userID string) ([]FlatMembership, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			m.society_id,
			s.name          AS society_name,
			m.flat_id,
			f.unit          AS flat_unit,
			f.block         AS flat_block,
			array_agg(m.role ORDER BY m.role) AS roles,
			bool_or(m.is_primary)             AS is_primary
		FROM memberships m
		JOIN societies s ON s.id = m.society_id
		JOIN flats     f ON f.id = m.flat_id
		WHERE m.user_id = $1 AND m.is_active = true
		GROUP BY m.society_id, s.name, m.flat_id, f.unit, f.block
		ORDER BY is_primary DESC, s.name, f.block, f.unit`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FlatMembership
	for rows.Next() {
		var m FlatMembership
		if err := rows.Scan(
			&m.SocietyID, &m.SocietyName,
			&m.FlatID, &m.FlatUnit, &m.FlatBlock,
			&m.Roles, &m.IsPrimary,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ── Refresh token helpers ─────────────────────────────────────────────────────

// GenerateRefreshToken creates a 32-byte random hex token and returns both
// the raw value (sent to client) and its SHA-256 hash (stored in DB).
func GenerateRefreshToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	raw = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *PGStorer) SaveRefreshToken(ctx context.Context, userID, tokenHash, deviceType, app string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, device_type, app)
		VALUES ($1, $2, $3, $4)`,
		userID, tokenHash, deviceType, app,
	)
	return err
}

func (s *PGStorer) LoadRefreshToken(ctx context.Context, rawToken string) (userID string, err error) {
	h := hashToken(rawToken)
	err = s.db.QueryRow(ctx, `
		SELECT user_id FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`, h,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("refresh token invalid or expired")
	}
	return
}

func (s *PGStorer) RevokeRefreshToken(ctx context.Context, rawToken string) error {
	h := hashToken(rawToken)
	_, err := s.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1`, h)
	return err
}
