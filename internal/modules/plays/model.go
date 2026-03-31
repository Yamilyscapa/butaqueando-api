package plays

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type FeedQuery struct {
	Section string `form:"section"`
	GenreID string `form:"genreId"`
	Cursor  string `form:"cursor"`
	Limit   int    `form:"limit"`
}

type SearchQuery struct {
	Q                  string `form:"q"`
	GenreID            string `form:"genreId"`
	City               string `form:"city"`
	Theater            string `form:"theater"`
	AvailabilityStatus string `form:"availabilityStatus"`
	Cursor             string `form:"cursor"`
	Limit              int    `form:"limit"`
}

type ListReviewsQuery struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type ListSubmissionsQuery struct {
	Status string `form:"status"`
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type CreateReviewRequest struct {
	Rating           int     `json:"rating"`
	Title            *string `json:"title"`
	Body             string  `json:"body"`
	ContainsSpoilers *bool   `json:"containsSpoilers"`
}

type UpdateReviewRequest struct {
	Rating           *int    `json:"rating"`
	Title            *string `json:"title"`
	Body             *string `json:"body"`
	ContainsSpoilers *bool   `json:"containsSpoilers"`
}

type CreateReviewCommentRequest struct {
	Body string `json:"body"`
}

type CreateSubmissionRequest struct {
	Title              string  `json:"title"`
	Synopsis           string  `json:"synopsis"`
	Director           string  `json:"director"`
	DurationMinutes    int     `json:"durationMinutes"`
	TheaterName        string  `json:"theaterName"`
	City               *string `json:"city"`
	AvailabilityStatus *string `json:"availabilityStatus"`
}

type UpdateSubmissionRequest struct {
	Title              *string `json:"title"`
	Synopsis           *string `json:"synopsis"`
	Director           *string `json:"director"`
	DurationMinutes    *int    `json:"durationMinutes"`
	TheaterName        *string `json:"theaterName"`
	City               *string `json:"city"`
	AvailabilityStatus *string `json:"availabilityStatus"`
}

type RejectSubmissionRequest struct {
	Reason string `json:"reason"`
}

type SetEngagementRequest struct {
	Kind string `json:"kind"`
}

type PlayCardData struct {
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

type FeedData struct {
	Items      []PlayCardData `json:"items"`
	NextCursor *string        `json:"nextCursor,omitempty"`
}

type SearchData struct {
	Items      []PlayCardData `json:"items"`
	NextCursor *string        `json:"nextCursor,omitempty"`
}

type PlayStatsData struct {
	AverageRating *float64 `json:"averageRating,omitempty"`
	ReviewCount   int64    `json:"reviewCount"`
}

type PlayGenreData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PlayCastMemberData struct {
	PersonName   string `json:"personName"`
	RoleName     string `json:"roleName"`
	BillingOrder int    `json:"billingOrder"`
}

type PlayMediaData struct {
	Kind      string  `json:"kind"`
	URL       string  `json:"url"`
	AltText   *string `json:"altText"`
	SortOrder int     `json:"sortOrder"`
}

type PlayDetailsData struct {
	ID                 string               `json:"id"`
	Title              string               `json:"title"`
	Synopsis           string               `json:"synopsis"`
	Director           string               `json:"director"`
	DurationMinutes    int                  `json:"durationMinutes"`
	TheaterName        string               `json:"theaterName"`
	City               *string              `json:"city"`
	AvailabilityStatus string               `json:"availabilityStatus"`
	PublishedAt        string               `json:"publishedAt"`
	Stats              PlayStatsData        `json:"stats"`
	Genres             []PlayGenreData      `json:"genres"`
	Cast               []PlayCastMemberData `json:"cast"`
	Media              []PlayMediaData      `json:"media"`
}

type ReviewData struct {
	ID               string  `json:"id"`
	UserID           string  `json:"userId"`
	DisplayName      string  `json:"displayName"`
	Rating           int     `json:"rating"`
	Title            *string `json:"title"`
	Body             string  `json:"body"`
	ContainsSpoilers bool    `json:"containsSpoilers"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

type ReviewListData struct {
	Items      []ReviewData `json:"items"`
	NextCursor *string      `json:"nextCursor,omitempty"`
}

type ReviewCommentData struct {
	ID          string `json:"id"`
	ReviewID    string `json:"reviewId"`
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	Body        string `json:"body"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type SubmissionData struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	Synopsis           string  `json:"synopsis"`
	Director           string  `json:"director"`
	DurationMinutes    int     `json:"durationMinutes"`
	TheaterName        string  `json:"theaterName"`
	City               *string `json:"city"`
	AvailabilityStatus string  `json:"availabilityStatus"`
	CurationStatus     string  `json:"curationStatus"`
	CreatedByUserID    string  `json:"createdByUserId"`
	ModeratedByUserID  *string `json:"moderatedByUserId"`
	ModeratedAt        *string `json:"moderatedAt"`
	PublishedAt        *string `json:"publishedAt"`
	RejectedReason     *string `json:"rejectedReason"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
}

type SubmissionListData struct {
	Items      []SubmissionData `json:"items"`
	NextCursor *string          `json:"nextCursor,omitempty"`
}

type EngagementStateData struct {
	PlayID   string `json:"playId"`
	Wishlist bool   `json:"wishlist"`
	Attended bool   `json:"attended"`
}

type PlayListRecord struct {
	ID                 string
	Title              string
	TheaterName        string
	City               *string
	AvailabilityStatus string
	PublishedAt        time.Time
	PosterURL          *string
	AverageRating      *float64
	ReviewCount        int64
}

type PlayDetailsRecord struct {
	ID                 string
	Title              string
	Synopsis           string
	Director           string
	DurationMinutes    int
	TheaterName        string
	City               *string
	AvailabilityStatus string
	PublishedAt        time.Time
	AverageRating      *float64
	ReviewCount        int64
}

type PlayGenreRecord struct {
	ID   string
	Name string
}

type PlayCastRecord struct {
	PersonName   string
	RoleName     string
	BillingOrder int
}

type PlayMediaRecord struct {
	Kind      string
	URL       string
	AltText   *string
	SortOrder int
}

type ReviewRecord struct {
	ID               string
	UserID           string
	DisplayName      string
	Rating           int
	Title            *string
	Body             string
	ContainsSpoilers bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CreateReviewParams struct {
	Rating           int
	Title            *string
	Body             string
	ContainsSpoilers bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UpdateReviewParams struct {
	Rating           *int
	Title            *string
	TitleProvided    bool
	Body             *string
	ContainsSpoilers *bool
	UpdatedAt        time.Time
}

type ReviewMetadataRecord struct {
	ReviewID           string
	PlayID             string
	UserID             string
	ReviewStatus       string
	PlayCurationStatus string
}

type CreateReviewCommentParams struct {
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateSubmissionParams struct {
	Title              string
	Synopsis           string
	Director           string
	DurationMinutes    int
	TheaterName        string
	City               *string
	AvailabilityStatus string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type UpdateSubmissionParams struct {
	Title                *string
	Synopsis             *string
	Director             *string
	DurationMinutes      *int
	TheaterName          *string
	City                 *string
	CityProvided         bool
	AvailabilityStatus   *string
	SetPendingResubmit   bool
	ClearModerationAudit bool
	UpdatedAt            time.Time
}

type ListSubmissionsParams struct {
	Status *string
	After  *submissionListCursor
	Limit  int
}

type SubmissionRecord struct {
	ID                 string
	Title              string
	Synopsis           string
	Director           string
	DurationMinutes    int
	TheaterName        string
	City               *string
	AvailabilityStatus string
	CurationStatus     string
	CreatedByUserID    string
	ModeratedByUserID  *string
	ModeratedAt        *time.Time
	PublishedAt        *time.Time
	RejectedReason     *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ReviewCommentRecord struct {
	ID          string
	ReviewID    string
	UserID      string
	DisplayName string
	Body        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type EngagementStateRecord struct {
	Wishlist bool
	Attended bool
}

type playListCursor struct {
	PublishedAt time.Time `json:"publishedAt"`
	PlayID      string    `json:"playId"`
}

type reviewListCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ReviewID  string    `json:"reviewId"`
}

type submissionListCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	PlayID    string    `json:"playId"`
}

func encodePlayListCursor(cursor playListCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodePlayListCursor(raw string) (*playListCursor, error) {
	if raw == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor playListCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}

	if cursor.PublishedAt.IsZero() || cursor.PlayID == "" {
		return nil, fmt.Errorf("invalid cursor payload")
	}

	return &cursor, nil
}

func encodeReviewListCursor(cursor reviewListCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeReviewListCursor(raw string) (*reviewListCursor, error) {
	if raw == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor reviewListCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}

	if cursor.CreatedAt.IsZero() || cursor.ReviewID == "" {
		return nil, fmt.Errorf("invalid cursor payload")
	}

	return &cursor, nil
}

func encodeSubmissionListCursor(cursor submissionListCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeSubmissionListCursor(raw string) (*submissionListCursor, error) {
	if raw == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor submissionListCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}

	if cursor.CreatedAt.IsZero() || cursor.PlayID == "" {
		return nil, fmt.Errorf("invalid cursor payload")
	}

	return &cursor, nil
}
