package service

import (
	"context"
	"strings"
	"time"

	"acilkan.backend/internal/model"
	"acilkan.backend/internal/notification"
	"acilkan.backend/internal/repository"
	appErrors "acilkan.backend/pkg/errors"
	"acilkan.backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	MaxRequestsPerDay        = 3
	// Doğrulanmış kurumlar (hastane kan merkezleri) gün içinde çok sayıda ilan verebilir
	MaxRequestsPerDayInstitution = 20
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

	// Validate input
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Spam prevention: rolling 24h window, cancelled requests still count
	count, err := s.requestRepo.CountUserRequestsSince(ctx, uid, time.Now().Add(-24*time.Hour))
	if err != nil {
		logger.Error("Failed to count user requests", zap.String("uid", uid), zap.Error(err))
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}
	limit := MaxRequestsPerDay
	if requester.IsVerifiedInstitution() {
		limit = MaxRequestsPerDayInstitution
	}
	if count >= limit {
		logger.Warn("User exceeded daily request limit", zap.String("uid", uid), zap.Int("count", count))
		return nil, appErrors.ErrRequestLimitExceeded(limit)
	}

	// Create blood request
	bloodRequest := &model.BloodRequest{
		ID:              uuid.New().String(),
		RequesterUID:    uid,
		RequesterName:   requester.FullName,
		PatientName:     req.PatientName,
		BloodType:       req.parsedBloodType,
		ProductType:     model.ProductType(req.ProductType),
		Urgency:         model.Urgency(req.Urgency),
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
	if requester.IsVerifiedInstitution() {
		bloodRequest.VerifiedInstitution = requester.Institution.Name
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
		zap.Strings("topics", notification.RecipientTopics(request)),
	)
}

// GetActiveRequests retrieves all active blood requests without personal data
func (s *DonationService) GetActiveRequests(ctx context.Context) ([]*model.PublicBloodRequest, error) {
	requests, err := s.requestRepo.GetActiveRequests(ctx)
	if err != nil {
		return nil, err
	}
	return toPublic(requests), nil
}

// GetActiveRequestsWithFilters retrieves active blood requests with optional filters, without personal data
func (s *DonationService) GetActiveRequestsWithFilters(ctx context.Context, city, district, bloodType string) ([]*model.PublicBloodRequest, error) {
	filters := repository.RequestFilters{
		City:     city,
		District: district,
	}
	if bloodType != "" {
		bt, ok := model.ParseBloodType(bloodType)
		if !ok {
			return nil, appErrors.New(appErrors.ErrCodeInvalidBloodType, nil)
		}
		filters.BloodType = string(bt)
	}
	requests, err := s.requestRepo.GetActiveRequestsWithFilters(ctx, filters)
	if err != nil {
		return nil, err
	}
	return toPublic(requests), nil
}

// GetRequestContact returns an active request with contact details, for authenticated users
func (s *DonationService) GetRequestContact(ctx context.Context, requestID string) (*model.ContactView, error) {
	request, err := s.requestRepo.GetByID(ctx, requestID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, appErrors.ErrRequestNotFound()
		}
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}
	if !request.IsActive() {
		return nil, appErrors.New(appErrors.ErrCodeRequestExpired, nil)
	}
	return request.ToContactView(), nil
}

func toPublic(requests []*model.BloodRequest) []*model.PublicBloodRequest {
	out := make([]*model.PublicBloodRequest, 0, len(requests))
	for _, r := range requests {
		out = append(out, r.ToPublic())
	}
	return out
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
	ProductType     string `json:"product_type"`
	Urgency         string `json:"urgency"`
	City            string `json:"city"`
	District        string `json:"district,omitempty"` // Optional district for more specific targeting
	HospitalName    string `json:"hospital_name"`
	HospitalAddress string `json:"hospital_address"`
	ContactPhone    string `json:"contact_phone"`
	UnitsNeeded     int    `json:"units_needed"`
	Description     string `json:"description"`

	parsedBloodType model.BloodType
}

// Validate validates and normalizes the create request input
func (i *CreateRequestInput) Validate() error {
	i.PatientName = strings.TrimSpace(i.PatientName)
	i.City = strings.TrimSpace(i.City)
	i.District = strings.TrimSpace(i.District)
	i.HospitalName = strings.TrimSpace(i.HospitalName)
	i.ContactPhone = strings.TrimSpace(i.ContactPhone)

	if i.PatientName == "" {
		return appErrors.NewWithMessage(appErrors.ErrCodeMissingRequiredField, "patient name is required")
	}
	if i.BloodType == "" {
		return appErrors.NewWithMessage(appErrors.ErrCodeMissingRequiredField, "blood type is required")
	}
	bt, ok := model.ParseBloodType(i.BloodType)
	if !ok {
		return appErrors.New(appErrors.ErrCodeInvalidBloodType, nil)
	}
	i.parsedBloodType = bt

	if i.ProductType == "" {
		i.ProductType = string(model.ProductWholeBlood)
	}
	if !model.ProductType(i.ProductType).IsValid() {
		return appErrors.NewWithMessage(appErrors.ErrCodeInvalidRequestData, "invalid product type")
	}
	if i.Urgency == "" {
		i.Urgency = string(model.UrgencyNormal)
	}
	if !model.Urgency(i.Urgency).IsValid() {
		return appErrors.NewWithMessage(appErrors.ErrCodeInvalidRequestData, "invalid urgency")
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
	if !IsValidPhone(i.ContactPhone) {
		return appErrors.New(appErrors.ErrCodeInvalidPhoneNumber, nil)
	}
	if i.UnitsNeeded <= 0 || i.UnitsNeeded > 20 {
		return appErrors.NewWithMessage(appErrors.ErrCodeInvalidRequestData, "units needed must be between 1 and 20")
	}
	if len(i.Description) > 1000 {
		return appErrors.NewWithMessage(appErrors.ErrCodeInvalidRequestData, "description is too long")
	}
	return nil
}
