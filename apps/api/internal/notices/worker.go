package notices

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Blue-Bee-in/building-buddy/internal/push"
)

// PushJobArgs holds parameters for a notice push notification job.
type PushJobArgs struct {
	NoticeID   string  `json:"notice_id"`
	SocietyID  string  `json:"society_id"`
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	TargetFlat *string `json:"target_flat,omitempty"`
}

func (PushJobArgs) Kind() string { return "notice.push" }

// PushWorker sends Expo push notifications for a new notice.
type PushWorker struct {
	river.WorkerDefaults[PushJobArgs]
	pool *pgxpool.Pool
}

func NewPushWorker(pool *pgxpool.Pool) *PushWorker {
	return &PushWorker{pool: pool}
}

func (w *PushWorker) Work(ctx context.Context, job *river.Job[PushJobArgs]) error {
	args := job.Args
	slog.Info("sending notice push", "notice_id", args.NoticeID, "society_id", args.SocietyID)

	tokens, userIDs, err := w.fetchTokens(ctx, args.SocietyID, args.TargetFlat)
	if err != nil {
		return fmt.Errorf("fetch tokens: %w", err)
	}
	if len(tokens) == 0 {
		slog.Info("no push tokens for society", "society_id", args.SocietyID)
		return nil
	}

	tickets, err := push.SendToTokens(ctx, tokens, args.Title, args.Body, map[string]string{
		"type":      "notice",
		"notice_id": args.NoticeID,
	})
	if err != nil {
		return fmt.Errorf("send push: %w", err)
	}

	// Store Expo ticket IDs so the receipt poller can detect DeviceNotRegistered.
	w.storePushTickets(ctx, args.NoticeID, tokens, userIDs, tickets)

	// Record delivery in notifications table (batch insert via unnest).
	if err := w.recordDelivery(ctx, args, userIDs); err != nil {
		slog.Error("record delivery", "err", err)
		// Non-fatal — push was sent.
	}

	slog.Info("notice push sent", "notice_id", args.NoticeID, "count", len(tokens))
	return nil
}

func (w *PushWorker) fetchTokens(ctx context.Context, societyID string, targetFlat *string) ([]string, []string, error) {
	var query string
	var args []any

	if targetFlat != nil {
		query = `
			SELECT DISTINCT ft.token, ft.user_id
			FROM fcm_tokens ft
			JOIN memberships m ON m.user_id = ft.user_id
			WHERE m.society_id = $1 AND m.flat_id = $2 AND m.is_active = true
			  AND ft.app = 'resident'`
		args = []any{societyID, *targetFlat}
	} else {
		query = `
			SELECT DISTINCT ft.token, ft.user_id
			FROM fcm_tokens ft
			JOIN memberships m ON m.user_id = ft.user_id
			WHERE m.society_id = $1 AND m.is_active = true
			  AND ft.app = 'resident'`
		args = []any{societyID}
	}

	rows, err := w.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var tokens, userIDs []string
	for rows.Next() {
		var token, userID string
		if err := rows.Scan(&token, &userID); err != nil {
			continue
		}
		tokens = append(tokens, token)
		userIDs = append(userIDs, userID)
	}
	return tokens, userIDs, nil
}

// storePushTickets persists Expo ticket IDs for later receipt polling.
// tickets is parallel to tokens (same index = same token). Empty-token entries
// are skipped by SendToTokens, so ticket count may be ≤ token count.
func (w *PushWorker) storePushTickets(ctx context.Context, noticeID string, tokens []string, _ []string, tickets []push.Ticket) {
	// Build (expo_id, token) pairs for successful tickets only.
	// tickets[i] corresponds to tokens[i] (non-empty tokens, same order).
	nonEmpty := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t != "" {
			nonEmpty = append(nonEmpty, t)
		}
	}

	if len(tickets) == 0 || len(nonEmpty) == 0 {
		return
	}

	var expoIDs, ticketTokens []string
	for i, ticket := range tickets {
		if ticket.Status == "ok" && ticket.ID != "" && i < len(nonEmpty) {
			expoIDs = append(expoIDs, ticket.ID)
			ticketTokens = append(ticketTokens, nonEmpty[i])
		}
	}

	if len(expoIDs) == 0 {
		return
	}

	noticeIDs := make([]string, len(expoIDs))
	for i := range expoIDs {
		noticeIDs[i] = noticeID
	}

	_, err := w.pool.Exec(ctx, `
		INSERT INTO push_tickets (expo_id, token, notice_id)
		SELECT unnest($1::text[]), unnest($2::text[]), unnest($3::uuid[])`,
		expoIDs, ticketTokens, noticeIDs,
	)
	if err != nil {
		slog.Error("store push tickets", "notice_id", noticeID, "err", err)
	}
}

// recordDelivery batch-inserts one notification row per recipient.
func (w *PushWorker) recordDelivery(ctx context.Context, args PushJobArgs, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	societyIDs := make([]string, len(userIDs))
	titles := make([]string, len(userIDs))
	bodies := make([]string, len(userIDs))
	noticeIDs := make([]string, len(userIDs))
	for i := range userIDs {
		societyIDs[i] = args.SocietyID
		titles[i] = args.Title
		bodies[i] = args.Body
		noticeIDs[i] = args.NoticeID
	}

	_, err := w.pool.Exec(ctx, `
		INSERT INTO notifications (user_id, society_id, title, body, type, ref_id)
		SELECT unnest($1::uuid[]),
		       unnest($2::uuid[]),
		       unnest($3::text[]),
		       unnest($4::text[]),
		       'notice',
		       unnest($5::uuid[])`,
		userIDs, societyIDs, titles, bodies, noticeIDs,
	)
	return err
}
