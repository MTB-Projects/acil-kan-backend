package service

import (
	"context"
	"strings"

	"acilkan.backend/internal/model"
	"acilkan.backend/internal/notification"
	"acilkan.backend/internal/repository"
	appErrors "acilkan.backend/pkg/errors"
	"acilkan.backend/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	userRepo  *repository.UserRepository
	fcmClient *notification.FCMClient
}

func NewUserService(userRepo *repository.UserRepository, fcmClient *notification.FCMClient) *UserService {
	return &UserService{
		userRepo:  userRepo,
		fcmClient: fcmClient,
	}
}

// GetOrCreateUser gets user profile or creates one if it doesn't exist
// This is called on first successful login
func (s *UserService) GetOrCreateUser(ctx context.Context, uid, email string) (*model.User, error) {
	// Try to get existing user
	user, err := s.userRepo.GetByUID(ctx, uid)
	if err != nil {
		// Check if error is "not found"
		if status.Code(err) == codes.NotFound {
			logger.Info("Creating new user profile", zap.String("uid", uid), zap.String("email", email))
			// Create new user with default values
			user = &model.User{
				UID:     uid,
				Email:   email,
				Roles:   []model.UserRole{model.RoleUser},
				IsDonor: false,
			}
			if err := s.userRepo.Create(ctx, user); err != nil {
				logger.Error("Failed to create user", zap.String("uid", uid), zap.Error(err))
				return nil, err
			}
			logger.LogUserAction(uid, "user_created", zap.String("email", email))
			return user, nil
		}
		logger.Error("Failed to get user", zap.String("uid", uid), zap.Error(err))
		return nil, err
	}

	logger.Debug("User profile retrieved", zap.String("uid", uid))
	return user, nil
}

// UpdateProfile updates user profile information
func (s *UserService) UpdateProfile(ctx context.Context, uid string, updates map[string]interface{}) (*model.User, error) {
	logger.Info("Updating user profile", zap.String("uid", uid))
	user, err := s.userRepo.GetByUID(ctx, uid)
	if err != nil {
		logger.Error("Failed to get user for update", zap.String("uid", uid), zap.Error(err))
		if status.Code(err) == codes.NotFound {
			return nil, appErrors.ErrUserNotFound()
		}
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}

	before := *user

	// Apply updates
	if fullName, ok := updates["full_name"].(string); ok {
		user.FullName = strings.TrimSpace(fullName)
	}
	if phoneNumber, ok := updates["phone_number"].(string); ok {
		phoneNumber = strings.TrimSpace(phoneNumber)
		if phoneNumber != "" && !IsValidPhone(phoneNumber) {
			return nil, appErrors.New(appErrors.ErrCodeInvalidPhoneNumber, nil)
		}
		user.PhoneNumber = phoneNumber
	}
	if bloodType, ok := updates["blood_type"].(string); ok {
		bt, valid := model.ParseBloodType(bloodType)
		if !valid {
			return nil, appErrors.New(appErrors.ErrCodeInvalidBloodType, nil)
		}
		user.BloodType = bt
	}
	if city, ok := updates["city"].(string); ok {
		user.City = strings.TrimSpace(city)
	}
	if isDonor, ok := updates["is_donor"].(bool); ok {
		user.IsDonor = isDonor
	}
	if district, ok := updates["district"].(string); ok {
		user.District = strings.TrimSpace(district)
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		logger.Error("Failed to update user profile", zap.String("uid", uid), zap.Error(err))
		return nil, err
	}

	s.syncTopics(ctx, &before, user)

	logger.LogUserAction(uid, "profile_updated")
	return user, nil
}

// isSubscribable reports whether a user should receive blood request notifications
func isSubscribable(u *model.User) bool {
	return u.IsDonor && u.BloodType != "" && u.City != "" && u.FCMToken != ""
}

// syncTopics moves the user's FCM subscriptions when donor status, blood type or city changes.
// Failures are logged but do not fail the profile update.
func (s *UserService) syncTopics(ctx context.Context, before, after *model.User) {
	if before.IsDonor == after.IsDonor && before.BloodType == after.BloodType &&
		before.City == after.City && before.FCMToken == after.FCMToken {
		return
	}

	if isSubscribable(before) {
		if err := s.fcmClient.UnsubscribeFromTopics(ctx, before.FCMToken, before.BloodType, before.City); err != nil {
			logger.Error("Failed to unsubscribe old topics", zap.String("uid", after.UID), zap.Error(err))
		}
	}
	if isSubscribable(after) {
		if err := s.fcmClient.SubscribeToTopics(ctx, after.FCMToken, after.BloodType, after.City); err != nil {
			logger.Error("Failed to subscribe new topics", zap.String("uid", after.UID), zap.Error(err))
		}
	}
}

// UpdateFCMToken updates the FCM token for push notifications
// and subscribes the user to relevant topics based on their blood type and city
func (s *UserService) UpdateFCMToken(ctx context.Context, uid, token string) error {
	if token == "" {
		return appErrors.ErrInvalidRequestBody()
	}
	
	logger.Info("Updating FCM token", zap.String("uid", uid))
	
	// Get user profile to determine topics
	user, err := s.userRepo.GetByUID(ctx, uid)
	if err != nil {
		logger.Error("Failed to get user for FCM token update", zap.String("uid", uid), zap.Error(err))
		return appErrors.Wrap(appErrors.ErrCodeUserUpdateFailed, err)
	}
	
	// Update FCM token in database
	err = s.userRepo.UpdateFCMToken(ctx, uid, token)
	if err != nil {
		logger.Error("Failed to update FCM token", zap.String("uid", uid), zap.Error(err))
		return appErrors.Wrap(appErrors.ErrCodeUserUpdateFailed, err)
	}

	// Move subscriptions from the old token (if any) to the new one.
	// Topic subscription failures don't fail the request; the token is still saved.
	before := *user
	user.FCMToken = token
	if before.FCMToken == token && isSubscribable(user) {
		// Same token re-sent (e.g. app restart): re-subscribe, it is idempotent
		// and repairs a subscription that failed earlier.
		if err := s.fcmClient.SubscribeToTopics(ctx, token, user.BloodType, user.City); err != nil {
			logger.Error("Failed to re-subscribe topics", zap.String("uid", uid), zap.Error(err))
		}
	} else {
		s.syncTopics(ctx, &before, user)
	}

	logger.LogUserAction(uid, "fcm_token_updated")
	return nil
}

// GetProfile retrieves user profile
func (s *UserService) GetProfile(ctx context.Context, uid string) (*model.User, error) {
	return s.userRepo.GetByUID(ctx, uid)
}
