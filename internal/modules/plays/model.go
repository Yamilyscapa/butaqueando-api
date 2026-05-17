package plays

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
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

type ListGenresQuery struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type ListCitiesQuery struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type ListTheatersQuery struct {
	CityID string `form:"cityId"`
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type ListMyEngagementsQuery struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type ListUserReviewsQuery struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type ListOwnedPlaysQuery struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type ListPlayEditSuggestionsQuery struct {
	Status string `form:"status"`
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type ListModerationQueueQuery struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type CreateReviewRequest struct {
	Rating           float64 `json:"rating"`
	Title            *string `json:"title"`
	Body             string  `json:"body"`
	ContainsSpoilers *bool   `json:"containsSpoilers"`
}

type UpdateReviewRequest struct {
	Rating           *float64 `json:"rating"`
	Title            *string  `json:"title"`
	Body             *string  `json:"body"`
	ContainsSpoilers *bool    `json:"containsSpoilers"`
}

type CreateReviewCommentRequest struct {
	Body string `json:"body"`
}

type UpdateReviewCommentStatusRequest struct {
	Status string `json:"status"`
}

type CreateSubmissionRequest struct {
	Title              string   `json:"title"`
	Synopsis           string   `json:"synopsis"`
	Director           string   `json:"director"`
	DurationMinutes    int      `json:"durationMinutes"`
	TheaterName        string   `json:"theaterName"`
	IsCustomTheater    bool     `json:"isCustomTheater"`
	City               *string  `json:"city"`
	AvailabilityStatus *string  `json:"availabilityStatus"`
	GenreIDs           []string `json:"genreIds"`
	CustomGenreName    *string  `json:"customGenreName"`
	Production         *string  `json:"production"`
}

type CreateSubmissionMediaUploadRequest struct {
	Kind          string `json:"kind"`
	ContentType   string `json:"contentType"`
	ContentLength int64  `json:"contentLength"`
}

type AttachSubmissionMediaRequest struct {
	Kind      string  `json:"kind"`
	ObjectKey string  `json:"objectKey"`
	AltText   *string `json:"altText"`
	SortOrder *int    `json:"sortOrder"`
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

type CreatePlayEditSuggestionRequest struct {
	PlayID             string   `json:"playId"`
	Title              *string  `json:"title"`
	Synopsis           *string  `json:"synopsis"`
	Director           *string  `json:"director"`
	DurationMinutes    *int     `json:"durationMinutes"`
	TheaterName        *string  `json:"theaterName"`
	City               *string  `json:"city"`
	AvailabilityStatus *string  `json:"availabilityStatus"`
	GenreIDs           []string `json:"genreIds"`
}

type UpdatePlayEditSuggestionRequest struct {
	Title              *string  `json:"title"`
	Synopsis           *string  `json:"synopsis"`
	Director           *string  `json:"director"`
	DurationMinutes    *int     `json:"durationMinutes"`
	TheaterName        *string  `json:"theaterName"`
	City               *string  `json:"city"`
	AvailabilityStatus *string  `json:"availabilityStatus"`
	GenreIDs           []string `json:"genreIds"`
}

type RejectSubmissionRequest struct {
	Reason string `json:"reason"`
}

type CreateGenreRequest struct {
	Name string `json:"name"`
}

type CreateCityRequest struct {
	Name string `json:"name"`
}

type CreateTheaterRequest struct {
	CityID string `json:"cityId"`
	Name   string `json:"name"`
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

type GenreData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CityData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TheaterData struct {
	ID     string `json:"id"`
	CityID string `json:"cityId"`
	Name   string `json:"name"`
}

type GenreListData struct {
	Items      []GenreData `json:"items"`
	NextCursor *string     `json:"nextCursor,omitempty"`
}

type CityListData struct {
	Items      []CityData `json:"items"`
	NextCursor *string    `json:"nextCursor,omitempty"`
}

type TheaterListData struct {
	Items      []TheaterData `json:"items"`
	NextCursor *string       `json:"nextCursor,omitempty"`
}

type PlayCastMemberData struct {
	PersonName   string `json:"personName"`
	RoleName     string `json:"roleName"`
	BillingOrder int    `json:"billingOrder"`
}

type PlayMediaData struct {
	ID        string                  `json:"id"`
	Kind      string                  `json:"kind"`
	URL       string                  `json:"url"`
	AltText   *string                 `json:"altText"`
	SortOrder int                     `json:"sortOrder"`
	Variants  []PlayMediaVariantData  `json:"variants,omitempty"`
	Blurhash  *string                 `json:"blurhash,omitempty"`
}

type PlayMediaVariantData struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}

type CreateSubmissionMediaUploadData struct {
	ObjectKey string `json:"objectKey"`
	UploadURL string `json:"uploadUrl"`
}

type PlayDetailsData struct {
	ID                 string               `json:"id"`
	Title              string               `json:"title"`
	Synopsis           string               `json:"synopsis"`
	Director           string               `json:"director"`
	DurationMinutes    int                  `json:"durationMinutes"`
	TheaterName        string               `json:"theaterName"`
	IsCustomTheater    bool                 `json:"isCustomTheater"`
	City               *string              `json:"city"`
	AvailabilityStatus string               `json:"availabilityStatus"`
	PublishedAt        string               `json:"publishedAt"`
	Stats              PlayStatsData        `json:"stats"`
	Genres             []PlayGenreData      `json:"genres"`
	CustomGenreName    *string              `json:"customGenreName,omitempty"`
	Production         *string              `json:"production,omitempty"`
	Cast               []PlayCastMemberData `json:"cast"`
	Media              []PlayMediaData      `json:"media"`
}

type ReviewData struct {
	ID               string  `json:"id"`
	UserID           string  `json:"userId"`
	DisplayName      string  `json:"displayName"`
	Rating           float64 `json:"rating"`
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

type ReviewCommentStatusData struct {
	ID        string `json:"id"`
	ReviewID  string `json:"reviewId"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
}

type SubmissionData struct {
	ID                 string          `json:"id"`
	Title              string          `json:"title"`
	Synopsis           string          `json:"synopsis"`
	Director           string          `json:"director"`
	DurationMinutes    int             `json:"durationMinutes"`
	TheaterName        string          `json:"theaterName"`
	IsCustomTheater    bool            `json:"isCustomTheater"`
	City               *string         `json:"city"`
	AvailabilityStatus string          `json:"availabilityStatus"`
	Genres             []PlayGenreData `json:"genres"`
	CustomGenreName    *string         `json:"customGenreName,omitempty"`
	Production         *string         `json:"production,omitempty"`
	Media              []PlayMediaData `json:"media,omitempty"`
	CurationStatus     string          `json:"curationStatus"`
	CreatedByUserID    string          `json:"createdByUserId"`
	ModeratedByUserID  *string         `json:"moderatedByUserId"`
	ModeratedAt        *string         `json:"moderatedAt"`
	PublishedAt        *string         `json:"publishedAt"`
	RejectedReason     *string         `json:"rejectedReason"`
	CreatedAt          string          `json:"createdAt"`
	UpdatedAt          string          `json:"updatedAt"`
}

type SubmissionListData struct {
	Items      []SubmissionData `json:"items"`
	NextCursor *string          `json:"nextCursor,omitempty"`
}

type PlayEditSuggestionSummaryData struct {
	ID        string  `json:"id"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	PlayID    string  `json:"playId"`
	Title     string  `json:"title"`
	City      *string `json:"city"`
}

type OwnedPlayData struct {
	ID                   string                         `json:"id"`
	Title                string                         `json:"title"`
	TheaterName          string                         `json:"theaterName"`
	City                 *string                        `json:"city"`
	AvailabilityStatus   string                         `json:"availabilityStatus"`
	PublishedAt          string                         `json:"publishedAt"`
	PosterURL            *string                        `json:"posterUrl,omitempty"`
	LatestEditSuggestion *PlayEditSuggestionSummaryData `json:"latestEditSuggestion,omitempty"`
}

type OwnedPlayListData struct {
	Items      []OwnedPlayData `json:"items"`
	NextCursor *string         `json:"nextCursor,omitempty"`
}

type PlayEditSuggestionData struct {
	ID                 string          `json:"id"`
	PlayID             string          `json:"playId"`
	Title              string          `json:"title"`
	Synopsis           string          `json:"synopsis"`
	Director           string          `json:"director"`
	DurationMinutes    int             `json:"durationMinutes"`
	TheaterName        string          `json:"theaterName"`
	City               *string         `json:"city"`
	AvailabilityStatus string          `json:"availabilityStatus"`
	Genres             []PlayGenreData `json:"genres"`
	Media              []PlayMediaData `json:"media,omitempty"`
	Status             string          `json:"status"`
	CreatedByUserID    string          `json:"createdByUserId"`
	ModeratedByUserID  *string         `json:"moderatedByUserId"`
	ModeratedAt        *string         `json:"moderatedAt"`
	RejectedReason     *string         `json:"rejectedReason"`
	CreatedAt          string          `json:"createdAt"`
	UpdatedAt          string          `json:"updatedAt"`
}

type PlayEditSuggestionListData struct {
	Items      []PlayEditSuggestionData `json:"items"`
	NextCursor *string                  `json:"nextCursor,omitempty"`
}

type ModerationQueueItemData struct {
	ItemType         string  `json:"itemType"`
	ItemID           string  `json:"itemId"`
	PlayID           string  `json:"playId"`
	Title            string  `json:"title"`
	TheaterName      string  `json:"theaterName"`
	City             *string `json:"city"`
	CreatedAt        string  `json:"createdAt"`
	CreatedByUserID  string  `json:"createdByUserId"`
	SubmissionStatus *string `json:"submissionStatus,omitempty"`
	SuggestionStatus *string `json:"suggestionStatus,omitempty"`
}

type ModerationQueueData struct {
	Items      []ModerationQueueItemData `json:"items"`
	NextCursor *string                   `json:"nextCursor,omitempty"`
}

type EngagementStateData struct {
	PlayID    string `json:"playId"`
	Wishlist  bool   `json:"wishlist"`
	Attended  bool   `json:"attended"`
	Favorited bool   `json:"favorited"`
}

type MyEngagementPlayData struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	TheaterName        string   `json:"theaterName"`
	City               *string  `json:"city"`
	AvailabilityStatus string   `json:"availabilityStatus"`
	PublishedAt        string   `json:"publishedAt"`
	PosterURL          *string  `json:"posterUrl"`
	AverageRating      *float64 `json:"averageRating,omitempty"`
	ReviewCount        int64    `json:"reviewCount"`
	EngagedAt          string   `json:"engagedAt"`
}

type MyEngagementPlayListData struct {
	Items      []MyEngagementPlayData `json:"items"`
	NextCursor *string                `json:"nextCursor,omitempty"`
}

type UserReviewPlayData struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	TheaterName        string  `json:"theaterName"`
	City               *string `json:"city"`
	AvailabilityStatus string  `json:"availabilityStatus"`
	PublishedAt        string  `json:"publishedAt"`
	PosterURL          *string `json:"posterUrl"`
}

type UserReviewData struct {
	ID               string             `json:"id"`
	Play             UserReviewPlayData `json:"play"`
	Rating           float64            `json:"rating"`
	Title            *string            `json:"title"`
	Body             string             `json:"body"`
	ContainsSpoilers bool               `json:"containsSpoilers"`
	CreatedAt        string             `json:"createdAt"`
	UpdatedAt        string             `json:"updatedAt"`
}

type UserReviewListData struct {
	Items      []UserReviewData `json:"items"`
	NextCursor *string          `json:"nextCursor,omitempty"`
}

type PlayListRecord struct {
	ID                 string
	Title              string
	TheaterName        string
	City               *string
	AvailabilityStatus string
	PublishedAt        time.Time
	PosterMediaID      *string
	AverageRating      *float64
	ReviewCount        int64
	TrendScore         int64
}

type EngagementPlayRecord struct {
	ID                 string
	Title              string
	TheaterName        string
	City               *string
	AvailabilityStatus string
	PublishedAt        time.Time
	PosterMediaID      *string
	AverageRating      *float64
	ReviewCount        int64
	EngagedAt          time.Time
}

type PlayDetailsRecord struct {
	ID                 string
	Title              string
	Synopsis           string
	Director           string
	DurationMinutes    int
	TheaterName        string
	IsCustomTheater    bool
	City               *string
	AvailabilityStatus string
	PublishedAt        time.Time
	AverageRating      *float64
	ReviewCount        int64
	CustomGenreName    *string
	Production         *string
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
	ID        string
	Kind      string
	ObjectKey string
	AltText   *string
	SortOrder int
	PlayID    string
	Variants  []MediaVariantRecord
	Blurhash  *string
}

type MediaVariantRecord struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	ObjectKey string `json:"objectKey"`
	SizeBytes int64  `json:"sizeBytes"`
}

type ReviewRecord struct {
	ID               string
	UserID           string
	DisplayName      string
	Rating           float64
	Title            *string
	Body             string
	ContainsSpoilers bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UserReviewRecord struct {
	ID                 string
	PlayID             string
	PlayTitle          string
	TheaterName        string
	City               *string
	AvailabilityStatus string
	PublishedAt        time.Time
	PosterMediaID      *string
	Rating             float64
	Title              *string
	Body               string
	ContainsSpoilers   bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreateReviewParams struct {
	Rating           float64
	Title            *string
	Body             string
	ContainsSpoilers bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UpdateReviewParams struct {
	Rating           *float64
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
	IsCustomTheater    bool
	City               *string
	AvailabilityStatus string
	GenreIDs           []string
	CustomGenreName    *string
	Production         *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreatePlayMediaParams struct {
	PlayID    string
	Kind      string
	ObjectKey string
	AltText   *string
	SortOrder int
	Variants  []MediaVariantRecord
	Blurhash  *string
	CreatedAt time.Time
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

type CreatePlayEditSuggestionParams struct {
	PlayID             string
	CreatedByUserID    string
	Title              string
	Synopsis           string
	Director           string
	DurationMinutes    int
	TheaterName        string
	City               *string
	AvailabilityStatus string
	GenreIDs           []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type UpdatePlayEditSuggestionParams struct {
	Title              *string
	Synopsis           *string
	Director           *string
	DurationMinutes    *int
	TheaterName        *string
	City               *string
	CityProvided       bool
	AvailabilityStatus *string
	GenreIDs           []string
	GenreIDsProvided   bool
	UpdatedAt          time.Time
}

type ModeratePlayEditSuggestionParams struct {
	AdminUserID    string
	Status         string
	RejectedReason *string
	ModeratedAt    time.Time
}

type ListSubmissionsParams struct {
	Status *string
	After  *submissionListCursor
	Limit  int
}

type ListPlayEditSuggestionsParams struct {
	CreatedByUserID *string
	PlayID          *string
	Status          *string
	After           *playEditSuggestionListCursor
	Limit           int
}

type ListGenresParams struct {
	After *genreListCursor
	Limit int
}

type ListCitiesParams struct {
	After *genreListCursor
	Limit int
}

type ListTheatersParams struct {
	CityID *string
	After  *genreListCursor
	Limit  int
}

type ListUserReviewsParams struct {
	After *reviewListCursor
	Limit int
}

type SubmissionRecord struct {
	ID                 string
	Title              string
	Synopsis           string
	Director           string
	DurationMinutes    int
	TheaterName        string
	IsCustomTheater    bool
	City               *string
	AvailabilityStatus string
	CurationStatus     string
	CreatedByUserID    string
	ModeratedByUserID  *string
	ModeratedAt        *time.Time
	PublishedAt        *time.Time
	RejectedReason     *string
	CustomGenreName    *string
	Production         *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type OwnedPlayRecord struct {
	ID                 string
	Title              string
	TheaterName        string
	City               *string
	AvailabilityStatus string
	PublishedAt        time.Time
	PosterMediaID      *string
	CreatedAt          time.Time
}

type PlayEditSuggestionRecord struct {
	ID                 string
	PlayID             string
	Title              string
	Synopsis           string
	Director           string
	DurationMinutes    int
	TheaterName        string
	City               *string
	AvailabilityStatus string
	Status             string
	CreatedByUserID    string
	ModeratedByUserID  *string
	ModeratedAt        *time.Time
	RejectedReason     *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ModerationQueueRecord struct {
	ItemType         string
	ItemID           string
	PlayID           string
	Title            string
	TheaterName      string
	City             *string
	CreatedAt        time.Time
	CreatedByUserID  string
	SubmissionStatus *string
	SuggestionStatus *string
}

type GenreRecord struct {
	ID   string
	Name string
}

type CityRecord struct {
	ID   string
	Name string
}

type TheaterRecord struct {
	ID     string
	CityID string
	Name   string
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

type ReviewCommentStatusRecord struct {
	ID        string
	ReviewID  string
	Status    string
	UpdatedAt time.Time
}

type EngagementStateRecord struct {
	Wishlist  bool
	Attended  bool
	Favorited bool
}

type playListCursor struct {
	PublishedAt time.Time `json:"publishedAt"`
	PlayID      string    `json:"playId"`
}

type trendingFeedCursor struct {
	Section     string    `json:"section"`
	TrendScore  int64     `json:"trendScore"`
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

type playEditSuggestionListCursor struct {
	CreatedAt    time.Time `json:"createdAt"`
	SuggestionID string    `json:"suggestionId"`
}

type genreListCursor struct {
	Name    string `json:"name"`
	GenreID string `json:"genreId"`
}

type engagementPlayListCursor struct {
	EngagedAt time.Time `json:"engagedAt"`
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

func encodeTrendingFeedCursor(cursor trendingFeedCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeTrendingFeedCursor(raw string) (*trendingFeedCursor, error) {
	if raw == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor trendingFeedCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}

	if cursor.Section != "trending" || cursor.PublishedAt.IsZero() || cursor.PlayID == "" {
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

func encodePlayEditSuggestionListCursor(cursor playEditSuggestionListCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodePlayEditSuggestionListCursor(raw string) (*playEditSuggestionListCursor, error) {
	if raw == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor playEditSuggestionListCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}

	if cursor.CreatedAt.IsZero() || cursor.SuggestionID == "" {
		return nil, fmt.Errorf("invalid cursor payload")
	}

	return &cursor, nil
}

func encodeGenreListCursor(cursor genreListCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeGenreListCursor(raw string) (*genreListCursor, error) {
	if raw == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor genreListCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}

	if strings.TrimSpace(cursor.Name) == "" || cursor.GenreID == "" {
		return nil, fmt.Errorf("invalid cursor payload")
	}

	return &cursor, nil
}

func encodeEngagementPlayListCursor(cursor engagementPlayListCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeEngagementPlayListCursor(raw string) (*engagementPlayListCursor, error) {
	if raw == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor engagementPlayListCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}

	if cursor.EngagedAt.IsZero() || cursor.PlayID == "" {
		return nil, fmt.Errorf("invalid cursor payload")
	}

	return &cursor, nil
}
