package users

import (
	"context"
	"errors"
	"strings"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxBioLength = 500

type repositoryPort interface {
	GetPublicProfile(ctx context.Context, userID string) (PublicProfileRecord, error)
	GetMeProfile(ctx context.Context, userID string) (MeProfileRecord, error)
	UpdateMeProfile(ctx context.Context, userID string, patch UpdateMeProfilePatch) (MeProfileRecord, error)
}

type Service struct {
	repo repositoryPort
}

func NewService(repo repositoryPort) *Service {
	return &Service{repo: repo}
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

	return mapPublicProfileRecord(record), nil
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

	return mapMeProfileRecord(record), nil
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

	record, err := s.repo.UpdateMeProfile(ctx, userID, patch)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MeProfileData{}, sharederrors.NotFound("user profile not found", nil)
		}

		return MeProfileData{}, sharederrors.Internal("failed to update profile", nil)
	}

	return mapMeProfileRecord(record), nil
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

	if req.Bio != nil {
		bio := strings.TrimSpace(*req.Bio)
		if len(bio) > maxBioLength {
			return UpdateMeProfilePatch{}, sharederrors.Validation("bio length must be at most 500", nil)
		}

		patch.BioSet = true
		if bio == "" {
			patch.Bio = nil
		} else {
			patch.Bio = &bio
		}
	}

	if !patch.DisplayNameSet && !patch.BioSet {
		return UpdateMeProfilePatch{}, sharederrors.Validation("at least one field must be provided", nil)
	}

	return patch, nil
}

func mapMeProfileRecord(record MeProfileRecord) MeProfileData {
	return MeProfileData{
		ID:          record.ID,
		DisplayName: record.DisplayName,
		Email:       record.Email,
		Role:        record.Role,
		Bio:         record.Bio,
		Stats: ProfileStatsData{
			FollowersCount: record.FollowersCount,
			FollowingCount: record.FollowingCount,
			WatchedCount:   record.WatchedCount,
			ReviewsCount:   record.ReviewsCount,
		},
	}
}

func mapPublicProfileRecord(record PublicProfileRecord) PublicProfileData {
	return PublicProfileData{
		ID:          record.ID,
		DisplayName: record.DisplayName,
		Bio:         record.Bio,
		Stats: ProfileStatsData{
			FollowersCount: record.FollowersCount,
			FollowingCount: record.FollowingCount,
			WatchedCount:   record.WatchedCount,
			ReviewsCount:   record.ReviewsCount,
		},
	}
}

func isValidUUID(raw string) bool {
	_, err := uuid.Parse(strings.TrimSpace(raw))
	return err == nil
}
