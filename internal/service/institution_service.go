package service

import (
	"context"
	"strings"
	"time"

	"acilkan.backend/internal/model"
	"acilkan.backend/internal/repository"
	appErrors "acilkan.backend/pkg/errors"
	"acilkan.backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InstitutionService handles verification of hospital / blood center accounts.
// Verification itself happens out of band: an admin calls the institution's
// official phone number before approving.
type InstitutionService struct {
	repo     *repository.InstitutionRepository
	userRepo *repository.UserRepository
}

func NewInstitutionService(repo *repository.InstitutionRepository, userRepo *repository.UserRepository) *InstitutionService {
	return &InstitutionService{repo: repo, userRepo: userRepo}
}

// ApplyInput is the body of POST /institution/applications
type ApplyInput struct {
	ApplicantName   string `json:"applicant_name"`
	ApplicantTitle  string `json:"applicant_title"`
	InstitutionName string `json:"institution_name"`
	Type            string `json:"type"`
	City            string `json:"city"`
	District        string `json:"district"`
	OfficialPhone   string `json:"official_phone"`
	OfficialEmail   string `json:"official_email"`
	Note            string `json:"note"`
}

// Validate trims and checks the application fields
func (in *ApplyInput) Validate() error {
	for _, f := range []*string{&in.ApplicantName, &in.ApplicantTitle, &in.InstitutionName,
		&in.City, &in.District, &in.OfficialPhone, &in.OfficialEmail, &in.Note} {
		*f = strings.TrimSpace(*f)
	}
	required := map[string]string{
		"applicant name":   in.ApplicantName,
		"applicant title":  in.ApplicantTitle,
		"institution name": in.InstitutionName,
		"city":             in.City,
		"official phone":   in.OfficialPhone,
	}
	for field, v := range required {
		if v == "" {
			return appErrors.NewWithMessage(appErrors.ErrCodeMissingRequiredField, field+" is required")
		}
	}
	if !model.InstitutionType(in.Type).IsValid() {
		return appErrors.NewWithMessage(appErrors.ErrCodeValidationFailed, "invalid institution type")
	}
	if !IsValidPhone(in.OfficialPhone) {
		return appErrors.New(appErrors.ErrCodeInvalidPhoneNumber, nil)
	}
	if in.OfficialEmail != "" && (!strings.Contains(in.OfficialEmail, "@") || len(in.OfficialEmail) > 200) {
		return appErrors.NewWithMessage(appErrors.ErrCodeValidationFailed, "invalid e-mail")
	}
	if len(in.InstitutionName) > 150 || len(in.Note) > 1000 {
		return appErrors.NewWithMessage(appErrors.ErrCodeValidationFailed, "field too long")
	}
	return nil
}

// Apply creates a pending application for the user
func (s *InstitutionService) Apply(ctx context.Context, uid string, in *ApplyInput) (*model.InstitutionApplication, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByUID(ctx, uid)
	if err != nil {
		return nil, appErrors.ErrUserNotFound()
	}
	if user.IsVerifiedInstitution() {
		return nil, appErrors.New(appErrors.ErrCodeAlreadyVerified, nil)
	}
	latest, err := s.repo.LatestForUser(ctx, uid)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}
	if latest != nil && latest.Status == model.ApplicationPending {
		return nil, appErrors.New(appErrors.ErrCodeApplicationPending, nil)
	}

	app := &model.InstitutionApplication{
		ID:              uuid.New().String(),
		UID:             uid,
		ApplicantName:   in.ApplicantName,
		ApplicantTitle:  in.ApplicantTitle,
		InstitutionName: in.InstitutionName,
		Type:            model.InstitutionType(in.Type),
		City:            in.City,
		District:        in.District,
		OfficialPhone:   in.OfficialPhone,
		OfficialEmail:   in.OfficialEmail,
		Note:            in.Note,
		Status:          model.ApplicationPending,
	}
	if err := s.repo.Create(ctx, app); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}
	logger.LogUserAction(uid, "institution_application_created",
		zap.String("application_id", app.ID), zap.String("institution", app.InstitutionName))
	return app, nil
}

// MyApplication returns the user's latest application, or nil
func (s *InstitutionService) MyApplication(ctx context.Context, uid string) (*model.InstitutionApplication, error) {
	app, err := s.repo.LatestForUser(ctx, uid)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}
	return app, nil
}

// List returns applications in a status for admins
func (s *InstitutionService) List(ctx context.Context, st model.ApplicationStatus) ([]*model.InstitutionApplication, error) {
	if !st.IsValid() {
		return nil, appErrors.NewWithMessage(appErrors.ErrCodeValidationFailed, "invalid status")
	}
	list, err := s.repo.ListByStatus(ctx, st, 100)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}
	return list, nil
}

// Approve marks the application approved and turns the applicant into a verified institution
func (s *InstitutionService) Approve(ctx context.Context, adminUID, id string) (*model.InstitutionApplication, error) {
	app, err := s.pending(ctx, id)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByUID(ctx, app.UID)
	if err != nil {
		return nil, appErrors.ErrUserNotFound()
	}

	now := time.Now()
	user.AddRole(model.RoleInstitution)
	user.Institution = &model.Institution{
		Name:       app.InstitutionName,
		Type:       app.Type,
		City:       app.City,
		District:   app.District,
		VerifiedAt: now,
		VerifiedBy: adminUID,
	}
	// Önce kullanıcı: uygulama güncellemesi başarısız olursa onay tekrar denenebilir
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrCodeUserUpdateFailed, err)
	}

	app.Status = model.ApplicationApproved
	app.ReviewedAt = &now
	app.ReviewedBy = adminUID
	if err := s.repo.Update(ctx, app); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}
	logger.LogUserAction(adminUID, "institution_application_approved",
		zap.String("application_id", id), zap.String("uid", app.UID))
	return app, nil
}

// Reject marks the application rejected with a reason shown to the applicant
func (s *InstitutionService) Reject(ctx context.Context, adminUID, id, reason string) (*model.InstitutionApplication, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, appErrors.NewWithMessage(appErrors.ErrCodeMissingRequiredField, "reason is required")
	}
	app, err := s.pending(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	app.Status = model.ApplicationRejected
	app.RejectReason = reason
	app.ReviewedAt = &now
	app.ReviewedBy = adminUID
	if err := s.repo.Update(ctx, app); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}
	logger.LogUserAction(adminUID, "institution_application_rejected", zap.String("application_id", id))
	return app, nil
}

func (s *InstitutionService) pending(ctx context.Context, id string) (*model.InstitutionApplication, error) {
	app, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, appErrors.New(appErrors.ErrCodeApplicationNotFound, nil)
		}
		return nil, appErrors.Wrap(appErrors.ErrCodeDatabaseError, err)
	}
	if app.Status != model.ApplicationPending {
		return nil, appErrors.New(appErrors.ErrCodeApplicationReviewed, nil)
	}
	return app, nil
}
