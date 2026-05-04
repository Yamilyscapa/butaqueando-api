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

type ListFollowingActivityQuery struct {
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

type FollowingActivityActorData struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	Bio         *string `json:"bio"`
}

type FollowingActivityPlayData struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	TheaterName        string   `json:"theaterName"`
	City               *string  `json:"city"`
	AvailabilityStatus string   `json:"availabilityStatus"`
	PublishedAt        string   `json:"publishedAt"`
	PosterURL          *string  `json:"posterUrl"`
	AverageRating      *float64 `json:"averageRating,omitempty"`
	ReviewCount        int64    `json:"reviewCount"`
}

type FollowingActivityReviewData struct {
	ID               string  `json:"id"`
	Rating           int     `json:"rating"`
	Title            *string `json:"title"`
	Body             string  `json:"body"`
	ContainsSpoilers bool    `json:"containsSpoilers"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

type FollowingActivityItemData struct {
	ActivityType string                       `json:"activityType"`
	ActivityAt   string                       `json:"activityAt"`
	ActivityID   string                       `json:"activityId"`
	Actor        FollowingActivityActorData   `json:"actor"`
	Play         FollowingActivityPlayData    `json:"play"`
	Review       *FollowingActivityReviewData `json:"review,omitempty"`
}

type FollowingActivityListData struct {
	Items      []FollowingActivityItemData `json:"items"`
	NextCursor *string                     `json:"nextCursor,omitempty"`
}

type FollowRecord struct {
	UserID      string
	DisplayName string
	Bio         *string
	FollowedAt  time.Time
}

type FollowingActivityRecord struct {
	ActivityType       string
	ActivityAt         time.Time
	ActivityID         string
	ActorID            string
	ActorDisplayName   string
	ActorBio           *string
	PlayID             string
	PlayTitle          string
	TheaterName        string
	City               *string
	AvailabilityStatus string
	PublishedAt        time.Time
	PosterMediaID      *string
	AverageRating      *float64
	ReviewCount        int64
	ReviewID           *string
	Rating             *int
	Title              *string
	Body               *string
	ContainsSpoilers   *bool
	ReviewCreatedAt    *time.Time
	ReviewUpdatedAt    *time.Time
}

type followCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	UserID    string    `json:"userId"`
}

type followingActivityCursor struct {
	ActivityAt time.Time `json:"activityAt"`
	ActivityID string    `json:"activityId"`
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

func encodeFollowingActivityCursor(cursor followingActivityCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeFollowingActivityCursor(raw string) (*followingActivityCursor, error) {
	if raw == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor followingActivityCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}

	if cursor.ActivityAt.IsZero() || cursor.ActivityID == "" {
		return nil, fmt.Errorf("invalid cursor payload")
	}

	return &cursor, nil
}
