package maintenance_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Blue-Bee-in/building-buddy/internal/maintenance"
	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

// setupDB connects to TEST_DATABASE_URL; skips the test if the variable is unset.
func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration tests")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// injectAuth returns a chi middleware that injects rc without JWT validation.
func injectAuth(rc *apimw.RequestContext) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := apimw.WithRequestContext(r.Context(), rc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// testRouter builds a maintenance router with a fake auth middleware.
func testRouter(t *testing.T, pool *pgxpool.Pool, rc *apimw.RequestContext) http.Handler {
	t.Helper()
	rc2, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatalf("river client: %v", err)
	}
	return maintenance.RouterForTest(pool, rc2, nil, injectAuth(rc))
}

// ── DB seed helpers ───────────────────────────────────────────────────────────

func insertTestSociety(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO societies (name, address, city, state, pincode, country)
		VALUES ($1,'Test Addr','City','State','123456','India')
		RETURNING id`,
		fmt.Sprintf("TestSociety-%d", time.Now().UnixNano()),
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert society: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM societies WHERE id=$1`, id)
	})
	return id
}

func insertTestUser(t *testing.T, pool *pgxpool.Pool, societyID string) string {
	t.Helper()
	var userID string
	email := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (email, name, password_hash) VALUES ($1, 'Test User', 'x') RETURNING id`,
		email,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	_, err = pool.Exec(context.Background(), `
		INSERT INTO memberships (user_id, society_id, role, is_active) VALUES ($1, $2, 'resident', true)`,
		userID, societyID,
	)
	if err != nil {
		t.Fatalf("insert membership: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
	})
	return userID
}

func insertTestRequest(t *testing.T, pool *pgxpool.Pool, societyID, userID string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO maintenance_requests (society_id, title, contractor_name, quoted_amount, created_by)
		VALUES ($1, 'Fix Leak', 'Plumber Co', 5000, $2) RETURNING id`,
		societyID, userID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert request: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM maintenance_requests WHERE id=$1`, id)
	})
	return id
}

// ── Cross-society security tests (E13) ───────────────────────────────────────

// TestServePDF_CrossSociety: user in society A cannot fetch a PDF from society B.
func TestServePDF_CrossSociety(t *testing.T) {
	pool := setupDB(t)
	societyA := insertTestSociety(t, pool)
	societyB := insertTestSociety(t, pool)
	userA := insertTestUser(t, pool, societyA)
	userB := insertTestUser(t, pool, societyB)
	requestB := insertTestRequest(t, pool, societyB, userB)

	// Insert a fake PDF record so the PDF endpoint has something to find.
	pool.Exec(context.Background(),
		`INSERT INTO maintenance_request_pdfs (request_id, storage_path) VALUES ($1, '/tmp/fake.pdf')`,
		requestB)

	rcA := &apimw.RequestContext{UserID: userA, SocietyID: societyA}
	req := httptest.NewRequest(http.MethodGet, "/"+requestB+"/pdf", nil)
	rr := httptest.NewRecorder()
	testRouter(t, pool, rcA).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-society PDF access, got %d", rr.Code)
	}
}

// TestGetByID_CrossSociety: user in society A cannot retrieve a request from society B.
func TestGetByID_CrossSociety(t *testing.T) {
	pool := setupDB(t)
	societyA := insertTestSociety(t, pool)
	societyB := insertTestSociety(t, pool)
	userA := insertTestUser(t, pool, societyA)
	userB := insertTestUser(t, pool, societyB)
	requestB := insertTestRequest(t, pool, societyB, userB)

	rcA := &apimw.RequestContext{UserID: userA, SocietyID: societyA}
	req := httptest.NewRequest(http.MethodGet, "/"+requestB, nil)
	rr := httptest.NewRecorder()
	testRouter(t, pool, rcA).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-society get, got %d", rr.Code)
	}
}

// TestApprove_CrossSociety: admin in society A cannot approve a request from society B.
func TestApprove_CrossSociety(t *testing.T) {
	pool := setupDB(t)
	societyA := insertTestSociety(t, pool)
	societyB := insertTestSociety(t, pool)
	userA := insertTestUser(t, pool, societyA)
	userB := insertTestUser(t, pool, societyB)
	requestB := insertTestRequest(t, pool, societyB, userB)

	// Advance to submitted so approve would succeed for a same-society admin.
	pool.Exec(context.Background(),
		`UPDATE maintenance_requests SET status='submitted' WHERE id=$1`, requestB)

	rcA := &apimw.RequestContext{UserID: userA, SocietyID: societyA, IsAdmin: true}
	body := strings.NewReader(`{"approved_amount": 5000}`)
	req := httptest.NewRequest(http.MethodPost, "/"+requestB+"/approve", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	testRouter(t, pool, rcA).ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Errorf("expected non-200 for cross-society approve, got %d", rr.Code)
	}

	// Verify the database row was NOT changed.
	var status string
	pool.QueryRow(context.Background(),
		`SELECT status FROM maintenance_requests WHERE id=$1`, requestB).Scan(&status)
	if status != "submitted" {
		t.Errorf("request status mutated to %q — cross-society approve succeeded", status)
	}
}

// TestList_SocietyScope: list only returns requests for the caller's society.
func TestList_SocietyScope(t *testing.T) {
	pool := setupDB(t)
	societyA := insertTestSociety(t, pool)
	societyB := insertTestSociety(t, pool)
	userA := insertTestUser(t, pool, societyA)
	userB := insertTestUser(t, pool, societyB)
	requestA := insertTestRequest(t, pool, societyA, userA)
	requestB := insertTestRequest(t, pool, societyB, userB)

	rcA := &apimw.RequestContext{UserID: userA, SocietyID: societyA}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	testRouter(t, pool, rcA).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("list returned %d", rr.Code)
	}

	var results []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&results); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	for _, r := range results {
		if id, _ := r["id"].(string); id == requestB {
			t.Errorf("society B request %s appeared in society A list", requestB)
		}
	}

	// Sanity: society A's own request must be present.
	found := false
	for _, r := range results {
		if id, _ := r["id"].(string); id == requestA {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("society A request %s missing from list", requestA)
	}
}
