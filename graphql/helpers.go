package graphql

import (
	"context"
	"member_API/graphql/model"
	"member_API/models"
	"time"

	"github.com/google/uuid"
)

// dbToModel converts DB Member to GraphQL model
func dbToModel(m models.Member) *model.Member {
	var created, updated *string
	if !m.CreationTime.IsZero() {
		s := formatTime(m.CreationTime)
		created = &s
	}
	if m.LastModificationTime != nil && !m.LastModificationTime.IsZero() {
		s := formatTime(*m.LastModificationTime)
		updated = &s
	}
	return &model.Member{
		ID:        formatID(m.ID),
		Name:      m.Name,
		Email:     m.Email,
		CreatedAt: created,
		UpdatedAt: updated,
	}
}

// formatTime formats time to RFC3339 string
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// formatID converts UUID to string
func formatID(id uuid.UUID) string {
	return id.String()
}

// getUserIDFromContext extracts user ID from context
func getUserIDFromContext(ctx context.Context) uuid.UUID {
	userIDStr, ok := ctx.Value("user_id").(string)
	if !ok || userIDStr == "" {
		return uuid.Nil
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil
	}
	return userID
}

// stringPtr converts string to *string pointer
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ptrToString converts *string pointer to string
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
