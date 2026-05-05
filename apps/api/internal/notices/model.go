package notices

import "time"

type Notice struct {
	ID          string
	SocietyID   string
	CreatedBy   string
	Title       string
	Body        string
	Category    string
	TargetFlat  *string
	IsPinned    bool
	PublishedAt *time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	// Delivery stats (populated from notifications table)
	DeliveredCount int
	FailedCount    int
}

const (
	CategoryGeneral     = "general"
	CategoryMaintenance = "maintenance"
	CategoryEvent       = "event"
	CategoryEmergency   = "emergency"
)
