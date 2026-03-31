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

type playListCursor struct {
	PublishedAt time.Time `json:"publishedAt"`
	PlayID      string    `json:"playId"`
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
