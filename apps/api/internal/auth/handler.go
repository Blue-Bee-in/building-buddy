// Package auth wires AuthBoss with JWT state, PostgreSQL storage, and JSON
// adapters for a phone+password / OTP mobile API.
//
// Routes provided after Setup():
//   POST   /register           — create account (phone + password)
//   POST   /login              — login with password
//   POST   /logout             — logout (revokes session)
//   GET    /recover            — start password recovery (sends SMS)
//   POST   /recover/end        — complete recovery with token + new password
//   POST   /otp/login          — login with AuthBoss-generated OTP
//   POST   /otp/send           — send phone-verification OTP (custom)
//   POST   /otp/verify         — verify phone OTP after signup (custom)
//   POST   /refresh            — exchange refresh token for new access token (custom)
//   PUT    /profile            — update name / email (requires auth)
//   POST   /invite/accept      — accept a flat invite (requires auth)
package auth
