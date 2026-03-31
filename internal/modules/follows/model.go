package follows

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type ListFollowsQuery struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type FollowActionData struct {
	UserID    string `json:"userId"`
	Following bool   `json:"following"`
}

type FollowListItemData struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	Bio         *string `json:"bio"`
	FollowedAt  string  `json:"followedAt"`
}

type FollowListData struct {
	Items      []FollowListItemData `json:"items"`
	NextCursor *string              `json:"nextCursor,omitempty"`
}

type FollowRecord struct {
	UserID      string
	DisplayName string
	Bio         *string
	FollowedAt  time.Time
}

type followCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	UserID    string    `json:"userId"`
}

func encodeFollowCursor(cursor followCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeFollowCursor(raw string) (*followCursor, error) {
	if raw == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor followCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}

	if cursor.CreatedAt.IsZero() || cursor.UserID == "" {
		return nil, fmt.Errorf("invalid cursor payload")
	}

	return &cursor, nil
}
