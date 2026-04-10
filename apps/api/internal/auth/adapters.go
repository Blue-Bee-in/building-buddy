package auth

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	ab "github.com/volatiletech/authboss/v3"
)

// ── JSON Renderer ─────────────────────────────────────────────────────────────
// Satisfies authboss.Renderer — renders AuthBoss template data as JSON.

type JSONRenderer struct{}

func (JSONRenderer) Load(_ ...string) error { return nil }

func (JSONRenderer) Render(_ context.Context, _ string, data ab.HTMLData) ([]byte, string, error) {
	b, err := json.Marshal(data)
	return b, "application/json", err
}

// ── JSON Responder ────────────────────────────────────────────────────────────
// Satisfies authboss.Responder — sends an HTTP response with JSON body.

type JSONResponder struct{}

func (JSONResponder) Respond(w http.ResponseWriter, _ *http.Request, code int, _ string, data ab.HTMLData) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(data)
}

// ── JSON Redirector ───────────────────────────────────────────────────────────
// Satisfies authboss.Redirector — mobile clients don't follow HTTP redirects,
// so we return a JSON body with a "redirect" hint instead.

type JSONRedirector struct{}

func (JSONRedirector) Redirect(w http.ResponseWriter, _ *http.Request, ro ab.RedirectOptions) error {
	w.Header().Set("Content-Type", "application/json")
	code := ro.Code
	if code == 0 {
		code = http.StatusOK
	}
	w.WriteHeader(code)

	body := map[string]string{"redirect": ro.RedirectPath}
	if ro.Success != "" {
		body["message"] = ro.Success
	}
	if ro.Failure != "" {
		body["error"] = ro.Failure
	}
	return json.NewEncoder(w).Encode(body)
}

// ── JSON BodyReader ───────────────────────────────────────────────────────────
// Satisfies authboss.BodyReader — decodes JSON request bodies and exposes
// all fields that AuthBoss modules expect (PID, password, token, OTP…).

type JSONBodyReader struct{}

func (JSONBodyReader) Read(_ string, r *http.Request) (ab.Validator, error) {
	var body struct {
		Phone           string `json:"phone"`
		Password        string `json:"password"`
		PasswordConfirm string `json:"password_confirm"`
		Token           string `json:"token"`  // recover/end token
		OTP             string `json:"otp"`    // otp module login
		Name            string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		return nil, err
	}
	return &jsonVals{
		pid:             body.Phone,
		password:        body.Password,
		passwordConfirm: body.PasswordConfirm,
		token:           body.Token,
		otp:             body.OTP,
		name:            body.Name,
	}, nil
}

// jsonVals satisfies every AuthBoss valuer interface used across all modules.
type jsonVals struct {
	pid, password, passwordConfirm, token, otp, name string
}

func (v *jsonVals) GetPID() string             { return v.pid }
func (v *jsonVals) GetPassword() string        { return v.password }
func (v *jsonVals) GetPasswordConfirm() string { return v.passwordConfirm }
func (v *jsonVals) GetToken() string           { return v.token } // recover/end
func (v *jsonVals) GetOTP() string             { return v.otp }
func (v *jsonVals) Validate() []error          { return nil }

// ── SMS / WhatsApp Mailer ─────────────────────────────────────────────────────
// Satisfies authboss.Mailer — AuthBoss calls this for recover links.
// In production, integrate Twilio / Gupshup here.

type SMSMailer struct{}

func (SMSMailer) Send(_ context.Context, mail ab.Email) error {
	// TODO: send via WhatsApp / SMS
	// mail.To[0] is the phone number (or email if set)
	// mail.TextBody contains the recovery link / OTP
	to := ""
	if len(mail.To) > 0 {
		to = mail.To[0]
	}
	slog.Info("mailer stub — would send SMS/WhatsApp",
		"to", to,
		"subject", mail.Subject,
		"body", strings.TrimSpace(mail.TextBody),
	)
	return nil
}

// ── chi-compatible AuthBoss Router ────────────────────────────────────────────
// authboss.Router is an interface; we wrap chi so AuthBoss can register routes
// on our existing chi router tree.

type chiABRouter struct {
	mux interface {
		Get(pattern string, handlerFn http.HandlerFunc)
		Post(pattern string, handlerFn http.HandlerFunc)
		Delete(pattern string, handlerFn http.HandlerFunc)
		ServeHTTP(http.ResponseWriter, *http.Request)
	}
}

func (c *chiABRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}
func (c *chiABRouter) Get(path string, h http.Handler) {
	c.mux.Get(path, h.ServeHTTP)
}
func (c *chiABRouter) Post(path string, h http.Handler) {
	c.mux.Post(path, h.ServeHTTP)
}
func (c *chiABRouter) Delete(path string, h http.Handler) {
	c.mux.Delete(path, h.ServeHTTP)
}

// ── slog Logger ───────────────────────────────────────────────────────────────
// Satisfies authboss.Logger.

type SlogLogger struct{}

func (SlogLogger) Info(msg string)  { slog.Info(msg) }
func (SlogLogger) Error(msg string) { slog.Error(msg) }
