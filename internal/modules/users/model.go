package users

type ProfileStatsData struct {
	FollowersCount int64 `json:"followersCount"`
	FollowingCount int64 `json:"followingCount"`
	WatchedCount   int64 `json:"watchedCount"`
	ReviewsCount   int64 `json:"reviewsCount"`
}

type MeProfileData struct {
	ID            string           `json:"id"`
	DisplayName   string           `json:"displayName"`
	Username      string           `json:"username"`
	Email         string           `json:"email"`
	Role          string           `json:"role"`
	Bio           *string          `json:"bio"`
	AvatarURL     *string          `json:"avatarUrl"`
	AvatarVersion *string          `json:"avatarVersion"`
	Stats         ProfileStatsData `json:"stats"`
}

type PublicProfileData struct {
	ID            string           `json:"id"`
	DisplayName   string           `json:"displayName"`
	Username      string           `json:"username"`
	Bio           *string          `json:"bio"`
	AvatarURL     *string          `json:"avatarUrl"`
	AvatarVersion *string          `json:"avatarVersion"`
	Stats         ProfileStatsData `json:"stats"`
}

type UpdateMeProfileRequest struct {
	DisplayName     *string `json:"displayName"`
	Username        *string `json:"username"`
	Bio             *string `json:"bio"`
	AvatarObjectKey *string `json:"avatarObjectKey"`
}

type CreateAvatarUploadRequest struct {
	ContentType   string `json:"contentType"`
	ContentLength int64  `json:"contentLength"`
}

type CreateAvatarUploadData struct {
	ObjectKey string `json:"objectKey"`
	UploadURL string `json:"uploadUrl"`
}

type AccountDeletionRequestData struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	RequestedAt string `json:"requestedAt"`
}

type UpdateMeProfilePatch struct {
	DisplayNameSet     bool
	DisplayName        *string
	UsernameSet        bool
	Username           *string
	BioSet             bool
	Bio                *string
	AvatarObjectKeySet bool
	AvatarObjectKey    *string
	AvatarVariants     []AvatarVariantRecord
	AvatarBlurhash     *string
}

type AvatarVariantRecord struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	ObjectKey string `json:"objectKey"`
	SizeBytes int64  `json:"sizeBytes"`
}

type MeProfileRecord struct {
	ID              string
	DisplayName     string
	Username        string
	Email           string
	Role            string
	Bio             *string
	AvatarObjectKey *string
	AvatarVersion   string
	FollowersCount  int64
	FollowingCount  int64
	WatchedCount    int64
	ReviewsCount    int64
}

type PublicProfileRecord struct {
	ID              string
	DisplayName     string
	Username        string
	Bio             *string
	AvatarObjectKey *string
	AvatarVersion   string
	FollowersCount  int64
	FollowingCount  int64
	WatchedCount    int64
	ReviewsCount    int64
}

type AccountDeletionRequestRecord struct {
	ID          string
	Status      string
	RequestedAt string
}
