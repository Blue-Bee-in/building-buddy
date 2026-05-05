package maintenance

import "time"

type Request struct {
	ID              string
	SocietyID       string
	Title           string
	Description     *string
	ContractorName  string
	QuotedAmount    float64
	ApprovedAmount  *float64
	Status          string
	RejectionReason *string
	CreatedBy       string
	ApprovedBy      *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ApprovedAt      *time.Time
	// Populated when PDF is ready
	PDFURL *string
}

const (
	StatusDraft     = "draft"
	StatusSubmitted = "submitted"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
)
