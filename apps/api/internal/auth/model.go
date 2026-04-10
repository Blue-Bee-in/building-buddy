package auth

import "time"

// User implements every AuthBoss user interface needed by the
// auth, register, lock, recover, and otp modules.
type User struct {
	ID           string
	Phone        string  // PID — the primary identifier used by AuthBoss
	Name         string
	Email        *string // optional; used as delivery address for recover links
	AvatarURL    *string
	IsSuperAdmin bool
	IsConfirmed  bool // phone OTP verified after signup

	// auth module
	PasswordHash *string

	// lock module
	AttemptCount int
	LastAttempt  time.Time
	LockedUntil  time.Time // zero value = not locked

	// recover module
	RecoverSelector *string
	RecoverVerifier *string
	RecoverExpiry   time.Time

	// otp module — AuthBoss stores bcrypt-hashed OTPs as comma-separated list
	OTPs string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ── authboss.User ─────────────────────────────────────────────────────────────
func (u *User) GetPID() string     { return u.Phone }
func (u *User) PutPID(pid string)  { u.Phone = pid }

// ── authboss.AuthableUser ─────────────────────────────────────────────────────
func (u *User) GetPassword() string {
	if u.PasswordHash == nil {
		return ""
	}
	return *u.PasswordHash
}
func (u *User) PutPassword(hash string) { u.PasswordHash = &hash }

// ── authboss.LockableUser ─────────────────────────────────────────────────────
// GetLocked returns the time until which the account is locked.
// A zero time means the account is not locked.
func (u *User) GetAttemptCount() int          { return u.AttemptCount }
func (u *User) PutAttemptCount(n int)         { u.AttemptCount = n }
func (u *User) GetLastAttempt() time.Time     { return u.LastAttempt }
func (u *User) PutLastAttempt(t time.Time)    { u.LastAttempt = t }
func (u *User) GetLocked() time.Time          { return u.LockedUntil }
func (u *User) PutLocked(t time.Time)         { u.LockedUntil = t }

// ── authboss.RecoverableUser ──────────────────────────────────────────────────
// GetEmail falls back to Phone so the SMS mailer always has a destination.
func (u *User) GetEmail() string {
	if u.Email != nil && *u.Email != "" {
		return *u.Email
	}
	return u.Phone
}
func (u *User) PutEmail(e string) { u.Email = &e }

func (u *User) GetRecoverSelector() string {
	if u.RecoverSelector == nil {
		return ""
	}
	return *u.RecoverSelector
}
func (u *User) PutRecoverSelector(s string) { u.RecoverSelector = &s }

func (u *User) GetRecoverVerifier() string {
	if u.RecoverVerifier == nil {
		return ""
	}
	return *u.RecoverVerifier
}
func (u *User) PutRecoverVerifier(v string) { u.RecoverVerifier = &v }

func (u *User) GetRecoverExpiry() time.Time     { return u.RecoverExpiry }
func (u *User) PutRecoverExpiry(t time.Time)    { u.RecoverExpiry = t }

// ── authboss.OTPableUser (otp module) ─────────────────────────────────────────
func (u *User) GetOTPs() string      { return u.OTPs }
func (u *User) PutOTPs(otps string)  { u.OTPs = otps }

// ── DTO ───────────────────────────────────────────────────────────────────────

type UserDTO struct {
	ID          string  `json:"id"`
	Phone       string  `json:"phone"`
	Name        string  `json:"name"`
	Email       *string `json:"email,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	IsConfirmed bool    `json:"is_confirmed"`
}

func (u *User) ToDTO() UserDTO {
	return UserDTO{
		ID:          u.ID,
		Phone:       u.Phone,
		Name:        u.Name,
		Email:       u.Email,
		AvatarURL:   u.AvatarURL,
		IsConfirmed: u.IsConfirmed,
	}
}

// FlatMembership is returned on login so the client can build a society picker.
type FlatMembership struct {
	SocietyID   string   `json:"society_id"`
	SocietyName string   `json:"society_name"`
	FlatID      string   `json:"flat_id"`
	FlatUnit    string   `json:"flat_unit"`
	FlatBlock   *string  `json:"flat_block,omitempty"`
	Roles       []string `json:"roles"` // ["owner"] | ["resident"] | ["owner","resident"]
	IsPrimary   bool     `json:"is_primary"`
}
