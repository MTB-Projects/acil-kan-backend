package service

import (
	"context"

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
		return nil, err
	}

	// Apply updates
	if fullName, ok := updates["full_name"].(string); ok {
		user.FullName = fullName
	}
	if phoneNumber, ok := updates["phone_number"].(string); ok {
		user.PhoneNumber = phoneNumber
	}
	if bloodType, ok := updates["blood_type"].(string); ok {
		user.BloodType = model.BloodType(bloodType)
	}
	if city, ok := updates["city"].(string); ok {
		user.City = city
	}
	if isDonor, ok := updates["is_donor"].(bool); ok {
		user.IsDonor = isDonor
	}
	if district, ok := updates["district"].(string); ok {
		user.District = district
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		logger.Error("Failed to update user profile", zap.String("uid", uid), zap.Error(err))
		return nil, err
	}

	logger.LogUserAction(uid, "profile_updated")
	return user, nil
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
	
	// Subscribe to topics if user is a donor and has blood type + city
	if user.IsDonor && user.BloodType != "" && user.City != "" {
		err = s.fcmClient.SubscribeToTopics(ctx, token, user.BloodType, user.City, user.District)
		if err != nil {
			logger.Error("Failed to subscribe to topics",
				zap.String("uid", uid),
				zap.String("blood_type", string(user.BloodType)),
				zap.String("city", user.City),
				zap.String("district", user.District),
				zap.Error(err),
			)
			// Don't fail the request if topic subscription fails
			// Token is still updated in database
		}
	}
	
	logger.LogUserAction(uid, "fcm_token_updated")
	return nil
}

// GetProfile retrieves user profile
func (s *UserService) GetProfile(ctx context.Context, uid string) (*model.User, error) {
	return s.userRepo.GetByUID(ctx, uid)
}
