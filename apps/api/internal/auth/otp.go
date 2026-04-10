package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

// generateOTP returns a 6-digit OTP string and its bcrypt hash.
func generateOTP() (otp, hash string, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return
	}
	otp = fmt.Sprintf("%06d", n.Int64())
	b, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	hash = string(b)
	return
}

// verifyOTPHash returns true if otp matches the stored bcrypt hash.
func verifyOTPHash(otp, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(otp)) == nil
}
