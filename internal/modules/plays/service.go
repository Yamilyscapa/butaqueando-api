package plays

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	defaultListLimit       = 20
	maxListLimit           = 50
	defaultUploadURLTTL    = 15 * time.Minute
	defaultMaxImageBytes   = 10 * 1024 * 1024
	defaultPlayMediaPrefix = "/v1/media/plays"
)

type repositoryPort interface {
	ListFeed(ctx context.Context, params FeedListParams) ([]PlayListRecord, error)
	SearchPublishedPlays(ctx context.Context, params SearchListParams) ([]PlayListRecord, error)
	GetPublishedPlayByID(ctx context.Context, playID string) (PlayDetailsRecord, error)
	ListPlayGenres(ctx context.Context, playID string) ([]PlayGenreRecord, error)
	ListPlayCast(ctx context.Context, playID string) ([]PlayCastRecord, error)
	ListPlayMedia(ctx context.Context, playID string) ([]PlayMediaRecord, error)
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
	GetSubmissionByID(ctx context.Context, playID string) (SubmissionRecord, error)
	UpdateSubmission(ctx context.Context, playID string, params UpdateSubmissionParams) (SubmissionRecord, error)
	ListAdminSubmissions(ctx context.Context, params ListSubmissionsParams) ([]SubmissionRecord, error)
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
	maxImageBytes    int64
	playMediaBaseURL string
}

type ServiceOption func(*Service)

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

func NewService(repo repositoryPort, options ...ServiceOption) *Service {
	service := &Service{
		repo:             repo,
		mediaStorage:     storage.NoopClient{},
		mediaUploadTTL:   defaultUploadURLTTL,
		maxImageBytes:    defaultMaxImageBytes,
		playMediaBaseURL: defaultPlayMediaPrefix,
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

	if service.maxImageBytes <= 0 {
		service.maxImageBytes = defaultMaxImageBytes
	}

	if strings.TrimSpace(service.playMediaBaseURL) == "" {
		service.playMediaBaseURL = defaultPlayMediaPrefix
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
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return SubmissionData{}, sharederrors.Internal("failed to create submission", nil)
	}

	return mapSubmissionRecord(record), nil
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

	contentType, extension, err := normalizeImageContentType(req.ContentType)
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

	objectKey := fmt.Sprintf("plays/%s/%s.%s", playID, uuid.NewString(), extension)
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

	record, err := s.repo.CreatePlayMedia(ctx, CreatePlayMediaParams{
		PlayID:    playID,
		Kind:      kind,
		ObjectKey: objectKey,
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

	return mapSubmissionRecord(record), nil
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

	return mapSubmissionRecord(record), nil
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

	return mapSubmissionRecord(record), nil
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

func mapSubmissionRecord(record SubmissionRecord) SubmissionData {
	return SubmissionData{
		ID:                 record.ID,
		Title:              record.Title,
		Synopsis:           record.Synopsis,
		Director:           record.Director,
		DurationMinutes:    record.DurationMinutes,
		TheaterName:        record.TheaterName,
		City:               record.City,
		AvailabilityStatus: record.AvailabilityStatus,
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

func buildSubmissionListData(records []SubmissionRecord, limit int) (SubmissionListData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]SubmissionData, 0, len(records))
	for _, record := range records {
		items = append(items, mapSubmissionRecord(record))
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

func requiredSubmissionText(raw string, field string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", sharederrors.Validation(field+" must not be empty", nil)
	}

	return trimmed, nil
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
