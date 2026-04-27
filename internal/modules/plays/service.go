package plays

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/imageproc"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/butaqueando/api/internal/shared/worker"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	defaultListLimit       = 20
	maxListLimit           = 50
	defaultUploadURLTTL    = 15 * time.Minute
	defaultDownloadURLTTL  = 15 * time.Minute
	defaultMaxImageBytes   = 10 * 1024 * 1024
	defaultPlayMediaPrefix = "/v1/media/plays"
	playMediaMaxWidth      = 1080
	playMediaMaxHeight     = 1920
)

type repositoryPort interface {
	ListFeed(ctx context.Context, params FeedListParams) ([]PlayListRecord, error)
	SearchPublishedPlays(ctx context.Context, params SearchListParams) ([]PlayListRecord, error)
	GetPublishedPlayByID(ctx context.Context, playID string) (PlayDetailsRecord, error)
	ListPlayGenres(ctx context.Context, playID string) ([]PlayGenreRecord, error)
	ListPlayCast(ctx context.Context, playID string) ([]PlayCastRecord, error)
	ListPlayMedia(ctx context.Context, playID string) ([]PlayMediaRecord, error)
	GetPlayMediaByID(ctx context.Context, mediaID string) (PlayMediaRecord, error)
	DeletePlayMedia(ctx context.Context, mediaID string) error
	IsPlayPublished(ctx context.Context, playID string) (bool, error)
	UserExists(ctx context.Context, userID string) (bool, error)
	ListPublishedReviews(ctx context.Context, params ListReviewsParams) ([]ReviewRecord, error)
	ListUserPublishedReviews(ctx context.Context, userID string, params ListUserReviewsParams) ([]UserReviewRecord, error)
	CreateReview(ctx context.Context, userID string, playID string, params CreateReviewParams) (ReviewRecord, error)
	GetReviewMetadata(ctx context.Context, reviewID string) (ReviewMetadataRecord, error)
	UpdateReview(ctx context.Context, reviewID string, params UpdateReviewParams) (ReviewRecord, error)
	CreateReviewComment(ctx context.Context, userID string, reviewID string, params CreateReviewCommentParams) (ReviewCommentRecord, error)
	UpdateReviewCommentStatus(ctx context.Context, commentID string, status string, updatedAt time.Time) (ReviewCommentStatusRecord, error)
	CreateSubmission(ctx context.Context, userID string, params CreateSubmissionParams) (SubmissionRecord, error)
	ListUserSubmissions(ctx context.Context, userID string, params ListSubmissionsParams) ([]SubmissionRecord, error)
	ListOwnedPublishedPlays(ctx context.Context, userID string, params ListSubmissionsParams) ([]OwnedPlayRecord, error)
	GetSubmissionByID(ctx context.Context, playID string) (SubmissionRecord, error)
	UpdateSubmission(ctx context.Context, playID string, params UpdateSubmissionParams) (SubmissionRecord, error)
	GetOwnedPublishedPlayByID(ctx context.Context, playID string, userID string) (OwnedPlayRecord, error)
	ListGenres(ctx context.Context, params ListGenresParams) ([]GenreRecord, error)
	CountGenresByIDs(ctx context.Context, genreIDs []string) (int64, error)
	CreateGenre(ctx context.Context, name string) (GenreRecord, error)
	DeleteGenre(ctx context.Context, genreID string) error
	ListAdminSubmissions(ctx context.Context, params ListSubmissionsParams) ([]SubmissionRecord, error)
	ListPlayEditSuggestions(ctx context.Context, params ListPlayEditSuggestionsParams) ([]PlayEditSuggestionRecord, error)
	GetPlayEditSuggestionByID(ctx context.Context, suggestionID string) (PlayEditSuggestionRecord, error)
	CreatePlayEditSuggestion(ctx context.Context, params CreatePlayEditSuggestionParams) (PlayEditSuggestionRecord, error)
	UpdatePlayEditSuggestion(ctx context.Context, suggestionID string, params UpdatePlayEditSuggestionParams) (PlayEditSuggestionRecord, error)
	ModeratePlayEditSuggestion(ctx context.Context, suggestionID string, params ModeratePlayEditSuggestionParams) (PlayEditSuggestionRecord, error)
	ReplacePlayEditSuggestionMedia(ctx context.Context, suggestionID string, media []CreatePlayMediaParams) error
	ListPlayEditSuggestionGenres(ctx context.Context, suggestionID string) ([]PlayGenreRecord, error)
	ListPlayEditSuggestionMedia(ctx context.Context, suggestionID string) ([]PlayMediaRecord, error)
	ReplacePlayEditSuggestionGenres(ctx context.Context, suggestionID string, genreIDs []string) error
	CreatePlayEditSuggestionMedia(ctx context.Context, suggestionID string, params CreatePlayMediaParams) (PlayMediaRecord, error)
	GetPlayEditSuggestionMediaByID(ctx context.Context, mediaID string) (PlayMediaRecord, error)
	DeletePlayEditSuggestionMedia(ctx context.Context, mediaID string) error
	ApplyApprovedPlayEditSuggestion(ctx context.Context, suggestionID string, updatedAt time.Time) error
	ApproveSubmission(ctx context.Context, playID string, adminUserID string, now time.Time) (SubmissionRecord, error)
	RejectSubmission(ctx context.Context, playID string, adminUserID string, reason string, now time.Time) (SubmissionRecord, error)
	CreatePlayMedia(ctx context.Context, params CreatePlayMediaParams) (PlayMediaRecord, error)
	SetEngagement(ctx context.Context, userID string, playID string, kind string, createdAt time.Time) error
	DeleteEngagement(ctx context.Context, userID string, playID string, kind string) error
	GetEngagementState(ctx context.Context, userID string, playID string) (EngagementStateRecord, error)
	ListUserEngagementPlays(ctx context.Context, userID string, params ListUserEngagementPlaysParams) ([]EngagementPlayRecord, error)
}

type Service struct {
	repo             repositoryPort
	mediaStorage     storage.Client
	mediaUploadTTL   time.Duration
	mediaDownloadTTL time.Duration
	maxImageBytes    int64
	playMediaBaseURL string
	imageQueue       ImageQueue
	optimizeEnabled  bool
	webpQuality      int
}

type ServiceOption func(*Service)

type ImageQueue interface {
	Enqueue(job worker.Job) bool
}

type ImageJob interface {
	Run(ctx context.Context) error
	Name() string
}

type OptimizeMediaJob struct {
	Storage      storage.Client
	ObjectKey    string
	Quality      int
	MaxWidth     int
	MaxHeight    int
	CacheControl string
}

func (j OptimizeMediaJob) Name() string {
	return "optimize-play-media"
}

func (j OptimizeMediaJob) Run(ctx context.Context) error {
	if j.Storage == nil {
		return storage.ErrClientNotConfigured
	}

	content, err := j.Storage.GetObject(ctx, storage.GetObjectInput{ObjectKey: j.ObjectKey})
	if err != nil {
		return err
	}

	result, err := imageproc.OptimizeImage(content, imageproc.OptimizeOptions{
		Quality:           j.Quality,
		MaxWidth:          j.MaxWidth,
		MaxHeight:         j.MaxHeight,
		TargetContentType: "image/webp",
	})
	if err != nil {
		return err
	}

	if !result.Optimized {
		return nil
	}

	return j.Storage.PutObject(ctx, storage.PutObjectInput{
		ObjectKey:    j.ObjectKey,
		Content:      result.Content,
		ContentType:  result.ContentType,
		CacheControl: j.CacheControl,
	})
}

func WithMediaStorage(mediaStorage storage.Client) ServiceOption {
	return func(s *Service) {
		s.mediaStorage = mediaStorage
	}
}

func WithMediaUploadTTL(ttl time.Duration) ServiceOption {
	return func(s *Service) {
		s.mediaUploadTTL = ttl
	}
}

func WithMediaDownloadTTL(ttl time.Duration) ServiceOption {
	return func(s *Service) {
		s.mediaDownloadTTL = ttl
	}
}

func WithMaxImageBytes(maxBytes int64) ServiceOption {
	return func(s *Service) {
		s.maxImageBytes = maxBytes
	}
}

func WithPlayMediaBaseURL(baseURL string) ServiceOption {
	return func(s *Service) {
		s.playMediaBaseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	}
}

func WithImageQueue(queue ImageQueue) ServiceOption {
	return func(s *Service) {
		s.imageQueue = queue
	}
}

func WithImageOptimization(enabled bool, webpQuality int) ServiceOption {
	return func(s *Service) {
		s.optimizeEnabled = enabled
		s.webpQuality = webpQuality
	}
}

func NewService(repo repositoryPort, options ...ServiceOption) *Service {
	service := &Service{
		repo:             repo,
		mediaStorage:     storage.NoopClient{},
		mediaUploadTTL:   defaultUploadURLTTL,
		mediaDownloadTTL: defaultDownloadURLTTL,
		maxImageBytes:    defaultMaxImageBytes,
		playMediaBaseURL: defaultPlayMediaPrefix,
		optimizeEnabled:  true,
		webpQuality:      80,
	}

	for _, option := range options {
		if option != nil {
			option(service)
		}
	}

	if service.mediaStorage == nil {
		service.mediaStorage = storage.NoopClient{}
	}

	if service.mediaUploadTTL <= 0 {
		service.mediaUploadTTL = defaultUploadURLTTL
	}

	if service.mediaDownloadTTL <= 0 {
		service.mediaDownloadTTL = defaultDownloadURLTTL
	}

	if service.maxImageBytes <= 0 {
		service.maxImageBytes = defaultMaxImageBytes
	}

	if strings.TrimSpace(service.playMediaBaseURL) == "" {
		service.playMediaBaseURL = defaultPlayMediaPrefix
	}

	if service.webpQuality <= 0 || service.webpQuality > 100 {
		service.webpQuality = 80
	}

	return service
}

func (s *Service) Feed(ctx context.Context, query FeedQuery) (FeedData, error) {
	section := strings.TrimSpace(strings.ToLower(query.Section))
	if section != "highlighted" && section != "trending" && section != "genre" {
		return FeedData{}, sharederrors.Validation("section must be one of: highlighted, trending, genre", nil)
	}

	var genreID *string
	rawGenreID := strings.TrimSpace(query.GenreID)
	if section == "genre" {
		if rawGenreID == "" {
			return FeedData{}, sharederrors.Validation("genreId is required when section=genre", nil)
		}

		if !isValidUUID(rawGenreID) {
			return FeedData{}, sharederrors.Validation("invalid genreId", nil)
		}

		genreID = &rawGenreID
	} else if rawGenreID != "" {
		if !isValidUUID(rawGenreID) {
			return FeedData{}, sharederrors.Validation("invalid genreId", nil)
		}
		genreID = &rawGenreID
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return FeedData{}, err
	}

	rawCursor := strings.TrimSpace(query.Cursor)

	var cursor *playListCursor
	var trendingCursor *trendingFeedCursor
	if section == "trending" {
		trendingCursor, err = decodeTrendingFeedCursor(rawCursor)
		if err != nil {
			return FeedData{}, sharederrors.Validation("invalid cursor", nil)
		}
	} else {
		cursor, err = decodePlayListCursor(rawCursor)
		if err != nil {
			return FeedData{}, sharederrors.Validation("invalid cursor", nil)
		}
	}

	records, err := s.repo.ListFeed(ctx, FeedListParams{
		Section:       section,
		GenreID:       genreID,
		After:         cursor,
		TrendingAfter: trendingCursor,
		Limit:         limit + 1,
	})
	if err != nil {
		return FeedData{}, sharederrors.Internal("failed to load feed", nil)
	}

	return buildFeedData(records, limit, section, s.playMediaBaseURL)
}

func (s *Service) Search(ctx context.Context, query SearchQuery) (SearchData, error) {
	q := strings.TrimSpace(query.Q)
	genreIDRaw := strings.TrimSpace(query.GenreID)
	city := strings.TrimSpace(query.City)
	theater := strings.TrimSpace(query.Theater)
	availabilityStatus := strings.TrimSpace(strings.ToLower(query.AvailabilityStatus))

	if q == "" && genreIDRaw == "" && city == "" && theater == "" && availabilityStatus == "" {
		return SearchData{}, sharederrors.Validation("at least one search filter must be provided", nil)
	}

	var genreID *string
	if genreIDRaw != "" {
		if !isValidUUID(genreIDRaw) {
			return SearchData{}, sharederrors.Validation("invalid genreId", nil)
		}
		genreID = &genreIDRaw
	}

	var cityFilter *string
	if city != "" {
		cityFilter = &city
	}

	var theaterFilter *string
	if theater != "" {
		theaterFilter = &theater
	}

	var availabilityFilter *string
	if availabilityStatus != "" {
		if availabilityStatus != "in_theaters" && availabilityStatus != "archive" {
			return SearchData{}, sharederrors.Validation("availabilityStatus must be one of: in_theaters, archive", nil)
		}
		availabilityFilter = &availabilityStatus
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return SearchData{}, err
	}

	cursor, err := decodePlayListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return SearchData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.SearchPublishedPlays(ctx, SearchListParams{
		Q:                  q,
		GenreID:            genreID,
		City:               cityFilter,
		Theater:            theaterFilter,
		AvailabilityStatus: availabilityFilter,
		After:              cursor,
		Limit:              limit + 1,
	})
	if err != nil {
		return SearchData{}, sharederrors.Internal("failed to search plays", nil)
	}

	data, err := buildPlayListData(records, limit, s.playMediaBaseURL)
	if err != nil {
		return SearchData{}, err
	}

	return SearchData{Items: data.Items, NextCursor: data.NextCursor}, nil
}

func (s *Service) GetByID(ctx context.Context, playID string) (PlayDetailsData, error) {
	if !isValidUUID(playID) {
		return PlayDetailsData{}, sharederrors.Validation("invalid playId", nil)
	}

	play, err := s.repo.GetPublishedPlayByID(ctx, playID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayDetailsData{}, sharederrors.NotFound("play not found", nil)
		}

		return PlayDetailsData{}, sharederrors.Internal("failed to load play", nil)
	}

	genres, err := s.repo.ListPlayGenres(ctx, playID)
	if err != nil {
		return PlayDetailsData{}, sharederrors.Internal("failed to load play", nil)
	}

	cast, err := s.repo.ListPlayCast(ctx, playID)
	if err != nil {
		return PlayDetailsData{}, sharederrors.Internal("failed to load play", nil)
	}

	media, err := s.repo.ListPlayMedia(ctx, playID)
	if err != nil {
		return PlayDetailsData{}, sharederrors.Internal("failed to load play", nil)
	}

	return mapPlayDetails(play, genres, cast, media, s.playMediaBaseURL), nil
}

func (s *Service) ListReviews(ctx context.Context, playID string, query ListReviewsQuery) (ReviewListData, error) {
	if !isValidUUID(playID) {
		return ReviewListData{}, sharederrors.Validation("invalid playId", nil)
	}

	isPublished, err := s.repo.IsPlayPublished(ctx, playID)
	if err != nil {
		return ReviewListData{}, sharederrors.Internal("failed to load reviews", nil)
	}

	if !isPublished {
		return ReviewListData{}, sharederrors.NotFound("play not found", nil)
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return ReviewListData{}, err
	}

	cursor, err := decodeReviewListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return ReviewListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListPublishedReviews(ctx, ListReviewsParams{
		PlayID: playID,
		After:  cursor,
		Limit:  limit + 1,
	})
	if err != nil {
		return ReviewListData{}, sharederrors.Internal("failed to load reviews", nil)
	}

	return buildReviewListData(records, limit)
}

func (s *Service) CreateReview(ctx context.Context, userID string, playID string, req CreateReviewRequest) (ReviewData, error) {
	if !isValidAuthUserID(userID) {
		return ReviewData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(playID) {
		return ReviewData{}, sharederrors.Validation("invalid playId", nil)
	}

	isPublished, err := s.repo.IsPlayPublished(ctx, playID)
	if err != nil {
		return ReviewData{}, sharederrors.Internal("failed to create review", nil)
	}

	if !isPublished {
		return ReviewData{}, sharederrors.NotFound("play not found", nil)
	}

	rating := req.Rating
	if rating < 1 || rating > 5 {
		return ReviewData{}, sharederrors.Validation("rating must be between 1 and 5", nil)
	}

	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ReviewData{}, sharederrors.Validation("body must not be empty", nil)
	}

	var title *string
	if req.Title != nil {
		trimmedTitle := strings.TrimSpace(*req.Title)
		if trimmedTitle != "" {
			title = &trimmedTitle
		}
	}

	containsSpoilers := false
	if req.ContainsSpoilers != nil {
		containsSpoilers = *req.ContainsSpoilers
	}

	now := time.Now().UTC()
	record, err := s.repo.CreateReview(ctx, userID, playID, CreateReviewParams{
		Rating:           rating,
		Title:            title,
		Body:             body,
		ContainsSpoilers: containsSpoilers,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		if isDuplicatedKeyError(err) {
			return ReviewData{}, sharederrors.New(http.StatusConflict, "REVIEW_ALREADY_EXISTS", "review already exists", nil)
		}

		return ReviewData{}, sharederrors.Internal("failed to create review", nil)
	}

	return mapReviewRecord(record), nil
}

func (s *Service) UpdateReview(ctx context.Context, userID string, reviewID string, req UpdateReviewRequest) (ReviewData, error) {
	if !isValidAuthUserID(userID) {
		return ReviewData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(reviewID) {
		return ReviewData{}, sharederrors.Validation("invalid reviewId", nil)
	}

	if !hasReviewPatch(req) {
		return ReviewData{}, sharederrors.Validation("at least one review field must be provided", nil)
	}

	metadata, err := s.repo.GetReviewMetadata(ctx, reviewID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ReviewData{}, sharederrors.NotFound("review not found", nil)
		}

		return ReviewData{}, sharederrors.Internal("failed to update review", nil)
	}

	if metadata.ReviewStatus != "published" || metadata.PlayCurationStatus != "published" {
		return ReviewData{}, sharederrors.NotFound("review not found", nil)
	}

	if !sameUUID(metadata.UserID, userID) {
		return ReviewData{}, sharederrors.Forbidden("you can only edit your own reviews", nil)
	}

	updateParams := UpdateReviewParams{UpdatedAt: time.Now().UTC()}

	if req.Rating != nil {
		if *req.Rating < 1 || *req.Rating > 5 {
			return ReviewData{}, sharederrors.Validation("rating must be between 1 and 5", nil)
		}

		updateParams.Rating = req.Rating
	}

	if req.Title != nil {
		updateParams.TitleProvided = true
		trimmedTitle := strings.TrimSpace(*req.Title)
		if trimmedTitle != "" {
			updateParams.Title = &trimmedTitle
		}
	}

	if req.Body != nil {
		trimmedBody := strings.TrimSpace(*req.Body)
		if trimmedBody == "" {
			return ReviewData{}, sharederrors.Validation("body must not be empty", nil)
		}

		updateParams.Body = &trimmedBody
	}

	if req.ContainsSpoilers != nil {
		updateParams.ContainsSpoilers = req.ContainsSpoilers
	}

	record, err := s.repo.UpdateReview(ctx, reviewID, updateParams)
	if err != nil {
		return ReviewData{}, sharederrors.Internal("failed to update review", nil)
	}

	return mapReviewRecord(record), nil
}

func (s *Service) CreateReviewComment(ctx context.Context, userID string, reviewID string, req CreateReviewCommentRequest) (ReviewCommentData, error) {
	if !isValidAuthUserID(userID) {
		return ReviewCommentData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(reviewID) {
		return ReviewCommentData{}, sharederrors.Validation("invalid reviewId", nil)
	}

	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ReviewCommentData{}, sharederrors.Validation("body must not be empty", nil)
	}

	metadata, err := s.repo.GetReviewMetadata(ctx, reviewID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ReviewCommentData{}, sharederrors.NotFound("review not found", nil)
		}

		return ReviewCommentData{}, sharederrors.Internal("failed to create review comment", nil)
	}

	if metadata.ReviewStatus != "published" || metadata.PlayCurationStatus != "published" {
		return ReviewCommentData{}, sharederrors.NotFound("review not found", nil)
	}

	now := time.Now().UTC()
	record, err := s.repo.CreateReviewComment(ctx, userID, reviewID, CreateReviewCommentParams{
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return ReviewCommentData{}, sharederrors.Internal("failed to create review comment", nil)
	}

	return mapReviewCommentRecord(record), nil
}

func (s *Service) UpdateReviewCommentStatus(ctx context.Context, userID string, role string, commentID string, req UpdateReviewCommentStatusRequest) (ReviewCommentStatusData, error) {
	if !isValidAuthUserID(userID) {
		return ReviewCommentStatusData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return ReviewCommentStatusData{}, err
	}

	if !isValidUUID(commentID) {
		return ReviewCommentStatusData{}, sharederrors.Validation("invalid commentId", nil)
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status != "published" && status != "hidden" {
		return ReviewCommentStatusData{}, sharederrors.Validation("status must be one of: published, hidden", nil)
	}

	record, err := s.repo.UpdateReviewCommentStatus(ctx, commentID, status, time.Now().UTC())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ReviewCommentStatusData{}, sharederrors.NotFound("review comment not found", nil)
		}

		return ReviewCommentStatusData{}, sharederrors.Internal("failed to update review comment status", nil)
	}

	return mapReviewCommentStatusRecord(record), nil
}

func (s *Service) CreateSubmission(ctx context.Context, userID string, req CreateSubmissionRequest) (SubmissionData, error) {
	if !isValidAuthUserID(userID) {
		return SubmissionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	title, err := requiredSubmissionText(req.Title, "title")
	if err != nil {
		return SubmissionData{}, err
	}

	synopsis, err := requiredSubmissionText(req.Synopsis, "synopsis")
	if err != nil {
		return SubmissionData{}, err
	}

	director, err := requiredSubmissionText(req.Director, "director")
	if err != nil {
		return SubmissionData{}, err
	}

	theaterName, err := requiredSubmissionText(req.TheaterName, "theaterName")
	if err != nil {
		return SubmissionData{}, err
	}

	if req.DurationMinutes <= 0 {
		return SubmissionData{}, sharederrors.Validation("durationMinutes must be greater than 0", nil)
	}

	availabilityStatus, err := normalizeAvailabilityStatus(req.AvailabilityStatus)
	if err != nil {
		return SubmissionData{}, err
	}

	genreIDs, err := normalizeSubmissionGenreIDs(req.GenreIDs)
	if err != nil {
		return SubmissionData{}, err
	}

	genreCount, err := s.repo.CountGenresByIDs(ctx, genreIDs)
	if err != nil {
		return SubmissionData{}, sharederrors.Internal("failed to create submission", nil)
	}

	if genreCount != int64(len(genreIDs)) {
		return SubmissionData{}, sharederrors.Validation("one or more genreIds are invalid", nil)
	}

	city := normalizeOptionalText(req.City)
	now := time.Now().UTC()
	record, err := s.repo.CreateSubmission(ctx, userID, CreateSubmissionParams{
		Title:              title,
		Synopsis:           synopsis,
		Director:           director,
		DurationMinutes:    req.DurationMinutes,
		TheaterName:        theaterName,
		City:               city,
		AvailabilityStatus: availabilityStatus,
		GenreIDs:           genreIDs,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return SubmissionData{}, sharederrors.Internal("failed to create submission", nil)
	}

	return s.mapSubmissionWithGenres(ctx, record)
}

func (s *Service) CreateSubmissionMediaUpload(ctx context.Context, userID string, playID string, req CreateSubmissionMediaUploadRequest) (CreateSubmissionMediaUploadData, error) {
	if !isValidAuthUserID(userID) {
		return CreateSubmissionMediaUploadData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(playID) {
		return CreateSubmissionMediaUploadData{}, sharederrors.Validation("invalid playId", nil)
	}

	if _, err := normalizePlayMediaKind(req.Kind); err != nil {
		return CreateSubmissionMediaUploadData{}, err
	}

	contentType, _, err := normalizeImageContentType(req.ContentType)
	if err != nil {
		return CreateSubmissionMediaUploadData{}, err
	}

	if req.ContentLength <= 0 {
		return CreateSubmissionMediaUploadData{}, sharederrors.Validation("contentLength must be greater than 0", nil)
	}

	if req.ContentLength > s.maxImageBytes {
		return CreateSubmissionMediaUploadData{}, sharederrors.Validation("contentLength exceeds max allowed size", nil)
	}

	current, err := s.repo.GetSubmissionByID(ctx, playID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreateSubmissionMediaUploadData{}, sharederrors.NotFound("submission not found", nil)
		}

		return CreateSubmissionMediaUploadData{}, sharederrors.Internal("failed to create media upload", nil)
	}

	if !sameUUID(current.CreatedByUserID, userID) {
		return CreateSubmissionMediaUploadData{}, sharederrors.Forbidden("you can only edit your own submissions", nil)
	}

	if current.CurationStatus != "pending" && current.CurationStatus != "rejected" {
		return CreateSubmissionMediaUploadData{}, invalidTransitionError(current.CurationStatus, "pending")
	}

	objectKey := fmt.Sprintf("plays/%s/%s.webp", playID, uuid.NewString())
	contentLength := req.ContentLength
	uploadURL, err := s.mediaStorage.PresignPutObject(ctx, storage.PresignPutObjectInput{
		ObjectKey:     objectKey,
		ContentType:   contentType,
		ContentLength: &contentLength,
		ExpiresIn:     s.mediaUploadTTL,
	})
	if err != nil {
		return CreateSubmissionMediaUploadData{}, sharederrors.Internal("failed to create media upload", nil)
	}

	return CreateSubmissionMediaUploadData{ObjectKey: objectKey, UploadURL: uploadURL}, nil
}

func (s *Service) AttachSubmissionMedia(ctx context.Context, userID string, playID string, req AttachSubmissionMediaRequest) (PlayMediaData, error) {
	if !isValidAuthUserID(userID) {
		return PlayMediaData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(playID) {
		return PlayMediaData{}, sharederrors.Validation("invalid playId", nil)
	}

	kind, err := normalizePlayMediaKind(req.Kind)
	if err != nil {
		return PlayMediaData{}, err
	}

	objectKey := strings.TrimSpace(req.ObjectKey)
	if objectKey == "" {
		return PlayMediaData{}, sharederrors.Validation("objectKey must not be empty", nil)
	}

	if !strings.HasPrefix(objectKey, "plays/"+playID+"/") {
		return PlayMediaData{}, sharederrors.Validation("objectKey does not belong to this play", nil)
	}

	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
		if sortOrder < 0 {
			return PlayMediaData{}, sharederrors.Validation("sortOrder must be greater than or equal to 0", nil)
		}
	}

	altText := normalizeOptionalText(req.AltText)

	current, err := s.repo.GetSubmissionByID(ctx, playID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayMediaData{}, sharederrors.NotFound("submission not found", nil)
		}

		return PlayMediaData{}, sharederrors.Internal("failed to attach media", nil)
	}

	if !sameUUID(current.CreatedByUserID, userID) {
		return PlayMediaData{}, sharederrors.Forbidden("you can only edit your own submissions", nil)
	}

	if current.CurationStatus != "pending" && current.CurationStatus != "rejected" {
		return PlayMediaData{}, invalidTransitionError(current.CurationStatus, "pending")
	}

	headObject, err := s.mediaStorage.HeadObject(ctx, storage.HeadObjectInput{ObjectKey: objectKey})
	if err != nil {
		if storage.IsNotFoundError(err) {
			return PlayMediaData{}, sharederrors.Validation("uploaded object was not found", nil)
		}

		return PlayMediaData{}, sharederrors.Internal("failed to attach media", nil)
	}

	if _, _, err := normalizeImageContentType(headObject.ContentType); err != nil {
		return PlayMediaData{}, err
	}

	if headObject.ContentLength <= 0 || headObject.ContentLength > s.maxImageBytes {
		return PlayMediaData{}, sharederrors.Validation("uploaded object size is invalid", nil)
	}

	optimizedObjectKey, err := s.optimizeAndStorePlayMedia(ctx, objectKey)
	if err != nil {
		return PlayMediaData{}, err
	}

	record, err := s.repo.CreatePlayMedia(ctx, CreatePlayMediaParams{
		PlayID:    playID,
		Kind:      kind,
		ObjectKey: optimizedObjectKey,
		AltText:   altText,
		SortOrder: sortOrder,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		if isDuplicatedKeyError(err) {
			return PlayMediaData{}, sharederrors.New(http.StatusConflict, "PLAY_MEDIA_ALREADY_EXISTS", "play media already exists", nil)
		}

		return PlayMediaData{}, sharederrors.Internal("failed to attach media", nil)
	}

	return mapPlayMediaRecord(playID, record, s.playMediaBaseURL), nil
}

func (s *Service) ListMySubmissions(ctx context.Context, userID string, query ListSubmissionsQuery) (SubmissionListData, error) {
	if !isValidAuthUserID(userID) {
		return SubmissionListData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	statusFilter, err := normalizeSubmissionStatus(query.Status, false)
	if err != nil {
		return SubmissionListData{}, err
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return SubmissionListData{}, err
	}

	cursor, err := decodeSubmissionListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return SubmissionListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListUserSubmissions(ctx, userID, ListSubmissionsParams{Status: statusFilter, After: cursor, Limit: limit + 1})
	if err != nil {
		return SubmissionListData{}, sharederrors.Internal("failed to load submissions", nil)
	}

	return buildSubmissionListData(records, limit)
}

func (s *Service) ListMyBookmarks(ctx context.Context, userID string, query ListMyEngagementsQuery) (MyEngagementPlayListData, error) {
	return s.listMyEngagementPlays(ctx, userID, query, "wishlist", "failed to load bookmarks")
}

func (s *Service) ListMyWatched(ctx context.Context, userID string, query ListMyEngagementsQuery) (MyEngagementPlayListData, error) {
	return s.listMyEngagementPlays(ctx, userID, query, "attended", "failed to load watched plays")
}

func (s *Service) ListUserWatched(ctx context.Context, userID string, query ListMyEngagementsQuery) (MyEngagementPlayListData, error) {
	if !isValidUUID(userID) {
		return MyEngagementPlayListData{}, sharederrors.Validation("invalid userId", nil)
	}

	if err := s.ensureUserExists(ctx, userID); err != nil {
		return MyEngagementPlayListData{}, err
	}

	return s.listEngagementPlays(ctx, userID, query, "attended", "failed to load watched plays")
}

func (s *Service) ListUserReviews(ctx context.Context, userID string, query ListUserReviewsQuery) (UserReviewListData, error) {
	if !isValidUUID(userID) {
		return UserReviewListData{}, sharederrors.Validation("invalid userId", nil)
	}

	if err := s.ensureUserExists(ctx, userID); err != nil {
		return UserReviewListData{}, err
	}

	return s.listUserReviews(ctx, userID, query)
}

func (s *Service) ListMyReviews(ctx context.Context, userID string, query ListUserReviewsQuery) (UserReviewListData, error) {
	if !isValidAuthUserID(userID) {
		return UserReviewListData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := s.ensureUserExists(ctx, userID); err != nil {
		return UserReviewListData{}, err
	}

	return s.listUserReviews(ctx, userID, query)
}

func (s *Service) UpdateMySubmission(ctx context.Context, userID string, playID string, req UpdateSubmissionRequest) (SubmissionData, error) {
	if !isValidAuthUserID(userID) {
		return SubmissionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(playID) {
		return SubmissionData{}, sharederrors.Validation("invalid playId", nil)
	}

	if !hasSubmissionPatch(req) {
		return SubmissionData{}, sharederrors.Validation("at least one submission field must be provided", nil)
	}

	current, err := s.repo.GetSubmissionByID(ctx, playID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmissionData{}, sharederrors.NotFound("submission not found", nil)
		}

		return SubmissionData{}, sharederrors.Internal("failed to update submission", nil)
	}

	if !sameUUID(current.CreatedByUserID, userID) {
		return SubmissionData{}, sharederrors.Forbidden("you can only edit your own submissions", nil)
	}

	if current.CurationStatus != "pending" && current.CurationStatus != "rejected" {
		return SubmissionData{}, invalidTransitionError(current.CurationStatus, "pending")
	}

	patch, err := validateSubmissionPatch(req)
	if err != nil {
		return SubmissionData{}, err
	}

	patch.UpdatedAt = time.Now().UTC()
	if current.CurationStatus == "rejected" {
		patch.SetPendingResubmit = true
		patch.ClearModerationAudit = true
	}

	record, err := s.repo.UpdateSubmission(ctx, playID, patch)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmissionData{}, sharederrors.NotFound("submission not found", nil)
		}

		return SubmissionData{}, sharederrors.Internal("failed to update submission", nil)
	}

	return s.mapSubmissionWithGenres(ctx, record)
}

func (s *Service) ListMyOwnedPlays(ctx context.Context, userID string, query ListOwnedPlaysQuery) (OwnedPlayListData, error) {
	if !isValidAuthUserID(userID) {
		return OwnedPlayListData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return OwnedPlayListData{}, err
	}

	cursor, err := decodeSubmissionListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return OwnedPlayListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListOwnedPublishedPlays(ctx, userID, ListSubmissionsParams{After: cursor, Limit: limit + 1})
	if err != nil {
		return OwnedPlayListData{}, sharederrors.Internal("failed to load owned plays", nil)
	}

	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]OwnedPlayData, 0, len(records))
	for _, record := range records {
		item := OwnedPlayData{
			ID:                 record.ID,
			Title:              record.Title,
			TheaterName:        record.TheaterName,
			City:               record.City,
			AvailabilityStatus: record.AvailabilityStatus,
			PublishedAt:        record.PublishedAt.UTC().Format(time.RFC3339Nano),
			PosterURL:          buildPosterURL(s.playMediaBaseURL, record.ID, record.PosterMediaID),
		}

		pendingStatus := "pending"
		suggestionRecords, err := s.repo.ListPlayEditSuggestions(ctx, ListPlayEditSuggestionsParams{
			CreatedByUserID: &userID,
			PlayID:          &record.ID,
			Status:          &pendingStatus,
			Limit:           1,
		})
		if err == nil && len(suggestionRecords) == 0 {
			suggestionRecords, err = s.repo.ListPlayEditSuggestions(ctx, ListPlayEditSuggestionsParams{
				CreatedByUserID: &userID,
				PlayID:          &record.ID,
				Limit:           1,
			})
		}
		if err == nil && len(suggestionRecords) > 0 {
			latest := suggestionRecords[0]
			item.LatestEditSuggestion = &PlayEditSuggestionSummaryData{
				ID:        latest.ID,
				Status:    latest.Status,
				CreatedAt: latest.CreatedAt.UTC().Format(time.RFC3339Nano),
				UpdatedAt: latest.UpdatedAt.UTC().Format(time.RFC3339Nano),
				PlayID:    latest.PlayID,
				Title:     latest.Title,
				City:      latest.City,
			}
		}

		items = append(items, item)
	}

	response := OwnedPlayListData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor, err := encodeSubmissionListCursor(submissionListCursor{CreatedAt: last.CreatedAt.UTC(), PlayID: last.ID})
		if err != nil {
			return OwnedPlayListData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}
		response.NextCursor = &nextCursor
	}

	return response, nil
}

func (s *Service) CreatePlayEditSuggestion(ctx context.Context, userID string, req CreatePlayEditSuggestionRequest) (PlayEditSuggestionData, error) {
	if !isValidAuthUserID(userID) {
		return PlayEditSuggestionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	playID := strings.TrimSpace(req.PlayID)
	if !isValidUUID(playID) {
		return PlayEditSuggestionData{}, sharederrors.Validation("invalid playId", nil)
	}

	ownedPlay, err := s.repo.GetOwnedPublishedPlayByID(ctx, playID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayEditSuggestionData{}, sharederrors.NotFound("owned published play not found", nil)
		}
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to create play edit suggestion", nil)
	}

	current, err := s.repo.GetSubmissionByID(ctx, playID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayEditSuggestionData{}, sharederrors.NotFound("play not found", nil)
		}
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to create play edit suggestion", nil)
	}
	if current.CurationStatus != "published" {
		return PlayEditSuggestionData{}, sharederrors.Validation("play must be published", nil)
	}

	currentGenres, err := s.repo.ListPlayGenres(ctx, playID)
	if err != nil {
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to create play edit suggestion", nil)
	}
	currentMedia, err := s.repo.ListPlayMedia(ctx, playID)
	if err != nil {
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to create play edit suggestion", nil)
	}

	title := current.Title
	if req.Title != nil {
		title, err = requiredSubmissionText(*req.Title, "title")
		if err != nil {
			return PlayEditSuggestionData{}, err
		}
	}
	synopsis := current.Synopsis
	if req.Synopsis != nil {
		synopsis, err = requiredSubmissionText(*req.Synopsis, "synopsis")
		if err != nil {
			return PlayEditSuggestionData{}, err
		}
	}
	director := current.Director
	if req.Director != nil {
		director, err = requiredSubmissionText(*req.Director, "director")
		if err != nil {
			return PlayEditSuggestionData{}, err
		}
	}
	durationMinutes := current.DurationMinutes
	if req.DurationMinutes != nil {
		if *req.DurationMinutes <= 0 {
			return PlayEditSuggestionData{}, sharederrors.Validation("durationMinutes must be greater than 0", nil)
		}
		durationMinutes = *req.DurationMinutes
	}
	theaterName := current.TheaterName
	if req.TheaterName != nil {
		theaterName, err = requiredSubmissionText(*req.TheaterName, "theaterName")
		if err != nil {
			return PlayEditSuggestionData{}, err
		}
	}
	city := current.City
	if req.City != nil {
		trimmed := strings.TrimSpace(*req.City)
		if trimmed == "" {
			city = nil
		} else {
			city = &trimmed
		}
	}
	availabilityStatus := current.AvailabilityStatus
	if req.AvailabilityStatus != nil {
		availabilityStatus, err = normalizeAvailabilityStatus(req.AvailabilityStatus)
		if err != nil {
			return PlayEditSuggestionData{}, err
		}
	}

	genreIDs := make([]string, 0, len(currentGenres))
	for _, genre := range currentGenres {
		genreIDs = append(genreIDs, genre.ID)
	}
	if req.GenreIDs != nil {
		genreIDs, err = normalizeSubmissionGenreIDs(req.GenreIDs)
		if err != nil {
			return PlayEditSuggestionData{}, err
		}
	}

	genreCount, err := s.repo.CountGenresByIDs(ctx, genreIDs)
	if err != nil {
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to validate genres", nil)
	}
	if int64(len(genreIDs)) != genreCount {
		return PlayEditSuggestionData{}, sharederrors.Validation("genreIds contains unknown genres", nil)
	}

	now := time.Now().UTC()
	created, err := s.repo.CreatePlayEditSuggestion(ctx, CreatePlayEditSuggestionParams{
		PlayID:             ownedPlay.ID,
		CreatedByUserID:    userID,
		Title:              title,
		Synopsis:           synopsis,
		Director:           director,
		DurationMinutes:    durationMinutes,
		TheaterName:        theaterName,
		City:               city,
		AvailabilityStatus: availabilityStatus,
		GenreIDs:           genreIDs,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to create play edit suggestion", nil)
	}

	mediaParams := make([]CreatePlayMediaParams, 0, len(currentMedia))
	for _, media := range currentMedia {
		mediaParams = append(mediaParams, CreatePlayMediaParams{
			Kind:      media.Kind,
			ObjectKey: media.ObjectKey,
			AltText:   media.AltText,
			SortOrder: media.SortOrder,
			CreatedAt: now,
		})
	}
	if err := s.repo.ReplacePlayEditSuggestionMedia(ctx, created.ID, mediaParams); err != nil {
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to initialize play edit suggestion media", nil)
	}

	return s.getPlayEditSuggestionData(ctx, created.ID)
}

func (s *Service) ListMyPlayEditSuggestions(ctx context.Context, userID string, query ListPlayEditSuggestionsQuery) (PlayEditSuggestionListData, error) {
	if !isValidAuthUserID(userID) {
		return PlayEditSuggestionListData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	status, err := normalizePlayEditSuggestionStatus(query.Status, false)
	if err != nil {
		return PlayEditSuggestionListData{}, err
	}
	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return PlayEditSuggestionListData{}, err
	}
	cursor, err := decodePlayEditSuggestionListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return PlayEditSuggestionListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListPlayEditSuggestions(ctx, ListPlayEditSuggestionsParams{
		CreatedByUserID: &userID,
		Status:          status,
		After:           cursor,
		Limit:           limit + 1,
	})
	if err != nil {
		return PlayEditSuggestionListData{}, sharederrors.Internal("failed to load play edit suggestions", nil)
	}

	return s.buildPlayEditSuggestionListData(ctx, records, limit)
}

func (s *Service) GetMyPlayEditSuggestionByID(ctx context.Context, userID string, suggestionID string) (PlayEditSuggestionData, error) {
	if !isValidAuthUserID(userID) {
		return PlayEditSuggestionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	record, err := s.repo.GetPlayEditSuggestionByID(ctx, suggestionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayEditSuggestionData{}, sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to load play edit suggestion", nil)
	}
	if !sameUUID(record.CreatedByUserID, userID) {
		return PlayEditSuggestionData{}, sharederrors.Forbidden("you can only view your own play edit suggestions", nil)
	}

	return s.mapPlayEditSuggestionWithDetails(ctx, record)
}

func (s *Service) UpdateMyPlayEditSuggestion(ctx context.Context, userID string, suggestionID string, req UpdatePlayEditSuggestionRequest) (PlayEditSuggestionData, error) {
	if !isValidAuthUserID(userID) {
		return PlayEditSuggestionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}
	if !isValidUUID(suggestionID) {
		return PlayEditSuggestionData{}, sharederrors.Validation("invalid suggestionId", nil)
	}

	current, err := s.repo.GetPlayEditSuggestionByID(ctx, suggestionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayEditSuggestionData{}, sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to update play edit suggestion", nil)
	}
	if !sameUUID(current.CreatedByUserID, userID) {
		return PlayEditSuggestionData{}, sharederrors.Forbidden("you can only edit your own play edit suggestions", nil)
	}
	if current.Status != "pending" {
		return PlayEditSuggestionData{}, sharederrors.Validation("only pending play edit suggestions can be updated", nil)
	}

	patch, err := validatePlayEditSuggestionPatch(req)
	if err != nil {
		return PlayEditSuggestionData{}, err
	}
	patch.UpdatedAt = time.Now().UTC()

	if patch.GenreIDsProvided {
		count, err := s.repo.CountGenresByIDs(ctx, patch.GenreIDs)
		if err != nil {
			return PlayEditSuggestionData{}, sharederrors.Internal("failed to validate genres", nil)
		}
		if int64(len(patch.GenreIDs)) != count {
			return PlayEditSuggestionData{}, sharederrors.Validation("genreIds contains unknown genres", nil)
		}
	}

	updated, err := s.repo.UpdatePlayEditSuggestion(ctx, suggestionID, patch)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayEditSuggestionData{}, sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to update play edit suggestion", nil)
	}

	return s.mapPlayEditSuggestionWithDetails(ctx, updated)
}

func (s *Service) ListAdminSubmissions(ctx context.Context, userID string, role string, query ListSubmissionsQuery) (SubmissionListData, error) {
	if !isValidAuthUserID(userID) {
		return SubmissionListData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return SubmissionListData{}, err
	}

	statusFilter, err := normalizeSubmissionStatus(query.Status, true)
	if err != nil {
		return SubmissionListData{}, err
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return SubmissionListData{}, err
	}

	cursor, err := decodeSubmissionListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return SubmissionListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListAdminSubmissions(ctx, ListSubmissionsParams{Status: statusFilter, After: cursor, Limit: limit + 1})
	if err != nil {
		return SubmissionListData{}, sharederrors.Internal("failed to load submissions", nil)
	}

	return buildSubmissionListData(records, limit)
}

func (s *Service) ListGenres(ctx context.Context, query ListGenresQuery) (GenreListData, error) {
	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return GenreListData{}, err
	}

	cursor, err := decodeGenreListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return GenreListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListGenres(ctx, ListGenresParams{After: cursor, Limit: limit + 1})
	if err != nil {
		return GenreListData{}, sharederrors.Internal("failed to load genres", nil)
	}

	return buildGenreListData(records, limit)
}

func (s *Service) ListAdminGenres(ctx context.Context, userID string, role string, query ListGenresQuery) (GenreListData, error) {
	if !isValidAuthUserID(userID) {
		return GenreListData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return GenreListData{}, err
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return GenreListData{}, err
	}

	cursor, err := decodeGenreListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return GenreListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListGenres(ctx, ListGenresParams{After: cursor, Limit: limit + 1})
	if err != nil {
		return GenreListData{}, sharederrors.Internal("failed to load genres", nil)
	}

	return buildGenreListData(records, limit)
}

func (s *Service) CreateAdminGenre(ctx context.Context, userID string, role string, req CreateGenreRequest) (GenreData, error) {
	if !isValidAuthUserID(userID) {
		return GenreData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return GenreData{}, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return GenreData{}, sharederrors.Validation("name must not be empty", nil)
	}

	record, err := s.repo.CreateGenre(ctx, name)
	if err != nil {
		if isDuplicatedKeyError(err) {
			return GenreData{}, sharederrors.New(http.StatusConflict, "GENRE_ALREADY_EXISTS", "genre already exists", nil)
		}

		return GenreData{}, sharederrors.Internal("failed to create genre", nil)
	}

	return mapGenreRecord(record), nil
}

func (s *Service) DeleteAdminGenre(ctx context.Context, userID string, role string, genreID string) error {
	if !isValidAuthUserID(userID) {
		return sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return err
	}

	if !isValidUUID(genreID) {
		return sharederrors.Validation("invalid genreId", nil)
	}

	err := s.repo.DeleteGenre(ctx, genreID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sharederrors.NotFound("genre not found", nil)
		}

		if isForeignKeyViolationError(err) {
			return sharederrors.New(http.StatusConflict, "GENRE_IN_USE", "genre is in use by plays", nil)
		}

		return sharederrors.Internal("failed to delete genre", nil)
	}

	return nil
}

func (s *Service) GetAdminSubmissionByID(ctx context.Context, userID string, role string, playID string) (SubmissionData, error) {
	if !isValidAuthUserID(userID) {
		return SubmissionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return SubmissionData{}, err
	}

	if !isValidUUID(playID) {
		return SubmissionData{}, sharederrors.Validation("invalid playId", nil)
	}

	record, err := s.repo.GetSubmissionByID(ctx, playID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmissionData{}, sharederrors.NotFound("submission not found", nil)
		}

		return SubmissionData{}, sharederrors.Internal("failed to load submission", nil)
	}

	return s.mapSubmissionWithGenresAndMedia(ctx, record)
}

func (s *Service) UpdateAdminSubmission(ctx context.Context, userID string, role string, playID string, req UpdateSubmissionRequest) (SubmissionData, error) {
	if !isValidAuthUserID(userID) {
		return SubmissionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return SubmissionData{}, err
	}

	if !isValidUUID(playID) {
		return SubmissionData{}, sharederrors.Validation("invalid playId", nil)
	}

	if !hasSubmissionPatch(req) {
		return SubmissionData{}, sharederrors.Validation("at least one submission field must be provided", nil)
	}

	current, err := s.repo.GetSubmissionByID(ctx, playID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmissionData{}, sharederrors.NotFound("submission not found", nil)
		}

		return SubmissionData{}, sharederrors.Internal("failed to update submission", nil)
	}

	if current.CurationStatus != "pending" {
		return SubmissionData{}, invalidTransitionError(current.CurationStatus, "pending")
	}

	patch, err := validateSubmissionPatch(req)
	if err != nil {
		return SubmissionData{}, err
	}

	patch.UpdatedAt = time.Now().UTC()

	record, err := s.repo.UpdateSubmission(ctx, playID, patch)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmissionData{}, sharederrors.NotFound("submission not found", nil)
		}

		return SubmissionData{}, sharederrors.Internal("failed to update submission", nil)
	}

	return s.mapSubmissionWithGenresAndMedia(ctx, record)
}

func (s *Service) ApproveSubmission(ctx context.Context, userID string, role string, playID string) (SubmissionData, error) {
	if !isValidAuthUserID(userID) {
		return SubmissionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return SubmissionData{}, err
	}

	if !isValidUUID(playID) {
		return SubmissionData{}, sharederrors.Validation("invalid playId", nil)
	}

	current, err := s.repo.GetSubmissionByID(ctx, playID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmissionData{}, sharederrors.NotFound("submission not found", nil)
		}

		return SubmissionData{}, sharederrors.Internal("failed to approve submission", nil)
	}

	if current.CurationStatus != "pending" {
		return SubmissionData{}, invalidTransitionError(current.CurationStatus, "published")
	}

	record, err := s.repo.ApproveSubmission(ctx, playID, userID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmissionData{}, sharederrors.NotFound("submission not found", nil)
		}

		return SubmissionData{}, sharederrors.Internal("failed to approve submission", nil)
	}

	return s.mapSubmissionWithGenresAndMedia(ctx, record)
}

func (s *Service) RejectSubmission(ctx context.Context, userID string, role string, playID string, req RejectSubmissionRequest) (SubmissionData, error) {
	if !isValidAuthUserID(userID) {
		return SubmissionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return SubmissionData{}, err
	}

	if !isValidUUID(playID) {
		return SubmissionData{}, sharederrors.Validation("invalid playId", nil)
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return SubmissionData{}, sharederrors.Validation("reason must not be empty", nil)
	}

	current, err := s.repo.GetSubmissionByID(ctx, playID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmissionData{}, sharederrors.NotFound("submission not found", nil)
		}

		return SubmissionData{}, sharederrors.Internal("failed to reject submission", nil)
	}

	if current.CurationStatus != "pending" {
		return SubmissionData{}, invalidTransitionError(current.CurationStatus, "rejected")
	}

	record, err := s.repo.RejectSubmission(ctx, playID, userID, reason, time.Now().UTC())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmissionData{}, sharederrors.NotFound("submission not found", nil)
		}

		return SubmissionData{}, sharederrors.Internal("failed to reject submission", nil)
	}

	return s.mapSubmissionWithGenresAndMedia(ctx, record)
}

func (s *Service) CreatePlayEditSuggestionMediaUpload(ctx context.Context, userID string, suggestionID string, req CreateSubmissionMediaUploadRequest) (CreateSubmissionMediaUploadData, error) {
	if !isValidAuthUserID(userID) {
		return CreateSubmissionMediaUploadData{}, sharederrors.Unauthorized("invalid access token", nil)
	}
	if !isValidUUID(suggestionID) {
		return CreateSubmissionMediaUploadData{}, sharederrors.Validation("invalid suggestionId", nil)
	}
	record, err := s.repo.GetPlayEditSuggestionByID(ctx, suggestionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreateSubmissionMediaUploadData{}, sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return CreateSubmissionMediaUploadData{}, sharederrors.Internal("failed to create media upload", nil)
	}
	if !sameUUID(record.CreatedByUserID, userID) {
		return CreateSubmissionMediaUploadData{}, sharederrors.Forbidden("you can only edit your own play edit suggestions", nil)
	}
	if record.Status != "pending" {
		return CreateSubmissionMediaUploadData{}, sharederrors.Validation("only pending play edit suggestions can be modified", nil)
	}

	if _, err := normalizePlayMediaKind(req.Kind); err != nil {
		return CreateSubmissionMediaUploadData{}, err
	}
	contentType, _, err := normalizeImageContentType(req.ContentType)
	if err != nil {
		return CreateSubmissionMediaUploadData{}, err
	}
	if req.ContentLength <= 0 {
		return CreateSubmissionMediaUploadData{}, sharederrors.Validation("contentLength must be greater than 0", nil)
	}
	if req.ContentLength > s.maxImageBytes {
		return CreateSubmissionMediaUploadData{}, sharederrors.Validation("contentLength exceeds max allowed size", nil)
	}

	objectKey := fmt.Sprintf("play-edit-suggestions/%s/%s.webp", suggestionID, uuid.NewString())
	contentLength := req.ContentLength
	uploadURL, err := s.mediaStorage.PresignPutObject(ctx, storage.PresignPutObjectInput{
		ObjectKey:     objectKey,
		ContentType:   contentType,
		ContentLength: &contentLength,
		ExpiresIn:     s.mediaUploadTTL,
	})
	if err != nil {
		return CreateSubmissionMediaUploadData{}, sharederrors.Internal("failed to create media upload", nil)
	}

	if s.imageQueue != nil && s.optimizeEnabled {
		_ = s.imageQueue.Enqueue(OptimizeMediaJob{
			Storage:      s.mediaStorage,
			ObjectKey:    objectKey,
			Quality:      s.webpQuality,
			MaxWidth:     playMediaMaxWidth,
			MaxHeight:    playMediaMaxHeight,
			CacheControl: "public, max-age=31536000, immutable",
		})
	}

	return CreateSubmissionMediaUploadData{ObjectKey: objectKey, UploadURL: uploadURL}, nil
}

func (s *Service) AttachPlayEditSuggestionMedia(ctx context.Context, userID string, suggestionID string, req AttachSubmissionMediaRequest) (PlayMediaData, error) {
	if !isValidAuthUserID(userID) {
		return PlayMediaData{}, sharederrors.Unauthorized("invalid access token", nil)
	}
	if !isValidUUID(suggestionID) {
		return PlayMediaData{}, sharederrors.Validation("invalid suggestionId", nil)
	}
	record, err := s.repo.GetPlayEditSuggestionByID(ctx, suggestionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayMediaData{}, sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return PlayMediaData{}, sharederrors.Internal("failed to attach media", nil)
	}
	if !sameUUID(record.CreatedByUserID, userID) {
		return PlayMediaData{}, sharederrors.Forbidden("you can only edit your own play edit suggestions", nil)
	}
	if record.Status != "pending" {
		return PlayMediaData{}, sharederrors.Validation("only pending play edit suggestions can be modified", nil)
	}

	kind, err := normalizePlayMediaKind(req.Kind)
	if err != nil {
		return PlayMediaData{}, err
	}
	objectKey := strings.TrimSpace(req.ObjectKey)
	if objectKey == "" {
		return PlayMediaData{}, sharederrors.Validation("objectKey must not be empty", nil)
	}
	if !strings.HasPrefix(objectKey, "play-edit-suggestions/"+suggestionID+"/") {
		return PlayMediaData{}, sharederrors.Validation("objectKey does not belong to this suggestion", nil)
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
		if sortOrder < 0 {
			return PlayMediaData{}, sharederrors.Validation("sortOrder must be greater than or equal to 0", nil)
		}
	}
	altText := normalizeOptionalText(req.AltText)

	headObject, err := s.mediaStorage.HeadObject(ctx, storage.HeadObjectInput{ObjectKey: objectKey})
	if err != nil {
		if storage.IsNotFoundError(err) {
			return PlayMediaData{}, sharederrors.Validation("uploaded object was not found", nil)
		}
		return PlayMediaData{}, sharederrors.Internal("failed to attach media", nil)
	}
	if _, _, err := normalizeImageContentType(headObject.ContentType); err != nil {
		return PlayMediaData{}, err
	}
	if headObject.ContentLength <= 0 || headObject.ContentLength > s.maxImageBytes {
		return PlayMediaData{}, sharederrors.Validation("uploaded object size is invalid", nil)
	}

	optimizedObjectKey, err := s.optimizeAndStorePlayMedia(ctx, objectKey)
	if err != nil {
		return PlayMediaData{}, err
	}

	mediaRecord, err := s.repo.CreatePlayEditSuggestionMedia(ctx, suggestionID, CreatePlayMediaParams{
		Kind:      kind,
		ObjectKey: optimizedObjectKey,
		AltText:   altText,
		SortOrder: sortOrder,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		if isDuplicatedKeyError(err) {
			return PlayMediaData{}, sharederrors.New(http.StatusConflict, "PLAY_MEDIA_ALREADY_EXISTS", "play media already exists", nil)
		}
		return PlayMediaData{}, sharederrors.Internal("failed to attach media", nil)
	}

	mediaURL := mediaRecord.ObjectKey
	if presignedURL, presignErr := s.mediaStorage.PresignGetObject(ctx, storage.PresignGetObjectInput{
		ObjectKey: mediaRecord.ObjectKey,
		ExpiresIn: s.mediaDownloadTTL,
	}); presignErr == nil && presignedURL != "" {
		mediaURL = presignedURL
	}

	return PlayMediaData{
		ID:        mediaRecord.ID,
		Kind:      mediaRecord.Kind,
		URL:       mediaURL,
		AltText:   mediaRecord.AltText,
		SortOrder: mediaRecord.SortOrder,
	}, nil
}

func (s *Service) DeleteMyPlayEditSuggestionMedia(ctx context.Context, userID string, suggestionID string, mediaID string) error {
	if !isValidAuthUserID(userID) {
		return sharederrors.Unauthorized("invalid access token", nil)
	}
	if !isValidUUID(suggestionID) {
		return sharederrors.Validation("invalid suggestionId", nil)
	}
	if !isValidUUID(mediaID) {
		return sharederrors.Validation("invalid mediaId", nil)
	}
	record, err := s.repo.GetPlayEditSuggestionByID(ctx, suggestionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return sharederrors.Internal("failed to delete media", nil)
	}
	if !sameUUID(record.CreatedByUserID, userID) {
		return sharederrors.Forbidden("you can only edit your own play edit suggestions", nil)
	}
	if record.Status != "pending" {
		return sharederrors.Validation("only pending play edit suggestions can be modified", nil)
	}

	media, err := s.repo.GetPlayEditSuggestionMediaByID(ctx, mediaID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sharederrors.NotFound("media not found", nil)
		}
		return sharederrors.Internal("failed to delete media", nil)
	}
	if media.PlayID != suggestionID {
		return sharederrors.NotFound("media not found", nil)
	}
	if err := s.repo.DeletePlayEditSuggestionMedia(ctx, mediaID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sharederrors.NotFound("media not found", nil)
		}
		return sharederrors.Internal("failed to delete media", nil)
	}
	return nil
}

func (s *Service) ListAdminModerationQueue(ctx context.Context, userID string, role string, query ListModerationQueueQuery) (ModerationQueueData, error) {
	if !isValidAuthUserID(userID) {
		return ModerationQueueData{}, sharederrors.Unauthorized("invalid access token", nil)
	}
	if err := requireAdminRole(role); err != nil {
		return ModerationQueueData{}, err
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return ModerationQueueData{}, err
	}

	submissionStatus := "pending"
	submissions, err := s.repo.ListAdminSubmissions(ctx, ListSubmissionsParams{Status: &submissionStatus, Limit: limit + 1})
	if err != nil {
		return ModerationQueueData{}, sharederrors.Internal("failed to load moderation queue", nil)
	}
	suggestionsStatus := "pending"
	suggestions, err := s.repo.ListPlayEditSuggestions(ctx, ListPlayEditSuggestionsParams{Status: &suggestionsStatus, Limit: limit + 1})
	if err != nil {
		return ModerationQueueData{}, sharederrors.Internal("failed to load moderation queue", nil)
	}

	merged := make([]ModerationQueueRecord, 0, len(submissions)+len(suggestions))
	for _, submission := range submissions {
		status := submission.CurationStatus
		merged = append(merged, ModerationQueueRecord{
			ItemType:         "submission",
			ItemID:           submission.ID,
			PlayID:           submission.ID,
			Title:            submission.Title,
			TheaterName:      submission.TheaterName,
			City:             submission.City,
			CreatedAt:        submission.CreatedAt,
			CreatedByUserID:  submission.CreatedByUserID,
			SubmissionStatus: &status,
		})
	}
	for _, suggestion := range suggestions {
		status := suggestion.Status
		merged = append(merged, ModerationQueueRecord{
			ItemType:         "edit_suggestion",
			ItemID:           suggestion.ID,
			PlayID:           suggestion.PlayID,
			Title:            suggestion.Title,
			TheaterName:      suggestion.TheaterName,
			City:             suggestion.City,
			CreatedAt:        suggestion.CreatedAt,
			CreatedByUserID:  suggestion.CreatedByUserID,
			SuggestionStatus: &status,
		})
	}

	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].CreatedAt.Equal(merged[j].CreatedAt) {
			return merged[i].ItemID > merged[j].ItemID
		}
		return merged[i].CreatedAt.After(merged[j].CreatedAt)
	})

	hasNext := len(merged) > limit
	if hasNext {
		merged = merged[:limit]
	}

	items := make([]ModerationQueueItemData, 0, len(merged))
	for _, item := range merged {
		items = append(items, ModerationQueueItemData{
			ItemType:         item.ItemType,
			ItemID:           item.ItemID,
			PlayID:           item.PlayID,
			Title:            item.Title,
			TheaterName:      item.TheaterName,
			City:             item.City,
			CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339Nano),
			CreatedByUserID:  item.CreatedByUserID,
			SubmissionStatus: item.SubmissionStatus,
			SuggestionStatus: item.SuggestionStatus,
		})
	}

	response := ModerationQueueData{Items: items}
	_ = hasNext
	return response, nil
}

func (s *Service) GetAdminPlayEditSuggestionByID(ctx context.Context, userID string, role string, suggestionID string) (PlayEditSuggestionData, error) {
	if !isValidAuthUserID(userID) {
		return PlayEditSuggestionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}
	if err := requireAdminRole(role); err != nil {
		return PlayEditSuggestionData{}, err
	}
	if !isValidUUID(suggestionID) {
		return PlayEditSuggestionData{}, sharederrors.Validation("invalid suggestionId", nil)
	}

	record, err := s.repo.GetPlayEditSuggestionByID(ctx, suggestionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayEditSuggestionData{}, sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to load play edit suggestion", nil)
	}
	return s.mapPlayEditSuggestionWithDetails(ctx, record)
}

func (s *Service) ApprovePlayEditSuggestion(ctx context.Context, userID string, role string, suggestionID string) (PlayEditSuggestionData, error) {
	if !isValidAuthUserID(userID) {
		return PlayEditSuggestionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}
	if err := requireAdminRole(role); err != nil {
		return PlayEditSuggestionData{}, err
	}
	if !isValidUUID(suggestionID) {
		return PlayEditSuggestionData{}, sharederrors.Validation("invalid suggestionId", nil)
	}

	current, err := s.repo.GetPlayEditSuggestionByID(ctx, suggestionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayEditSuggestionData{}, sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to approve play edit suggestion", nil)
	}
	if current.Status != "pending" {
		return PlayEditSuggestionData{}, sharederrors.Validation("only pending play edit suggestions can be approved", nil)
	}

	now := time.Now().UTC()
	record, err := s.repo.ModeratePlayEditSuggestion(ctx, suggestionID, ModeratePlayEditSuggestionParams{
		AdminUserID: userID,
		Status:      "approved",
		ModeratedAt: now,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayEditSuggestionData{}, sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to approve play edit suggestion", nil)
	}

	if err := s.repo.ApplyApprovedPlayEditSuggestion(ctx, suggestionID, now); err != nil {
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to apply approved play edit suggestion", nil)
	}

	return s.mapPlayEditSuggestionWithDetails(ctx, record)
}

func (s *Service) RejectPlayEditSuggestion(ctx context.Context, userID string, role string, suggestionID string, req RejectSubmissionRequest) (PlayEditSuggestionData, error) {
	if !isValidAuthUserID(userID) {
		return PlayEditSuggestionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}
	if err := requireAdminRole(role); err != nil {
		return PlayEditSuggestionData{}, err
	}
	if !isValidUUID(suggestionID) {
		return PlayEditSuggestionData{}, sharederrors.Validation("invalid suggestionId", nil)
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return PlayEditSuggestionData{}, sharederrors.Validation("reason must not be empty", nil)
	}

	current, err := s.repo.GetPlayEditSuggestionByID(ctx, suggestionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayEditSuggestionData{}, sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to reject play edit suggestion", nil)
	}
	if current.Status != "pending" {
		return PlayEditSuggestionData{}, sharederrors.Validation("only pending play edit suggestions can be rejected", nil)
	}

	record, err := s.repo.ModeratePlayEditSuggestion(ctx, suggestionID, ModeratePlayEditSuggestionParams{
		AdminUserID:    userID,
		Status:         "rejected",
		RejectedReason: &reason,
		ModeratedAt:    time.Now().UTC(),
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayEditSuggestionData{}, sharederrors.NotFound("play edit suggestion not found", nil)
		}
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to reject play edit suggestion", nil)
	}

	return s.mapPlayEditSuggestionWithDetails(ctx, record)
}

func (s *Service) CreateAdminSubmissionMediaUpload(ctx context.Context, userID string, role string, playID string, req CreateSubmissionMediaUploadRequest) (CreateSubmissionMediaUploadData, error) {
	if !isValidAuthUserID(userID) {
		return CreateSubmissionMediaUploadData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return CreateSubmissionMediaUploadData{}, err
	}

	if !isValidUUID(playID) {
		return CreateSubmissionMediaUploadData{}, sharederrors.Validation("invalid playId", nil)
	}

	if _, err := normalizePlayMediaKind(req.Kind); err != nil {
		return CreateSubmissionMediaUploadData{}, err
	}

	contentType, _, err := normalizeImageContentType(req.ContentType)
	if err != nil {
		return CreateSubmissionMediaUploadData{}, err
	}

	if req.ContentLength <= 0 {
		return CreateSubmissionMediaUploadData{}, sharederrors.Validation("contentLength must be greater than 0", nil)
	}

	if req.ContentLength > s.maxImageBytes {
		return CreateSubmissionMediaUploadData{}, sharederrors.Validation("contentLength exceeds max allowed size", nil)
	}

	if _, err := s.repo.GetSubmissionByID(ctx, playID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreateSubmissionMediaUploadData{}, sharederrors.NotFound("submission not found", nil)
		}

		return CreateSubmissionMediaUploadData{}, sharederrors.Internal("failed to create media upload", nil)
	}

	objectKey := fmt.Sprintf("plays/%s/%s.webp", playID, uuid.NewString())
	contentLength := req.ContentLength
	uploadURL, err := s.mediaStorage.PresignPutObject(ctx, storage.PresignPutObjectInput{
		ObjectKey:     objectKey,
		ContentType:   contentType,
		ContentLength: &contentLength,
		ExpiresIn:     s.mediaUploadTTL,
	})
	if err != nil {
		return CreateSubmissionMediaUploadData{}, sharederrors.Internal("failed to create media upload", nil)
	}

	if s.imageQueue != nil && s.optimizeEnabled {
		_ = s.imageQueue.Enqueue(OptimizeMediaJob{
			Storage:      s.mediaStorage,
			ObjectKey:    objectKey,
			Quality:      s.webpQuality,
			MaxWidth:     playMediaMaxWidth,
			MaxHeight:    playMediaMaxHeight,
			CacheControl: "public, max-age=31536000, immutable",
		})
	}

	return CreateSubmissionMediaUploadData{ObjectKey: objectKey, UploadURL: uploadURL}, nil
}

func (s *Service) AttachAdminSubmissionMedia(ctx context.Context, userID string, role string, playID string, req AttachSubmissionMediaRequest) (PlayMediaData, error) {
	if !isValidAuthUserID(userID) {
		return PlayMediaData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return PlayMediaData{}, err
	}

	if !isValidUUID(playID) {
		return PlayMediaData{}, sharederrors.Validation("invalid playId", nil)
	}

	kind, err := normalizePlayMediaKind(req.Kind)
	if err != nil {
		return PlayMediaData{}, err
	}

	objectKey := strings.TrimSpace(req.ObjectKey)
	if objectKey == "" {
		return PlayMediaData{}, sharederrors.Validation("objectKey must not be empty", nil)
	}

	if !strings.HasPrefix(objectKey, "plays/"+playID+"/") {
		return PlayMediaData{}, sharederrors.Validation("objectKey does not belong to this play", nil)
	}

	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
		if sortOrder < 0 {
			return PlayMediaData{}, sharederrors.Validation("sortOrder must be greater than or equal to 0", nil)
		}
	}

	altText := normalizeOptionalText(req.AltText)

	if _, err := s.repo.GetSubmissionByID(ctx, playID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayMediaData{}, sharederrors.NotFound("submission not found", nil)
		}

		return PlayMediaData{}, sharederrors.Internal("failed to attach media", nil)
	}

	headObject, err := s.mediaStorage.HeadObject(ctx, storage.HeadObjectInput{ObjectKey: objectKey})
	if err != nil {
		if storage.IsNotFoundError(err) {
			return PlayMediaData{}, sharederrors.Validation("uploaded object was not found", nil)
		}

		return PlayMediaData{}, sharederrors.Internal("failed to attach media", nil)
	}

	if _, _, err := normalizeImageContentType(headObject.ContentType); err != nil {
		return PlayMediaData{}, err
	}

	if headObject.ContentLength <= 0 || headObject.ContentLength > s.maxImageBytes {
		return PlayMediaData{}, sharederrors.Validation("uploaded object size is invalid", nil)
	}

	optimizedObjectKey, err := s.optimizeAndStorePlayMedia(ctx, objectKey)
	if err != nil {
		return PlayMediaData{}, err
	}

	record, err := s.repo.CreatePlayMedia(ctx, CreatePlayMediaParams{
		PlayID:    playID,
		Kind:      kind,
		ObjectKey: optimizedObjectKey,
		AltText:   altText,
		SortOrder: sortOrder,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		if isDuplicatedKeyError(err) {
			return PlayMediaData{}, sharederrors.New(http.StatusConflict, "PLAY_MEDIA_ALREADY_EXISTS", "play media already exists", nil)
		}

		return PlayMediaData{}, sharederrors.Internal("failed to attach media", nil)
	}

	presignedURL, presignErr := s.mediaStorage.PresignGetObject(ctx, storage.PresignGetObjectInput{
		ObjectKey: record.ObjectKey,
		ExpiresIn: s.mediaDownloadTTL,
	})
	if presignErr == nil && presignedURL != "" {
		return PlayMediaData{
			ID:        record.ID,
			Kind:      record.Kind,
			URL:       presignedURL,
			AltText:   record.AltText,
			SortOrder: record.SortOrder,
		}, nil
	}

	return mapPlayMediaRecord(playID, record, s.playMediaBaseURL), nil
}

func (s *Service) DeleteAdminSubmissionMedia(ctx context.Context, userID string, role string, playID string, mediaID string) error {
	if !isValidAuthUserID(userID) {
		return sharederrors.Unauthorized("invalid access token", nil)
	}

	if err := requireAdminRole(role); err != nil {
		return err
	}

	if !isValidUUID(playID) {
		return sharederrors.Validation("invalid playId", nil)
	}

	if !isValidUUID(mediaID) {
		return sharederrors.Validation("invalid mediaId", nil)
	}

	media, err := s.repo.GetPlayMediaByID(ctx, mediaID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sharederrors.NotFound("media not found", nil)
		}

		return sharederrors.Internal("failed to delete media", nil)
	}

	if media.PlayID != playID {
		return sharederrors.NotFound("media not found", nil)
	}

	if err := s.repo.DeletePlayMedia(ctx, mediaID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sharederrors.NotFound("media not found", nil)
		}

		return sharederrors.Internal("failed to delete media", nil)
	}

	return nil
}

func (s *Service) SetEngagement(ctx context.Context, userID string, playID string, req SetEngagementRequest) (EngagementStateData, error) {
	if !isValidAuthUserID(userID) {
		return EngagementStateData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(playID) {
		return EngagementStateData{}, sharederrors.Validation("invalid playId", nil)
	}

	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind != "wishlist" && kind != "attended" {
		return EngagementStateData{}, sharederrors.Validation("kind must be one of: wishlist, attended", nil)
	}

	isPublished, err := s.repo.IsPlayPublished(ctx, playID)
	if err != nil {
		return EngagementStateData{}, sharederrors.Internal("failed to set engagement", nil)
	}

	if !isPublished {
		return EngagementStateData{}, sharederrors.NotFound("play not found", nil)
	}

	if err := s.repo.SetEngagement(ctx, userID, playID, kind, time.Now().UTC()); err != nil {
		return EngagementStateData{}, sharederrors.Internal("failed to set engagement", nil)
	}

	return s.loadEngagementState(ctx, userID, playID)
}

func (s *Service) DeleteEngagement(ctx context.Context, userID string, playID string, kind string) (EngagementStateData, error) {
	if !isValidAuthUserID(userID) {
		return EngagementStateData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(playID) {
		return EngagementStateData{}, sharederrors.Validation("invalid playId", nil)
	}

	normalizedKind := strings.ToLower(strings.TrimSpace(kind))
	if normalizedKind != "wishlist" && normalizedKind != "attended" {
		return EngagementStateData{}, sharederrors.Validation("kind must be one of: wishlist, attended", nil)
	}

	isPublished, err := s.repo.IsPlayPublished(ctx, playID)
	if err != nil {
		return EngagementStateData{}, sharederrors.Internal("failed to delete engagement", nil)
	}

	if !isPublished {
		return EngagementStateData{}, sharederrors.NotFound("play not found", nil)
	}

	if err := s.repo.DeleteEngagement(ctx, userID, playID, normalizedKind); err != nil {
		return EngagementStateData{}, sharederrors.Internal("failed to delete engagement", nil)
	}

	return s.loadEngagementState(ctx, userID, playID)
}

func buildPlayListData(records []PlayListRecord, limit int, playMediaBaseURL string) (FeedData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]PlayCardData, 0, len(records))
	for _, record := range records {
		items = append(items, PlayCardData{
			ID:                 record.ID,
			Title:              record.Title,
			TheaterName:        record.TheaterName,
			City:               record.City,
			AvailabilityStatus: record.AvailabilityStatus,
			PublishedAt:        record.PublishedAt.UTC().Format(time.RFC3339Nano),
			PosterURL:          buildPosterURL(playMediaBaseURL, record.ID, record.PosterMediaID),
			AverageRating:      record.AverageRating,
			ReviewCount:        record.ReviewCount,
		})
	}

	response := FeedData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor, err := encodePlayListCursor(playListCursor{PublishedAt: last.PublishedAt.UTC(), PlayID: last.ID})
		if err != nil {
			return FeedData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}

		response.NextCursor = &nextCursor
	}

	return response, nil
}

func buildFeedData(records []PlayListRecord, limit int, section string, playMediaBaseURL string) (FeedData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]PlayCardData, 0, len(records))
	for _, record := range records {
		items = append(items, PlayCardData{
			ID:                 record.ID,
			Title:              record.Title,
			TheaterName:        record.TheaterName,
			City:               record.City,
			AvailabilityStatus: record.AvailabilityStatus,
			PublishedAt:        record.PublishedAt.UTC().Format(time.RFC3339Nano),
			PosterURL:          buildPosterURL(playMediaBaseURL, record.ID, record.PosterMediaID),
			AverageRating:      record.AverageRating,
			ReviewCount:        record.ReviewCount,
		})
	}

	response := FeedData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]

		if section == "trending" {
			nextCursor, err := encodeTrendingFeedCursor(trendingFeedCursor{Section: "trending", TrendScore: last.TrendScore, PublishedAt: last.PublishedAt.UTC(), PlayID: last.ID})
			if err != nil {
				return FeedData{}, sharederrors.Internal("failed to build pagination cursor", nil)
			}

			response.NextCursor = &nextCursor
			return response, nil
		}

		nextCursor, err := encodePlayListCursor(playListCursor{PublishedAt: last.PublishedAt.UTC(), PlayID: last.ID})
		if err != nil {
			return FeedData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}

		response.NextCursor = &nextCursor
	}

	return response, nil
}

func buildReviewListData(records []ReviewRecord, limit int) (ReviewListData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]ReviewData, 0, len(records))
	for _, record := range records {
		items = append(items, mapReviewRecord(record))
	}

	response := ReviewListData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor, err := encodeReviewListCursor(reviewListCursor{CreatedAt: last.CreatedAt.UTC(), ReviewID: last.ID})
		if err != nil {
			return ReviewListData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}

		response.NextCursor = &nextCursor
	}

	return response, nil
}

func mapPlayDetails(play PlayDetailsRecord, genres []PlayGenreRecord, cast []PlayCastRecord, media []PlayMediaRecord, playMediaBaseURL string) PlayDetailsData {
	genreItems := make([]PlayGenreData, 0, len(genres))
	for _, genre := range genres {
		genreItems = append(genreItems, PlayGenreData{ID: genre.ID, Name: genre.Name})
	}

	castItems := make([]PlayCastMemberData, 0, len(cast))
	for _, member := range cast {
		castItems = append(castItems, PlayCastMemberData{
			PersonName:   member.PersonName,
			RoleName:     member.RoleName,
			BillingOrder: member.BillingOrder,
		})
	}

	mediaItems := make([]PlayMediaData, 0, len(media))
	for _, item := range media {
		mediaItems = append(mediaItems, PlayMediaData{
			ID:        item.ID,
			Kind:      item.Kind,
			URL:       buildPlayMediaURL(playMediaBaseURL, play.ID, item.ID),
			AltText:   item.AltText,
			SortOrder: item.SortOrder,
		})
	}

	return PlayDetailsData{
		ID:                 play.ID,
		Title:              play.Title,
		Synopsis:           play.Synopsis,
		Director:           play.Director,
		DurationMinutes:    play.DurationMinutes,
		TheaterName:        play.TheaterName,
		City:               play.City,
		AvailabilityStatus: play.AvailabilityStatus,
		PublishedAt:        play.PublishedAt.UTC().Format(time.RFC3339Nano),
		Stats: PlayStatsData{
			AverageRating: play.AverageRating,
			ReviewCount:   play.ReviewCount,
		},
		Genres: genreItems,
		Cast:   castItems,
		Media:  mediaItems,
	}
}

func mapReviewRecord(record ReviewRecord) ReviewData {
	return ReviewData{
		ID:               record.ID,
		UserID:           record.UserID,
		DisplayName:      record.DisplayName,
		Rating:           record.Rating,
		Title:            record.Title,
		Body:             record.Body,
		ContainsSpoilers: record.ContainsSpoilers,
		CreatedAt:        record.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        record.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func mapReviewCommentRecord(record ReviewCommentRecord) ReviewCommentData {
	return ReviewCommentData{
		ID:          record.ID,
		ReviewID:    record.ReviewID,
		UserID:      record.UserID,
		DisplayName: record.DisplayName,
		Body:        record.Body,
		CreatedAt:   record.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   record.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func mapReviewCommentStatusRecord(record ReviewCommentStatusRecord) ReviewCommentStatusData {
	return ReviewCommentStatusData{
		ID:        record.ID,
		ReviewID:  record.ReviewID,
		Status:    record.Status,
		UpdatedAt: record.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (s *Service) mapSubmissionWithGenres(ctx context.Context, record SubmissionRecord) (SubmissionData, error) {
	genres, err := s.repo.ListPlayGenres(ctx, record.ID)
	if err != nil {
		return SubmissionData{}, sharederrors.Internal("failed to load submission", nil)
	}

	return mapSubmissionRecord(record, genres), nil
}

func (s *Service) mapSubmissionWithGenresAndMedia(ctx context.Context, record SubmissionRecord) (SubmissionData, error) {
	genres, err := s.repo.ListPlayGenres(ctx, record.ID)
	if err != nil {
		return SubmissionData{}, sharederrors.Internal("failed to load submission", nil)
	}

	mediaRecords, err := s.repo.ListPlayMedia(ctx, record.ID)
	if err != nil {
		return SubmissionData{}, sharederrors.Internal("failed to load submission media", nil)
	}

	data := mapSubmissionRecord(record, genres)

	mediaItems := make([]PlayMediaData, 0, len(mediaRecords))
	for _, item := range mediaRecords {
		mediaURL := buildPlayMediaURL(s.playMediaBaseURL, record.ID, item.ID)

		presignedURL, presignErr := s.mediaStorage.PresignGetObject(ctx, storage.PresignGetObjectInput{
			ObjectKey: item.ObjectKey,
			ExpiresIn: s.mediaDownloadTTL,
		})
		if presignErr == nil && presignedURL != "" {
			mediaURL = presignedURL
		}

		mediaItems = append(mediaItems, PlayMediaData{
			ID:        item.ID,
			Kind:      item.Kind,
			URL:       mediaURL,
			AltText:   item.AltText,
			SortOrder: item.SortOrder,
		})
	}
	data.Media = mediaItems

	return data, nil
}

func mapSubmissionRecord(record SubmissionRecord, genres []PlayGenreRecord) SubmissionData {
	return SubmissionData{
		ID:                 record.ID,
		Title:              record.Title,
		Synopsis:           record.Synopsis,
		Director:           record.Director,
		DurationMinutes:    record.DurationMinutes,
		TheaterName:        record.TheaterName,
		City:               record.City,
		AvailabilityStatus: record.AvailabilityStatus,
		Genres:             mapPlayGenreRecords(genres),
		CurationStatus:     record.CurationStatus,
		CreatedByUserID:    record.CreatedByUserID,
		ModeratedByUserID:  record.ModeratedByUserID,
		ModeratedAt:        formatTimePointer(record.ModeratedAt),
		PublishedAt:        formatTimePointer(record.PublishedAt),
		RejectedReason:     record.RejectedReason,
		CreatedAt:          record.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:          record.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (s *Service) getPlayEditSuggestionData(ctx context.Context, suggestionID string) (PlayEditSuggestionData, error) {
	record, err := s.repo.GetPlayEditSuggestionByID(ctx, suggestionID)
	if err != nil {
		return PlayEditSuggestionData{}, err
	}
	return s.mapPlayEditSuggestionWithDetails(ctx, record)
}

func (s *Service) mapPlayEditSuggestionWithDetails(ctx context.Context, record PlayEditSuggestionRecord) (PlayEditSuggestionData, error) {
	genres, err := s.repo.ListPlayEditSuggestionGenres(ctx, record.ID)
	if err != nil {
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to load play edit suggestion", nil)
	}
	mediaRecords, err := s.repo.ListPlayEditSuggestionMedia(ctx, record.ID)
	if err != nil {
		return PlayEditSuggestionData{}, sharederrors.Internal("failed to load play edit suggestion media", nil)
	}

	mediaItems := make([]PlayMediaData, 0, len(mediaRecords))
	for _, item := range mediaRecords {
		mediaURL := item.ObjectKey
		presignedURL, presignErr := s.mediaStorage.PresignGetObject(ctx, storage.PresignGetObjectInput{
			ObjectKey: item.ObjectKey,
			ExpiresIn: s.mediaDownloadTTL,
		})
		if presignErr == nil && presignedURL != "" {
			mediaURL = presignedURL
		}

		mediaItems = append(mediaItems, PlayMediaData{
			ID:        item.ID,
			Kind:      item.Kind,
			URL:       mediaURL,
			AltText:   item.AltText,
			SortOrder: item.SortOrder,
		})
	}

	return PlayEditSuggestionData{
		ID:                 record.ID,
		PlayID:             record.PlayID,
		Title:              record.Title,
		Synopsis:           record.Synopsis,
		Director:           record.Director,
		DurationMinutes:    record.DurationMinutes,
		TheaterName:        record.TheaterName,
		City:               record.City,
		AvailabilityStatus: record.AvailabilityStatus,
		Genres:             mapPlayGenreRecords(genres),
		Media:              mediaItems,
		Status:             record.Status,
		CreatedByUserID:    record.CreatedByUserID,
		ModeratedByUserID:  record.ModeratedByUserID,
		ModeratedAt:        formatTimePointer(record.ModeratedAt),
		RejectedReason:     record.RejectedReason,
		CreatedAt:          record.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:          record.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}, nil
}

func mapGenreRecord(record GenreRecord) GenreData {
	return GenreData{
		ID:   record.ID,
		Name: record.Name,
	}
}

func mapPlayGenreRecords(records []PlayGenreRecord) []PlayGenreData {
	genreItems := make([]PlayGenreData, 0, len(records))
	for _, genre := range records {
		genreItems = append(genreItems, PlayGenreData{ID: genre.ID, Name: genre.Name})
	}

	return genreItems
}

func buildGenreListData(records []GenreRecord, limit int) (GenreListData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]GenreData, 0, len(records))
	for _, record := range records {
		items = append(items, mapGenreRecord(record))
	}

	response := GenreListData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor, err := encodeGenreListCursor(genreListCursor{Name: last.Name, GenreID: last.ID})
		if err != nil {
			return GenreListData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}

		response.NextCursor = &nextCursor
	}

	return response, nil
}

func buildSubmissionListData(records []SubmissionRecord, limit int) (SubmissionListData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]SubmissionData, 0, len(records))
	for _, record := range records {
		items = append(items, mapSubmissionRecord(record, nil))
	}

	response := SubmissionListData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor, err := encodeSubmissionListCursor(submissionListCursor{CreatedAt: last.CreatedAt.UTC(), PlayID: last.ID})
		if err != nil {
			return SubmissionListData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}

		response.NextCursor = &nextCursor
	}

	return response, nil
}

func (s *Service) buildPlayEditSuggestionListData(ctx context.Context, records []PlayEditSuggestionRecord, limit int) (PlayEditSuggestionListData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]PlayEditSuggestionData, 0, len(records))
	for _, record := range records {
		data, err := s.mapPlayEditSuggestionWithDetails(ctx, record)
		if err != nil {
			return PlayEditSuggestionListData{}, err
		}
		items = append(items, data)
	}

	response := PlayEditSuggestionListData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor, err := encodePlayEditSuggestionListCursor(playEditSuggestionListCursor{
			CreatedAt:    last.CreatedAt.UTC(),
			SuggestionID: last.ID,
		})
		if err != nil {
			return PlayEditSuggestionListData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}
		response.NextCursor = &nextCursor
	}

	return response, nil
}

func buildMyEngagementPlayListData(records []EngagementPlayRecord, limit int, playMediaBaseURL string) (MyEngagementPlayListData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]MyEngagementPlayData, 0, len(records))
	for _, record := range records {
		items = append(items, MyEngagementPlayData{
			ID:                 record.ID,
			Title:              record.Title,
			TheaterName:        record.TheaterName,
			City:               record.City,
			AvailabilityStatus: record.AvailabilityStatus,
			PublishedAt:        record.PublishedAt.UTC().Format(time.RFC3339Nano),
			PosterURL:          buildPosterURL(playMediaBaseURL, record.ID, record.PosterMediaID),
			AverageRating:      record.AverageRating,
			ReviewCount:        record.ReviewCount,
			EngagedAt:          record.EngagedAt.UTC().Format(time.RFC3339Nano),
		})
	}

	response := MyEngagementPlayListData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor, err := encodeEngagementPlayListCursor(engagementPlayListCursor{EngagedAt: last.EngagedAt.UTC(), PlayID: last.ID})
		if err != nil {
			return MyEngagementPlayListData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}

		response.NextCursor = &nextCursor
	}

	return response, nil
}

func buildUserReviewListData(records []UserReviewRecord, limit int, playMediaBaseURL string) (UserReviewListData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]UserReviewData, 0, len(records))
	for _, record := range records {
		items = append(items, UserReviewData{
			ID: record.ID,
			Play: UserReviewPlayData{
				ID:                 record.PlayID,
				Title:              record.PlayTitle,
				TheaterName:        record.TheaterName,
				City:               record.City,
				AvailabilityStatus: record.AvailabilityStatus,
				PublishedAt:        record.PublishedAt.UTC().Format(time.RFC3339Nano),
				PosterURL:          buildPosterURL(playMediaBaseURL, record.PlayID, record.PosterMediaID),
			},
			Rating:           record.Rating,
			Title:            record.Title,
			Body:             record.Body,
			ContainsSpoilers: record.ContainsSpoilers,
			CreatedAt:        record.CreatedAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt:        record.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}

	response := UserReviewListData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor, err := encodeReviewListCursor(reviewListCursor{CreatedAt: last.CreatedAt.UTC(), ReviewID: last.ID})
		if err != nil {
			return UserReviewListData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}

		response.NextCursor = &nextCursor
	}

	return response, nil
}

func formatTimePointer(raw *time.Time) *string {
	if raw == nil {
		return nil
	}

	formatted := raw.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func mapPlayMediaRecord(playID string, record PlayMediaRecord, playMediaBaseURL string) PlayMediaData {
	return PlayMediaData{
		ID:        record.ID,
		Kind:      record.Kind,
		URL:       buildPlayMediaURL(playMediaBaseURL, playID, record.ID),
		AltText:   record.AltText,
		SortOrder: record.SortOrder,
	}
}

func buildPosterURL(baseURL string, playID string, mediaID *string) *string {
	if mediaID == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*mediaID)
	if trimmed == "" {
		return nil
	}

	url := buildPlayMediaURL(baseURL, playID, trimmed)
	return &url
}

func buildPlayMediaURL(baseURL string, playID string, mediaID string) string {
	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = defaultPlayMediaPrefix
	}

	return base + "/" + strings.TrimSpace(playID) + "/" + strings.TrimSpace(mediaID)
}

func normalizeOptionalText(raw *string) *string {
	if raw == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func (s *Service) optimizeAndStorePlayMedia(ctx context.Context, objectKey string) (string, error) {
	content, err := s.mediaStorage.GetObject(ctx, storage.GetObjectInput{ObjectKey: objectKey})
	if err != nil {
		if storage.IsNotFoundError(err) {
			return "", sharederrors.Validation("uploaded object was not found", nil)
		}

		return "", sharederrors.Internal("failed to attach media", nil)
	}

	result, err := imageproc.OptimizeImage(content, imageproc.OptimizeOptions{
		Quality:           s.webpQuality,
		MaxWidth:          playMediaMaxWidth,
		MaxHeight:         playMediaMaxHeight,
		TargetContentType: "image/webp",
	})
	if err != nil {
		return "", sharederrors.Validation("uploaded object is not a valid image", nil)
	}

	optimizedObjectKey := normalizeWebPObjectKey(objectKey)
	if err := s.mediaStorage.PutObject(ctx, storage.PutObjectInput{
		ObjectKey:    optimizedObjectKey,
		Content:      result.Content,
		ContentType:  result.ContentType,
		CacheControl: "public, max-age=31536000, immutable",
	}); err != nil {
		return "", sharederrors.Internal("failed to attach media", nil)
	}

	return optimizedObjectKey, nil
}

func normalizeWebPObjectKey(objectKey string) string {
	trimmed := strings.TrimSpace(objectKey)
	if trimmed == "" {
		return ""
	}

	lastSlash := strings.LastIndex(trimmed, "/")
	lastDot := strings.LastIndex(trimmed, ".")
	if lastDot > lastSlash {
		return trimmed[:lastDot] + ".webp"
	}

	return trimmed + ".webp"
}

func requiredSubmissionText(raw string, field string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", sharederrors.Validation(field+" must not be empty", nil)
	}

	return trimmed, nil
}

func normalizeSubmissionGenreIDs(rawIDs []string) ([]string, error) {
	if len(rawIDs) == 0 {
		return nil, sharederrors.Validation("genreIds must include at least one genre", nil)
	}

	normalized := make([]string, 0, len(rawIDs))
	seen := make(map[string]struct{}, len(rawIDs))

	for _, rawID := range rawIDs {
		genreID := strings.TrimSpace(rawID)
		if genreID == "" {
			return nil, sharederrors.Validation("genreIds must contain valid UUIDs", nil)
		}

		if !isValidUUID(genreID) {
			return nil, sharederrors.Validation("genreIds must contain valid UUIDs", nil)
		}

		if _, exists := seen[genreID]; exists {
			continue
		}

		seen[genreID] = struct{}{}
		normalized = append(normalized, genreID)
	}

	if len(normalized) == 0 {
		return nil, sharederrors.Validation("genreIds must include at least one genre", nil)
	}

	return normalized, nil
}

func normalizePlayMediaKind(raw string) (string, error) {
	kind := strings.ToLower(strings.TrimSpace(raw))
	if kind != "poster" && kind != "photo" {
		return "", sharederrors.Validation("kind must be one of: poster, photo", nil)
	}

	return kind, nil
}

func normalizeImageContentType(raw string) (string, string, error) {
	contentType := strings.ToLower(strings.TrimSpace(raw))
	if contentType == "" {
		return "", "", sharederrors.Validation("contentType must not be empty", nil)
	}

	if strings.Contains(contentType, ";") {
		parts := strings.SplitN(contentType, ";", 2)
		contentType = strings.TrimSpace(parts[0])
	}

	switch contentType {
	case "image/jpeg":
		return contentType, "jpg", nil
	case "image/png":
		return contentType, "png", nil
	case "image/webp":
		return contentType, "webp", nil
	default:
		return "", "", sharederrors.Validation("contentType must be one of: image/jpeg, image/png, image/webp", nil)
	}
}

func normalizeAvailabilityStatus(raw *string) (string, error) {
	if raw == nil {
		return "in_theaters", nil
	}

	trimmed := strings.ToLower(strings.TrimSpace(*raw))
	if trimmed != "in_theaters" && trimmed != "archive" {
		return "", sharederrors.Validation("availabilityStatus must be one of: in_theaters, archive", nil)
	}

	return trimmed, nil
}

func normalizeSubmissionStatus(raw string, defaultPending bool) (*string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		if !defaultPending {
			return nil, nil
		}

		status := "pending"
		return &status, nil
	}

	if trimmed != "pending" && trimmed != "published" && trimmed != "rejected" {
		return nil, sharederrors.Validation("status must be one of: pending, published, rejected", nil)
	}

	return &trimmed, nil
}

func normalizePlayEditSuggestionStatus(raw string, defaultPending bool) (*string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		if !defaultPending {
			return nil, nil
		}
		status := "pending"
		return &status, nil
	}
	if trimmed != "pending" && trimmed != "approved" && trimmed != "rejected" {
		return nil, sharederrors.Validation("status must be one of: pending, approved, rejected", nil)
	}
	return &trimmed, nil
}

func hasPlayEditSuggestionPatch(req UpdatePlayEditSuggestionRequest) bool {
	return req.Title != nil ||
		req.Synopsis != nil ||
		req.Director != nil ||
		req.DurationMinutes != nil ||
		req.TheaterName != nil ||
		req.City != nil ||
		req.AvailabilityStatus != nil ||
		req.GenreIDs != nil
}

func validatePlayEditSuggestionPatch(req UpdatePlayEditSuggestionRequest) (UpdatePlayEditSuggestionParams, error) {
	if !hasPlayEditSuggestionPatch(req) {
		return UpdatePlayEditSuggestionParams{}, sharederrors.Validation("at least one editable field must be provided", nil)
	}

	patch := UpdatePlayEditSuggestionParams{}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return UpdatePlayEditSuggestionParams{}, sharederrors.Validation("title must not be empty", nil)
		}
		patch.Title = &title
	}

	if req.Synopsis != nil {
		synopsis := strings.TrimSpace(*req.Synopsis)
		if synopsis == "" {
			return UpdatePlayEditSuggestionParams{}, sharederrors.Validation("synopsis must not be empty", nil)
		}
		patch.Synopsis = &synopsis
	}

	if req.Director != nil {
		director := strings.TrimSpace(*req.Director)
		if director == "" {
			return UpdatePlayEditSuggestionParams{}, sharederrors.Validation("director must not be empty", nil)
		}
		patch.Director = &director
	}

	if req.DurationMinutes != nil {
		if *req.DurationMinutes <= 0 {
			return UpdatePlayEditSuggestionParams{}, sharederrors.Validation("durationMinutes must be greater than 0", nil)
		}
		patch.DurationMinutes = req.DurationMinutes
	}

	if req.TheaterName != nil {
		theaterName := strings.TrimSpace(*req.TheaterName)
		if theaterName == "" {
			return UpdatePlayEditSuggestionParams{}, sharederrors.Validation("theaterName must not be empty", nil)
		}
		patch.TheaterName = &theaterName
	}

	if req.City != nil {
		patch.CityProvided = true
		city := strings.TrimSpace(*req.City)
		if city == "" {
			patch.City = nil
		} else {
			patch.City = &city
		}
	}

	if req.AvailabilityStatus != nil {
		normalized, err := normalizeAvailabilityStatus(req.AvailabilityStatus)
		if err != nil {
			return UpdatePlayEditSuggestionParams{}, err
		}
		patch.AvailabilityStatus = &normalized
	}

	if req.GenreIDs != nil {
		normalized, err := normalizeSubmissionGenreIDs(req.GenreIDs)
		if err != nil {
			return UpdatePlayEditSuggestionParams{}, err
		}
		patch.GenreIDs = normalized
		patch.GenreIDsProvided = true
	}

	return patch, nil
}

func hasSubmissionPatch(req UpdateSubmissionRequest) bool {
	return req.Title != nil || req.Synopsis != nil || req.Director != nil || req.DurationMinutes != nil || req.TheaterName != nil || req.City != nil || req.AvailabilityStatus != nil
}

func validateSubmissionPatch(req UpdateSubmissionRequest) (UpdateSubmissionParams, error) {
	patch := UpdateSubmissionParams{}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return UpdateSubmissionParams{}, sharederrors.Validation("title must not be empty", nil)
		}
		patch.Title = &title
	}

	if req.Synopsis != nil {
		synopsis := strings.TrimSpace(*req.Synopsis)
		if synopsis == "" {
			return UpdateSubmissionParams{}, sharederrors.Validation("synopsis must not be empty", nil)
		}
		patch.Synopsis = &synopsis
	}

	if req.Director != nil {
		director := strings.TrimSpace(*req.Director)
		if director == "" {
			return UpdateSubmissionParams{}, sharederrors.Validation("director must not be empty", nil)
		}
		patch.Director = &director
	}

	if req.DurationMinutes != nil {
		if *req.DurationMinutes <= 0 {
			return UpdateSubmissionParams{}, sharederrors.Validation("durationMinutes must be greater than 0", nil)
		}
		patch.DurationMinutes = req.DurationMinutes
	}

	if req.TheaterName != nil {
		theaterName := strings.TrimSpace(*req.TheaterName)
		if theaterName == "" {
			return UpdateSubmissionParams{}, sharederrors.Validation("theaterName must not be empty", nil)
		}
		patch.TheaterName = &theaterName
	}

	if req.City != nil {
		patch.CityProvided = true
		city := strings.TrimSpace(*req.City)
		if city == "" {
			patch.City = nil
		} else {
			patch.City = &city
		}
	}

	if req.AvailabilityStatus != nil {
		availabilityStatus, err := normalizeAvailabilityStatus(req.AvailabilityStatus)
		if err != nil {
			return UpdateSubmissionParams{}, err
		}
		patch.AvailabilityStatus = &availabilityStatus
	}

	return patch, nil
}

func requireAdminRole(role string) error {
	if strings.ToLower(strings.TrimSpace(role)) != "admin" {
		return sharederrors.Forbidden("admin role is required", nil)
	}

	return nil
}

func invalidTransitionError(from string, to string) *sharederrors.AppError {
	return sharederrors.New(http.StatusConflict, "INVALID_CURATION_TRANSITION", "invalid curation status transition", map[string]string{"from": from, "to": to})
}

func (s *Service) loadEngagementState(ctx context.Context, userID string, playID string) (EngagementStateData, error) {
	state, err := s.repo.GetEngagementState(ctx, userID, playID)
	if err != nil {
		return EngagementStateData{}, sharederrors.Internal("failed to load engagement state", nil)
	}

	return EngagementStateData{PlayID: playID, Wishlist: state.Wishlist, Attended: state.Attended}, nil
}

func (s *Service) listMyEngagementPlays(ctx context.Context, userID string, query ListMyEngagementsQuery, kind string, loadErrorMessage string) (MyEngagementPlayListData, error) {
	if !isValidAuthUserID(userID) {
		return MyEngagementPlayListData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	return s.listEngagementPlays(ctx, userID, query, kind, loadErrorMessage)
}

func (s *Service) listEngagementPlays(ctx context.Context, userID string, query ListMyEngagementsQuery, kind string, loadErrorMessage string) (MyEngagementPlayListData, error) {

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return MyEngagementPlayListData{}, err
	}

	cursor, err := decodeEngagementPlayListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return MyEngagementPlayListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListUserEngagementPlays(ctx, userID, ListUserEngagementPlaysParams{Kind: kind, After: cursor, Limit: limit + 1})
	if err != nil {
		return MyEngagementPlayListData{}, sharederrors.Internal(loadErrorMessage, nil)
	}

	return buildMyEngagementPlayListData(records, limit, s.playMediaBaseURL)
}

func (s *Service) listUserReviews(ctx context.Context, userID string, query ListUserReviewsQuery) (UserReviewListData, error) {
	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return UserReviewListData{}, err
	}

	cursor, err := decodeReviewListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return UserReviewListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListUserPublishedReviews(ctx, userID, ListUserReviewsParams{After: cursor, Limit: limit + 1})
	if err != nil {
		return UserReviewListData{}, sharederrors.Internal("failed to load reviews", nil)
	}

	return buildUserReviewListData(records, limit, s.playMediaBaseURL)
}

func (s *Service) ensureUserExists(ctx context.Context, userID string) error {
	exists, err := s.repo.UserExists(ctx, userID)
	if err != nil {
		return sharederrors.Internal("failed to load user", nil)
	}

	if !exists {
		return sharederrors.NotFound("user not found", nil)
	}

	return nil
}

func normalizeListLimit(rawLimit int) (int, error) {
	if rawLimit == 0 {
		return defaultListLimit, nil
	}

	if rawLimit < 1 || rawLimit > maxListLimit {
		return 0, sharederrors.Validation("limit must be between 1 and 50", nil)
	}

	return rawLimit, nil
}

func isValidUUID(raw string) bool {
	_, err := uuid.Parse(strings.TrimSpace(raw))
	return err == nil
}

func isValidAuthUserID(raw string) bool {
	return isValidUUID(strings.TrimSpace(raw))
}

func hasReviewPatch(req UpdateReviewRequest) bool {
	return req.Rating != nil || req.Title != nil || req.Body != nil || req.ContainsSpoilers != nil
}

func sameUUID(left string, right string) bool {
	leftUUID, leftErr := uuid.Parse(strings.TrimSpace(left))
	rightUUID, rightErr := uuid.Parse(strings.TrimSpace(right))
	if leftErr != nil || rightErr != nil {
		return false
	}

	return leftUUID == rightUUID
}

func isDuplicatedKeyError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}

func isForeignKeyViolationError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}

	return false
}
