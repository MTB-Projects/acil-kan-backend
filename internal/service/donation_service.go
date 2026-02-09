package service

import (
	"context"
	"time"

	"acilkan.backend/internal/model"
	"acilkan.backend/internal/notification"
	"acilkan.backend/internal/repository"
	appErrors "acilkan.backend/pkg/errors"
	"acilkan.backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	MaxRequestsPerDay        = 3
	DefaultRequestExpiration = 48 * time.Hour // 48 hours
)

type DonationService struct {
	requestRepo *repository.BloodRequestRepository
	userRepo    *repository.UserRepository
	fcmClient   *notification.FCMClient
}

func NewDonationService(
	requestRepo *repository.BloodRequestRepository,
	userRepo *repository.UserRepository,
	fcmClient *notification.FCMClient,
) *DonationService {
	return &DonationService{
		requestRepo: requestRepo,
		userRepo:    userRepo,
		fcmClient:   fcmClient,
	}
}

// CreateRequest creates a new blood donation request
// Implements spam prevention and validation
func (s *DonationService) CreateRequest(ctx context.Context, uid string, req *CreateRequestInput) (*model.BloodRequest, error) {
	logger.Info("Creating blood request", zap.String("uid", uid), zap.String("blood_type", req.BloodType))

	// Get requester profile
	requester, err := s.userRepo.GetByUID(ctx, uid)
	if err != nil {
		logger.Error("User profile not found for request", zap.String("uid", uid), zap.Error(err))
		return nil, appErrors.ErrUserNotFound()
	}

	// Validate user has completed profile
	if requester.City == "" || requester.PhoneNumber == "" {
		logger.Warn("Incomplete user profile", zap.String("uid", uid))
		return nil, appErrors.ErrIncompleteProfile()
	}

	// Check spam prevention - max requests per day
	count, err := s.requestRepo.CountUserActiveRequestsToday(ctx, uid)
	if err != nil {
		logger.Error("Failed to count user requests", zap.String("uid", uid), zap.Error(err))
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}
	if count >= MaxRequestsPerDay {
		logger.Warn("User exceeded daily request limit", zap.String("uid", uid), zap.Int("count", count))
		return nil, appErrors.ErrRequestLimitExceeded(MaxRequestsPerDay)
	}

	// Validate input
	if err := req.Validate(); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrCodeInvalidRequestData, err)
	}

	// Create blood request
	bloodRequest := &model.BloodRequest{
		ID:              uuid.New().String(),
		RequesterUID:    uid,
		RequesterName:   requester.FullName,
		PatientName:     req.PatientName,
		BloodType:       model.BloodType(req.BloodType),
		City:            req.City,
		District:        req.District, // Optional district for more specific targeting
		HospitalName:    req.HospitalName,
		HospitalAddress: req.HospitalAddress,
		ContactPhone:    req.ContactPhone,
		UnitsNeeded:     req.UnitsNeeded,
		Description:     req.Description,
		Status:          model.RequestStatusActive,
		ExpiresAt:       time.Now().Add(DefaultRequestExpiration),
	}

	if err := s.requestRepo.Create(ctx, bloodRequest); err != nil {
		logger.Error("Failed to create blood request", zap.String("uid", uid), zap.Error(err))
		return nil, appErrors.Wrap(appErrors.ErrCodeRequestCreationFailed, err)
	}

	logger.LogUserAction(uid, "blood_request_created",
		zap.String("request_id", bloodRequest.ID),
		zap.String("blood_type", string(bloodRequest.BloodType)),
		zap.String("city", bloodRequest.City),
	)

	// Send notifications to compatible donors asynchronously
	go s.notifyCompatibleDonors(context.Background(), bloodRequest)

	return bloodRequest, nil
}

// notifyCompatibleDonors sends notification to topic for scalable delivery
// Uses Firebase Cloud Messaging topics instead of individual tokens
func (s *DonationService) notifyCompatibleDonors(ctx context.Context, request *model.BloodRequest) {
	logger.Info("Sending blood request notification to topic",
		zap.String("request_id", request.ID),
		zap.String("blood_type", string(request.BloodType)),
		zap.String("city", request.City),
	)

	// Send notification to topic (much more scalable than individual tokens)
	err := s.fcmClient.SendBloodRequestNotificationToTopic(ctx, request)
	if err != nil {
		logger.Error("Failed to send topic notification",
			zap.String("request_id", request.ID),
			zap.Error(err),
		)
		return
	}

	logger.Info("Blood request notification sent successfully",
		zap.String("request_id", request.ID),
		zap.String("topic", notification.GenerateTopicName(request.BloodType, request.City)),
	)
}

// GetActiveRequests retrieves all active blood requests
func (s *DonationService) GetActiveRequests(ctx context.Context) ([]*model.BloodRequest, error) {
	return s.requestRepo.GetActiveRequests(ctx)
}

// GetActiveRequestsWithFilters retrieves active blood requests with optional filters
func (s *DonationService) GetActiveRequestsWithFilters(ctx context.Context, city, district, bloodType string) ([]*model.BloodRequest, error) {
	filters := repository.RequestFilters{
		City:      city,
		District:  district,
		BloodType: bloodType,
	}
	return s.requestRepo.GetActiveRequestsWithFilters(ctx, filters)
}

// GetUserRequests retrieves all requests created by a user
func (s *DonationService) GetUserRequests(ctx context.Context, uid string) ([]*model.BloodRequest, error) {
	return s.requestRepo.GetUserRequests(ctx, uid)
}

// CancelRequest cancels a blood request
func (s *DonationService) CancelRequest(ctx context.Context, uid, requestID string) error {
	logger.Info("Cancelling blood request", zap.String("uid", uid), zap.String("request_id", requestID))

	request, err := s.requestRepo.GetByID(ctx, requestID)
	if err != nil {
		logger.Error("Request not found for cancellation", zap.String("request_id", requestID), zap.Error(err))
		return appErrors.ErrRequestNotFound()
	}

	if !request.CanBeCancelledBy(uid) {
		logger.Warn("Unauthorized cancellation attempt", zap.String("uid", uid), zap.String("request_id", requestID))
		return appErrors.ErrUnauthorizedAction()
	}

	request.Status = model.RequestStatusCancelled
	err = s.requestRepo.Update(ctx, request)
	if err != nil {
		logger.Error("Failed to cancel request", zap.String("request_id", requestID), zap.Error(err))
		return appErrors.Wrap(appErrors.ErrCodeRequestUpdateFailed, err)
	}

	logger.LogUserAction(uid, "blood_request_cancelled", zap.String("request_id", requestID))
	return nil
}

// CreateRequestInput represents input for creating a blood request
type CreateRequestInput struct {
	PatientName     string `json:"patient_name"`
	BloodType       string `json:"blood_type"`
	City            string `json:"city"`
	District        string `json:"district,omitempty"` // Optional district for more specific targeting
	HospitalName    string `json:"hospital_name"`
	HospitalAddress string `json:"hospital_address"`
	ContactPhone    string `json:"contact_phone"`
	UnitsNeeded     int    `json:"units_needed"`
	Description     string `json:"description"`
}

// Validate validates the create request input
func (i *CreateRequestInput) Validate() error {
	if i.PatientName == "" {
		return appErrors.NewWithMessage(appErrors.ErrCodeMissingRequiredField, "patient name is required")
	}
	if i.BloodType == "" {
		return appErrors.NewWithMessage(appErrors.ErrCodeMissingRequiredField, "blood type is required")
	}
	if i.City == "" {
		return appErrors.NewWithMessage(appErrors.ErrCodeMissingRequiredField, "city is required")
	}
	if i.HospitalName == "" {
		return appErrors.NewWithMessage(appErrors.ErrCodeMissingRequiredField, "hospital name is required")
	}
	if i.ContactPhone == "" {
		return appErrors.NewWithMessage(appErrors.ErrCodeMissingRequiredField, "contact phone is required")
	}
	if i.UnitsNeeded <= 0 {
		return appErrors.NewWithMessage(appErrors.ErrCodeInvalidRequestData, "units needed must be greater than 0")
	}
	return nil
}
