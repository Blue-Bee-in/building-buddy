package billing

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func Router(db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	r.Use(apimw.Auth(db))

	r.With(apimw.RequireRoles("resident", "owner")).Get("/invoices", handleListInvoices)
	r.With(apimw.RequireRoles("resident", "owner")).Get("/invoices/{invoiceID}", handleGetInvoice)

	r.With(apimw.RequireAdmin).Get("/invoices/all", handleListAllInvoices)
	r.With(apimw.RequireAdmin).Post("/invoices/{invoiceID}/manual-payment", handleManualPayment)
	r.With(apimw.RequireAdmin).Get("/invoices/export", handleExport)

	r.Post("/webhooks/razorpay", handleRazorpayWebhook)

	return r
}

func handleListInvoices(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(http.StatusNotImplemented) }
func handleGetInvoice(w http.ResponseWriter, r *http.Request)      { w.WriteHeader(http.StatusNotImplemented) }
func handleListAllInvoices(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotImplemented) }
func handleManualPayment(w http.ResponseWriter, r *http.Request)   { w.WriteHeader(http.StatusNotImplemented) }
func handleExport(w http.ResponseWriter, r *http.Request)          { w.WriteHeader(http.StatusNotImplemented) }
func handleRazorpayWebhook(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotImplemented) }
