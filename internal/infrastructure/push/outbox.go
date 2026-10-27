package push

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"time"
)

// RunOutbox polls durable events inserted by the business transaction. Row locks
// allow multiple API workers without concurrent delivery of the same event.
// Delivery is at-least-once: consumers can deduplicate by notification_id.
func RunOutbox(ctx context.Context, pool *pgxpool.Pool, sender PushService) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for n := 0; n < 20; n++ {
				found, err := deliverOne(ctx, pool, sender)
				if err != nil {
					slog.Error("outbox delivery", slog.Any("error", err))
					break
				}
				if !found {
					break
				}
			}
		}
	}
}
func deliverOne(parent context.Context, pool *pgxpool.Pool, sender PushService) (bool, error) {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var id, user, title, body string
	var raw []byte
	var attempts int
	err = tx.QueryRow(ctx, `SELECT id::text,user_id::text,title,body,data,attempts FROM notification_outbox WHERE delivered_at IS NULL AND available_at<=NOW() ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&id, &user, &title, &body, &raw, &attempts)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	data := map[string]string{}
	if err = json.Unmarshal(raw, &data); err != nil {
		return false, fmt.Errorf("outbox payload: %w", err)
	}
	data["notification_id"] = id
	sendErr := sender.SendNotification(ctx, user, title, body, data)
	if sendErr == nil {
		_, err = tx.Exec(ctx, `UPDATE notification_outbox SET delivered_at=NOW(),attempts=attempts+1,last_error=NULL WHERE id=$1`, id)
	} else {
		backoff := time.Duration(1<<min(attempts, 10)) * time.Second
		// Do not persist provider error strings, which can contain device credentials.
		_, err = tx.Exec(ctx, `UPDATE notification_outbox SET attempts=attempts+1,available_at=NOW()+$2::interval,last_error='delivery_failed' WHERE id=$1`, id, backoff.String())
	}
	if err != nil {
		return true, err
	}
	return true, tx.Commit(ctx)
}
