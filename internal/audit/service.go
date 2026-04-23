package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"main/internal/database"
)

const (
	EventSetupCompleted        = "setup_completed"
	EventLoginFailed           = "login_failed"
	EventLoginSucceeded        = "login_succeeded"
	EventRegistrationSucceeded = "registration_succeeded"
	EventRefreshFailed         = "refresh_failed"
	EventRefreshSucceeded      = "refresh_succeeded"
	EventTokenReuseDetected    = "token_reuse_detected"
	EventLogoutSucceeded       = "logout_succeeded"
	EventPasswordChanged       = "password_changed"
	EventRoleChanged           = "role_changed"
	EventUserCreated           = "user_created"
	EventUserUpdated           = "user_updated"
	EventUserDisabled          = "user_disabled"
	EventUserEnabled           = "user_enabled"
	EventAccountDeleted        = "account_deleted"
)

const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
)

type Entry struct {
	ActorUserID   *int64
	SubjectUserID *int64
	AuthSessionID *string
	EventType     string
	Outcome       string
	Reason        *string
	IP            *string
	UserAgent     *string
	Metadata      map[string]any
	OccurredAt    time.Time
}

type Record struct {
	ID            int64          `json:"id"`
	ActorUserID   *int64         `json:"actorUserId"`
	SubjectUserID *int64         `json:"subjectUserId"`
	AuthSessionID *string        `json:"authSessionId"`
	EventType     string         `json:"eventType"`
	Outcome       string         `json:"outcome"`
	Reason        *string        `json:"reason"`
	IP            *string        `json:"ip"`
	UserAgent     *string        `json:"userAgent"`
	Metadata      map[string]any `json:"metadata"`
	OccurredAt    time.Time      `json:"occurredAt"`
}

type ListParams struct {
	Page     int
	PageSize int
}

type ListResult struct {
	Items    []Record `json:"items"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
	Total    int      `json:"total"`
}

type Service struct {
	db database.DBTX
}

func NewService(db database.DBTX) *Service {
	return &Service{db: db}
}

func (s *Service) Log(ctx context.Context, entry Entry) error {
	occurredAt := entry.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	var metadataJSON *string
	if len(entry.Metadata) > 0 {
		payload, err := json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("audit: marshal metadata: %w", err)
		}
		value := string(payload)
		metadataJSON = &value
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO audit_logs (
			actor_user_id,
			subject_user_id,
			auth_session_id,
			event_type,
			outcome,
			reason,
			ip,
			user_agent,
			metadata_json,
			occurred_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ActorUserID,
		entry.SubjectUserID,
		entry.AuthSessionID,
		entry.EventType,
		entry.Outcome,
		entry.Reason,
		entry.IP,
		entry.UserAgent,
		metadataJSON,
		occurredAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("audit: insert log: %w", err)
	}

	return nil
}

func (s *Service) List(ctx context.Context, params ListParams) (ListResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 50
	}
	if params.PageSize > 200 {
		params.PageSize = 200
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("audit: count logs: %w", err)
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
			id,
			actor_user_id,
			subject_user_id,
			auth_session_id,
			event_type,
			outcome,
			reason,
			ip,
			user_agent,
			metadata_json,
			occurred_at
		FROM audit_logs
		ORDER BY occurred_at DESC, id DESC
		LIMIT ? OFFSET ?`,
		params.PageSize,
		(params.Page-1)*params.PageSize,
	)
	if err != nil {
		return ListResult{}, fmt.Errorf("audit: list logs: %w", err)
	}
	defer rows.Close()

	items := make([]Record, 0, params.PageSize)
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return ListResult{}, err
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("audit: iterate logs: %w", err)
	}

	return ListResult{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(scanner rowScanner) (Record, error) {
	var record Record
	var actorUserID sql.NullInt64
	var subjectUserID sql.NullInt64
	var authSessionID sql.NullString
	var reason sql.NullString
	var ip sql.NullString
	var userAgent sql.NullString
	var metadataJSON sql.NullString
	var occurredAt int64

	err := scanner.Scan(
		&record.ID,
		&actorUserID,
		&subjectUserID,
		&authSessionID,
		&record.EventType,
		&record.Outcome,
		&reason,
		&ip,
		&userAgent,
		&metadataJSON,
		&occurredAt,
	)
	if err != nil {
		return Record{}, fmt.Errorf("audit: scan log: %w", err)
	}

	record.ActorUserID = nullableInt64Pointer(actorUserID)
	record.SubjectUserID = nullableInt64Pointer(subjectUserID)
	record.AuthSessionID = nullableStringPointer(authSessionID)
	record.Reason = nullableStringPointer(reason)
	record.IP = nullableStringPointer(ip)
	record.UserAgent = nullableStringPointer(userAgent)
	record.OccurredAt = time.Unix(occurredAt, 0).UTC()

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &record.Metadata); err != nil {
			return Record{}, fmt.Errorf("audit: decode metadata: %w", err)
		}
	}

	return record, nil
}

func nullableInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	next := value.Int64
	return &next
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	next := value.String
	return &next
}
