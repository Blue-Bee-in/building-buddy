package billing

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func Router() http.Handler {
	r := chi.NewRouter()
	r.Use(apimw.Auth)

	r.With(apimw.RequireRole("resident")).Get("/invoices", handleListInvoices)
	r.With(apimw.RequireRole("resident")).Get("/invoices/{invoiceID}", handleGetInvoice)

	r.With(apimw.RequireRole("society_admin", "committee_member")).Get("/invoices/all", handleListAllInvoices)
	r.With(apimw.RequireRole("society_admin")).Post("/invoices/{invoiceID}/manual-payment", handleManualPayment)
	r.With(apimw.RequireRole("society_admin")).Get("/invoices/export", handleExport)

	r.Post("/webhooks/razorpay", handleRazorpayWebhook)

	return r
}

func handleListInvoices(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(http.StatusNotImplemented) }
func handleGetInvoice(w http.ResponseWriter, r *http.Request)      { w.WriteHeader(http.StatusNotImplemented) }
func handleListAllInvoices(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotImplemented) }
func handleManualPayment(w http.ResponseWriter, r *http.Request)   { w.WriteHeader(http.StatusNotImplemented) }
func handleExport(w http.ResponseWriter, r *http.Request)          { w.WriteHeader(http.StatusNotImplemented) }
func handleRazorpayWebhook(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotImplemented) }
