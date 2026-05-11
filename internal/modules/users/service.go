package users

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/imageproc"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/butaqueando/api/internal/shared/worker"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	maxBioLength           = 140
	minUsernameLength      = 3
	maxUsernameLength      = 20
	defaultUploadURLTTL    = 15 * time.Minute
	defaultMaxImageBytes   = 10 * 1024 * 1024
	defaultUserMediaPrefix = "/v1/media/users"
	avatarMaxWidth         = 200
	avatarMaxHeight        = 200
	defaultUserSearchLimit = 20
	maxUserSearchLimit     = 50
)

var (
	usernameFormatRegex = regexp.MustCompile(`^[a-z0-9_]+$`)
	reservedUsernames   = map[string]struct{}{
		"admin":   {},
		"me":      {},
		"api":     {},
		"support": {},
		"null":    {},
		"system":  {},
		"root":    {},
	}
)

type repositoryPort interface {
	GetPublicProfile(ctx context.Context, userID string) (PublicProfileRecord, error)
	GetMeProfile(ctx context.Context, userID string) (MeProfileRecord, error)
	UpdateMeProfile(ctx context.Context, userID string, patch UpdateMeProfilePatch) (MeProfileRecord, error)
	IsUsernameAvailable(ctx context.Context, username string, excludeUserID string) (bool, error)
	SearchUsers(ctx context.Context, query string, limit int) ([]PublicProfileRecord, error)
	CreateAccountDeletionRequest(ctx context.Context, userID string) (AccountDeletionRequestRecord, error)
}

var ErrUsernameTaken = errors.New("username already taken")

type Service struct {
	repo             repositoryPort
	mediaStorage     storage.Client
	mediaUploadTTL   time.Duration
	maxImageBytes    int64
	userMediaBaseURL string
	imageQueue       ImageQueue
	optimizeEnabled  bool
	webpQuality      int
	variantWidths    []int
	blurhashEnabled  bool
}

type ServiceOption func(*Service)

type ImageQueue interface {
	Enqueue(job worker.Job) bool
}

type ImageJob interface {
	Run(ctx context.Context) error
	Name() string
}

type OptimizeAvatarJob struct {
	Storage      storage.Client
	ObjectKey    string
	Quality      int
	MaxWidth     int
	MaxHeight    int
	CacheControl string
}

func (j OptimizeAvatarJob) Name() string {
	return "optimize-user-avatar"
}

func (j OptimizeAvatarJob) Run(ctx context.Context) error {
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

func WithMaxImageBytes(maxBytes int64) ServiceOption {
	return func(s *Service) {
		s.maxImageBytes = maxBytes
	}
}

func WithUserMediaBaseURL(baseURL string) ServiceOption {
	return func(s *Service) {
		s.userMediaBaseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
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

func WithImageVariantWidths(widths []int) ServiceOption {
	return func(s *Service) {
		copied := make([]int, 0, len(widths))
		for _, w := range widths {
			if w > 0 {
				copied = append(copied, w)
			}
		}
		s.variantWidths = copied
	}
}

func WithBlurhashEnabled(enabled bool) ServiceOption {
	return func(s *Service) {
		s.blurhashEnabled = enabled
	}
}

func NewService(repo repositoryPort, options ...ServiceOption) *Service {
	service := &Service{
		repo:             repo,
		mediaStorage:     storage.NoopClient{},
		mediaUploadTTL:   defaultUploadURLTTL,
		maxImageBytes:    defaultMaxImageBytes,
		userMediaBaseURL: defaultUserMediaPrefix,
		optimizeEnabled:  true,
		webpQuality:      75,
		variantWidths:    []int{96, 240, 512},
		blurhashEnabled:  true,
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

	if strings.TrimSpace(service.userMediaBaseURL) == "" {
		service.userMediaBaseURL = defaultUserMediaPrefix
	}

	if service.webpQuality <= 0 || service.webpQuality > 100 {
		service.webpQuality = 75
	}

	if len(service.variantWidths) == 0 {
		service.variantWidths = []int{96, 240, 512}
	}

	return service
}

func (s *Service) GetPublicProfile(ctx context.Context, userID string) (PublicProfileData, error) {
	if !isValidUUID(userID) {
		return PublicProfileData{}, sharederrors.Validation("invalid userId", nil)
	}

	record, err := s.repo.GetPublicProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PublicProfileData{}, sharederrors.NotFound("user profile not found", nil)
		}

		return PublicProfileData{}, sharederrors.Internal("failed to load profile", nil)
	}

	return mapPublicProfileRecord(record, s.userMediaBaseURL), nil
}

func (s *Service) GetMeProfile(ctx context.Context, userID string) (MeProfileData, error) {
	if strings.TrimSpace(userID) == "" {
		return MeProfileData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(userID) {
		return MeProfileData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	record, err := s.repo.GetMeProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MeProfileData{}, sharederrors.NotFound("user profile not found", nil)
		}

		return MeProfileData{}, sharederrors.Internal("failed to load profile", nil)
	}

	return mapMeProfileRecord(record, s.userMediaBaseURL), nil
}

func (s *Service) UpdateMeProfile(ctx context.Context, userID string, req UpdateMeProfileRequest) (MeProfileData, error) {
	if strings.TrimSpace(userID) == "" {
		return MeProfileData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(userID) {
		return MeProfileData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	patch, err := validateAndBuildPatch(req)
	if err != nil {
		return MeProfileData{}, err
	}

	if patch.UsernameSet && patch.Username != nil {
		available, err := s.repo.IsUsernameAvailable(ctx, *patch.Username, userID)
		if err != nil {
			return MeProfileData{}, sharederrors.Internal("failed to update profile", nil)
		}
		if !available {
			return MeProfileData{}, sharederrors.New(http.StatusConflict, "USERNAME_TAKEN", "username already taken", nil)
		}
	}

	if patch.AvatarObjectKeySet {
		if patch.AvatarObjectKey != nil {
			if !strings.HasPrefix(*patch.AvatarObjectKey, "users/"+userID+"/avatar/") {
				return MeProfileData{}, sharederrors.Validation("avatarObjectKey does not belong to this user", nil)
			}

			headObject, headErr := s.mediaStorage.HeadObject(ctx, storage.HeadObjectInput{ObjectKey: *patch.AvatarObjectKey})
			if headErr != nil {
				if storage.IsNotFoundError(headErr) {
					return MeProfileData{}, sharederrors.Validation("uploaded object was not found", nil)
				}

				return MeProfileData{}, sharederrors.Internal("failed to update profile", nil)
			}

			if err := validateImageMetadata(headObject.ContentType, headObject.ContentLength, s.maxImageBytes); err != nil {
				return MeProfileData{}, err
			}

			optimized, optimizeErr := s.optimizeAndStoreAvatar(ctx, *patch.AvatarObjectKey)
			if optimizeErr != nil {
				return MeProfileData{}, optimizeErr
			}

			patch.AvatarObjectKey = &optimized.PrimaryObjectKey
			patch.AvatarVariants = optimized.Variants
			patch.AvatarBlurhash = optimized.Blurhash
		}
	}

	record, err := s.repo.UpdateMeProfile(ctx, userID, patch)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MeProfileData{}, sharederrors.NotFound("user profile not found", nil)
		}

		if errors.Is(err, ErrUsernameTaken) {
			return MeProfileData{}, sharederrors.New(http.StatusConflict, "USERNAME_TAKEN", "username already taken", nil)
		}

		return MeProfileData{}, sharederrors.Internal("failed to update profile", nil)
	}

	return mapMeProfileRecord(record, s.userMediaBaseURL), nil
}

func (s *Service) SearchUsers(ctx context.Context, query string, limit int) ([]PublicProfileData, error) {
	trimmed := strings.ToLower(strings.TrimSpace(query))
	if trimmed == "" {
		return []PublicProfileData{}, nil
	}

	if limit <= 0 {
		limit = defaultUserSearchLimit
	}
	if limit > maxUserSearchLimit {
		limit = maxUserSearchLimit
	}

	records, err := s.repo.SearchUsers(ctx, trimmed, limit)
	if err != nil {
		return nil, sharederrors.Internal("failed to search users", nil)
	}

	results := make([]PublicProfileData, 0, len(records))
	for _, record := range records {
		results = append(results, mapPublicProfileRecord(record, s.userMediaBaseURL))
	}

	return results, nil
}

func (s *Service) CreateAvatarUpload(ctx context.Context, userID string, req CreateAvatarUploadRequest) (CreateAvatarUploadData, error) {
	if strings.TrimSpace(userID) == "" || !isValidUUID(userID) {
		return CreateAvatarUploadData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	contentType, _, err := normalizeImageContentType(req.ContentType)
	if err != nil {
		return CreateAvatarUploadData{}, err
	}

	if req.ContentLength <= 0 {
		return CreateAvatarUploadData{}, sharederrors.Validation("contentLength must be greater than 0", nil)
	}

	if req.ContentLength > s.maxImageBytes {
		return CreateAvatarUploadData{}, sharederrors.Validation("contentLength exceeds max allowed size", nil)
	}

	objectKey := fmt.Sprintf("users/%s/avatar/%s.webp", userID, uuid.NewString())
	contentLength := req.ContentLength
	uploadURL, err := s.mediaStorage.PresignPutObject(ctx, storage.PresignPutObjectInput{
		ObjectKey:     objectKey,
		ContentType:   contentType,
		ContentLength: &contentLength,
		ExpiresIn:     s.mediaUploadTTL,
	})
	if err != nil {
		return CreateAvatarUploadData{}, sharederrors.Internal("failed to create avatar upload", nil)
	}

	return CreateAvatarUploadData{ObjectKey: objectKey, UploadURL: uploadURL}, nil
}

func (s *Service) CreateAccountDeletionRequest(ctx context.Context, userID string) (AccountDeletionRequestData, error) {
	if strings.TrimSpace(userID) == "" {
		return AccountDeletionRequestData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(userID) {
		return AccountDeletionRequestData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	record, err := s.repo.CreateAccountDeletionRequest(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AccountDeletionRequestData{}, sharederrors.NotFound("user profile not found", nil)
		}

		return AccountDeletionRequestData{}, sharederrors.Internal("failed to create account deletion request", nil)
	}

	return mapAccountDeletionRequestRecord(record), nil
}

func validateAndBuildPatch(req UpdateMeProfileRequest) (UpdateMeProfilePatch, error) {
	patch := UpdateMeProfilePatch{}

	if req.DisplayName != nil {
		displayName := strings.TrimSpace(*req.DisplayName)
		if displayName == "" {
			return UpdateMeProfilePatch{}, sharederrors.Validation("displayName must not be empty", nil)
		}

		if len(displayName) < 2 || len(displayName) > 80 {
			return UpdateMeProfilePatch{}, sharederrors.Validation("displayName length must be between 2 and 80", nil)
		}

		patch.DisplayNameSet = true
		patch.DisplayName = &displayName
	}

	if req.Username != nil {
		normalized, err := ValidateUsername(*req.Username)
		if err != nil {
			return UpdateMeProfilePatch{}, err
		}
		patch.UsernameSet = true
		patch.Username = &normalized
	}

	if req.Bio != nil {
		bio := strings.TrimSpace(*req.Bio)
		if len(bio) > maxBioLength {
			return UpdateMeProfilePatch{}, sharederrors.Validation(fmt.Sprintf("bio length must be at most %d", maxBioLength), nil)
		}

		patch.BioSet = true
		if bio == "" {
			patch.Bio = nil
		} else {
			patch.Bio = &bio
		}
	}

	if req.AvatarObjectKey != nil {
		patch.AvatarObjectKeySet = true
		avatarObjectKey := strings.TrimSpace(*req.AvatarObjectKey)
		if avatarObjectKey == "" {
			patch.AvatarObjectKey = nil
		} else {
			patch.AvatarObjectKey = &avatarObjectKey
		}
	}

	if !patch.DisplayNameSet && !patch.UsernameSet && !patch.BioSet && !patch.AvatarObjectKeySet {
		return UpdateMeProfilePatch{}, sharederrors.Validation("at least one field must be provided", nil)
	}

	return patch, nil
}

// ValidateUsername lowercases + trims input, enforces length/charset/reserved rules,
// and returns the normalized form (or a sharederrors.Validation error).
func ValidateUsername(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", sharederrors.Validation("username is required", nil)
	}
	if len(normalized) < minUsernameLength || len(normalized) > maxUsernameLength {
		return "", sharederrors.Validation(fmt.Sprintf("username length must be between %d and %d", minUsernameLength, maxUsernameLength), nil)
	}
	if !usernameFormatRegex.MatchString(normalized) {
		return "", sharederrors.Validation("username may only contain lowercase letters, numbers and underscores", nil)
	}
	if _, reserved := reservedUsernames[normalized]; reserved {
		return "", sharederrors.Validation("username is reserved", nil)
	}
	return normalized, nil
}

type optimizedAvatarResult struct {
	PrimaryObjectKey string
	Variants         []AvatarVariantRecord
	Blurhash         *string
}

func (s *Service) optimizeAndStoreAvatar(ctx context.Context, objectKey string) (optimizedAvatarResult, error) {
	content, err := s.mediaStorage.GetObject(ctx, storage.GetObjectInput{ObjectKey: objectKey})
	if err != nil {
		if storage.IsNotFoundError(err) {
			return optimizedAvatarResult{}, sharederrors.Validation("uploaded object was not found", nil)
		}

		return optimizedAvatarResult{}, sharederrors.Internal("failed to update profile", nil)
	}

	plan := buildAvatarVariantPlan(s.variantWidths, s.webpQuality)
	results, err := imageproc.OptimizeImageVariants(content, plan)
	if err != nil {
		return optimizedAvatarResult{}, sharederrors.Validation("uploaded object is not a valid image", nil)
	}

	primaryWidth := plan[len(plan)-1].MaxWidth
	primaryObjectKey := ""
	variants := make([]AvatarVariantRecord, 0, len(results))

	for index, res := range results {
		spec := plan[index]
		variantKey := avatarVariantObjectKey(objectKey, spec.MaxWidth, primaryWidth)
		if err := s.mediaStorage.PutObject(ctx, storage.PutObjectInput{
			ObjectKey:    variantKey,
			Content:      res.Content,
			ContentType:  res.ContentType,
			CacheControl: "public, max-age=31536000, immutable",
		}); err != nil {
			return optimizedAvatarResult{}, sharederrors.Internal("failed to update profile", nil)
		}

		variants = append(variants, AvatarVariantRecord{
			Width:     res.Width,
			Height:    res.Height,
			ObjectKey: variantKey,
			SizeBytes: int64(len(res.Content)),
		})

		if spec.MaxWidth == primaryWidth {
			primaryObjectKey = variantKey
		}
	}

	if primaryObjectKey == "" && len(variants) > 0 {
		primaryObjectKey = variants[len(variants)-1].ObjectKey
	}

	result := optimizedAvatarResult{
		PrimaryObjectKey: primaryObjectKey,
		Variants:         variants,
	}

	if s.blurhashEnabled {
		if hash, hashErr := imageproc.GenerateBlurhashFromBytes(content); hashErr == nil && hash != "" {
			h := hash
			result.Blurhash = &h
		}
	}

	return result, nil
}

func buildAvatarVariantPlan(widths []int, quality int) []imageproc.VariantSpec {
	if len(widths) == 0 {
		widths = []int{96, 240, 512}
	}
	sorted := make([]int, len(widths))
	copy(sorted, widths)
	for i := 1; i < len(sorted); i++ {
		j := i
		for j > 0 && sorted[j-1] > sorted[j] {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
			j--
		}
	}

	plan := make([]imageproc.VariantSpec, 0, len(sorted))
	for _, w := range sorted {
		plan = append(plan, imageproc.VariantSpec{
			MaxWidth:          w,
			MaxHeight:         w,
			Quality:           quality,
			TargetContentType: "image/webp",
		})
	}
	return plan
}

func avatarVariantObjectKey(originalKey string, width int, primaryWidth int) string {
	base := normalizeWebPObjectKey(originalKey)
	if width == primaryWidth {
		return base
	}
	lastDot := strings.LastIndex(base, ".")
	if lastDot <= 0 {
		return fmt.Sprintf("%s-%d", base, width)
	}
	return fmt.Sprintf("%s-%d%s", base[:lastDot], width, base[lastDot:])
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

func mapMeProfileRecord(record MeProfileRecord, userMediaBaseURL string) MeProfileData {
	return MeProfileData{
		ID:            record.ID,
		DisplayName:   record.DisplayName,
		Username:      record.Username,
		Email:         record.Email,
		Role:          record.Role,
		Bio:           record.Bio,
		AvatarURL:     buildAvatarURL(userMediaBaseURL, record.ID, record.AvatarObjectKey, record.AvatarVersion),
		AvatarVersion: &record.AvatarVersion,
		Stats: ProfileStatsData{
			FollowersCount: record.FollowersCount,
			FollowingCount: record.FollowingCount,
			WatchedCount:   record.WatchedCount,
			ReviewsCount:   record.ReviewsCount,
		},
	}
}

func mapPublicProfileRecord(record PublicProfileRecord, userMediaBaseURL string) PublicProfileData {
	return PublicProfileData{
		ID:            record.ID,
		DisplayName:   record.DisplayName,
		Username:      record.Username,
		Bio:           record.Bio,
		AvatarURL:     buildAvatarURL(userMediaBaseURL, record.ID, record.AvatarObjectKey, record.AvatarVersion),
		AvatarVersion: &record.AvatarVersion,
		Stats: ProfileStatsData{
			FollowersCount: record.FollowersCount,
			FollowingCount: record.FollowingCount,
			WatchedCount:   record.WatchedCount,
			ReviewsCount:   record.ReviewsCount,
		},
	}
}

func mapAccountDeletionRequestRecord(record AccountDeletionRequestRecord) AccountDeletionRequestData {
	return AccountDeletionRequestData{
		ID:          record.ID,
		Status:      record.Status,
		RequestedAt: record.RequestedAt,
	}
}

func isValidUUID(raw string) bool {
	_, err := uuid.Parse(strings.TrimSpace(raw))
	return err == nil
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

func validateImageMetadata(contentType string, contentLength int64, maxImageBytes int64) error {
	if _, _, err := normalizeImageContentType(contentType); err != nil {
		return err
	}

	if contentLength <= 0 || contentLength > maxImageBytes {
		return sharederrors.Validation("uploaded object size is invalid", nil)
	}

	return nil
}

func buildAvatarURL(baseURL string, userID string, avatarObjectKey *string, version string) *string {
	if avatarObjectKey == nil || strings.TrimSpace(*avatarObjectKey) == "" {
		return nil
	}

	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = defaultUserMediaPrefix
	}

	url := base + "/" + userID + "/avatar"
	if version != "" {
		url += "?v=" + version
	}
	return &url
}
