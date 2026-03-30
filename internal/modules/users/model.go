package users

type ProfileStatsData struct {
	FollowersCount int64 `json:"followersCount"`
	FollowingCount int64 `json:"followingCount"`
	WatchedCount   int64 `json:"watchedCount"`
	ReviewsCount   int64 `json:"reviewsCount"`
}

type MeProfileData struct {
	ID          string           `json:"id"`
	DisplayName string           `json:"displayName"`
	Email       string           `json:"email"`
	Role        string           `json:"role"`
	Bio         *string          `json:"bio"`
	Stats       ProfileStatsData `json:"stats"`
}

type PublicProfileData struct {
	ID          string           `json:"id"`
	DisplayName string           `json:"displayName"`
	Bio         *string          `json:"bio"`
	Stats       ProfileStatsData `json:"stats"`
}

type UpdateMeProfileRequest struct {
	DisplayName *string `json:"displayName"`
	Bio         *string `json:"bio"`
}

type UpdateMeProfilePatch struct {
	DisplayNameSet bool
	DisplayName    *string
	BioSet         bool
	Bio            *string
}

type MeProfileRecord struct {
	ID             string
	DisplayName    string
	Email          string
	Role           string
	Bio            *string
	FollowersCount int64
	FollowingCount int64
	WatchedCount   int64
	ReviewsCount   int64
}

type PublicProfileRecord struct {
	ID             string
	DisplayName    string
	Bio            *string
	FollowersCount int64
	FollowingCount int64
	WatchedCount   int64
	ReviewsCount   int64
}
