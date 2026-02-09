package errors

import "errors"

// Error codes for the application
const (
	// Authentication & Authorization Errors (1xxx)
	ErrCodeUnauthorized          = "AUTH_1001"
	ErrCodeInvalidToken          = "AUTH_1002"
	ErrCodeMissingToken          = "AUTH_1003"
	ErrCodeExpiredToken          = "AUTH_1004"
	ErrCodeInsufficientPermission = "AUTH_1005"

	// User Errors (2xxx)
	ErrCodeUserNotFound        = "USER_2001"
	ErrCodeUserAlreadyExists   = "USER_2002"
	ErrCodeIncompleteProfile   = "USER_2003"
	ErrCodeInvalidUserData     = "USER_2004"
	ErrCodeUserCreationFailed  = "USER_2005"
	ErrCodeUserUpdateFailed    = "USER_2006"
	ErrCodeInvalidFCMToken     = "USER_2007"

	// Blood Request Errors (3xxx)
	ErrCodeRequestNotFound       = "REQUEST_3001"
	ErrCodeRequestCreationFailed = "REQUEST_3002"
	ErrCodeRequestUpdateFailed   = "REQUEST_3003"
	ErrCodeRequestExpired        = "REQUEST_3004"
	ErrCodeRequestAlreadyCancelled = "REQUEST_3005"
	ErrCodeUnauthorizedAction    = "REQUEST_3006"
	ErrCodeInvalidRequestData    = "REQUEST_3007"
	ErrCodeRequestLimitExceeded  = "REQUEST_3008"

	// Validation Errors (4xxx)
	ErrCodeValidationFailed    = "VALIDATION_4001"
	ErrCodeInvalidBloodType    = "VALIDATION_4002"
	ErrCodeInvalidCity         = "VALIDATION_4003"
	ErrCodeInvalidPhoneNumber  = "VALIDATION_4004"
	ErrCodeMissingRequiredField = "VALIDATION_4005"
	ErrCodeInvalidRequestBody  = "VALIDATION_4006"

	// Database Errors (5xxx)
	ErrCodeDatabaseError       = "DB_5001"
	ErrCodeDatabaseQueryFailed = "DB_5002"
	ErrCodeDatabaseWriteFailed = "DB_5003"
	ErrCodeDatabaseReadFailed  = "DB_5004"

	// Notification Errors (6xxx)
	ErrCodeNotificationFailed = "NOTIFICATION_6001"
	ErrCodeFCMError           = "NOTIFICATION_6002"
	ErrCodeNoEligibleDonors   = "NOTIFICATION_6003"

	// General Errors (9xxx)
	ErrCodeInternalError    = "GENERAL_9001"
	ErrCodeServiceUnavailable = "GENERAL_9002"
	ErrCodeUnknownError     = "GENERAL_9999"
)

// Error messages
var ErrorMessages = map[string]string{
	// Authentication & Authorization
	ErrCodeUnauthorized:          "Unauthorized access",
	ErrCodeInvalidToken:          "Invalid or expired authentication token",
	ErrCodeMissingToken:          "Authentication token is required",
	ErrCodeExpiredToken:          "Authentication token has expired",
	ErrCodeInsufficientPermission: "Insufficient permissions for this action",

	// User
	ErrCodeUserNotFound:        "User profile not found",
	ErrCodeUserAlreadyExists:   "User already exists",
	ErrCodeIncompleteProfile:   "Please complete your profile before proceeding",
	ErrCodeInvalidUserData:     "Invalid user data provided",
	ErrCodeUserCreationFailed:  "Failed to create user profile",
	ErrCodeUserUpdateFailed:    "Failed to update user profile",
	ErrCodeInvalidFCMToken:     "Invalid FCM token provided",

	// Blood Request
	ErrCodeRequestNotFound:       "Blood request not found",
	ErrCodeRequestCreationFailed: "Failed to create blood request",
	ErrCodeRequestUpdateFailed:   "Failed to update blood request",
	ErrCodeRequestExpired:        "Blood request has expired",
	ErrCodeRequestAlreadyCancelled: "Blood request is already cancelled",
	ErrCodeUnauthorizedAction:    "You are not authorized to perform this action",
	ErrCodeInvalidRequestData:    "Invalid blood request data",
	ErrCodeRequestLimitExceeded:  "You have exceeded the maximum number of requests allowed per day",

	// Validation
	ErrCodeValidationFailed:    "Validation failed",
	ErrCodeInvalidBloodType:    "Invalid blood type",
	ErrCodeInvalidCity:         "Invalid city",
	ErrCodeInvalidPhoneNumber:  "Invalid phone number format",
	ErrCodeMissingRequiredField: "Required field is missing",
	ErrCodeInvalidRequestBody:  "Invalid request body format",

	// Database
	ErrCodeDatabaseError:       "Database operation failed",
	ErrCodeDatabaseQueryFailed: "Database query failed",
	ErrCodeDatabaseWriteFailed: "Failed to write to database",
	ErrCodeDatabaseReadFailed:  "Failed to read from database",

	// Notification
	ErrCodeNotificationFailed: "Failed to send notification",
	ErrCodeFCMError:           "Firebase Cloud Messaging error",
	ErrCodeNoEligibleDonors:   "No eligible donors found",

	// General
	ErrCodeInternalError:    "Internal server error",
	ErrCodeServiceUnavailable: "Service temporarily unavailable",
	ErrCodeUnknownError:     "An unknown error occurred",
}

// AppError represents an application error with code and message
type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// New creates a new AppError
func New(code string, err error) *AppError {
	message, ok := ErrorMessages[code]
	if !ok {
		message = ErrorMessages[ErrCodeUnknownError]
		code = ErrCodeUnknownError
	}

	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewWithMessage creates a new AppError with custom message
func NewWithMessage(code, customMessage string) *AppError {
	return &AppError{
		Code:    code,
		Message: customMessage,
		Err:     nil,
	}
}

// Wrap wraps an existing error with an error code
func Wrap(code string, err error) *AppError {
	if err == nil {
		return nil
	}
	return New(code, err)
}

// Common error constructors for convenience

func ErrUnauthorized(err error) *AppError {
	return New(ErrCodeUnauthorized, err)
}

func ErrInvalidToken() *AppError {
	return New(ErrCodeInvalidToken, nil)
}

func ErrUserNotFound() *AppError {
	return New(ErrCodeUserNotFound, nil)
}

func ErrIncompleteProfile() *AppError {
	return New(ErrCodeIncompleteProfile, nil)
}

func ErrRequestNotFound() *AppError {
	return New(ErrCodeRequestNotFound, nil)
}

func ErrUnauthorizedAction() *AppError {
	return New(ErrCodeUnauthorizedAction, nil)
}

func ErrRequestLimitExceeded(limit int) *AppError {
	return NewWithMessage(ErrCodeRequestLimitExceeded, 
		ErrorMessages[ErrCodeRequestLimitExceeded])
}

func ErrValidationFailed(field string) *AppError {
	return NewWithMessage(ErrCodeValidationFailed, 
		field + " validation failed")
}

func ErrInvalidRequestBody() *AppError {
	return New(ErrCodeInvalidRequestBody, nil)
}

func ErrDatabaseOperation(err error) *AppError {
	return New(ErrCodeDatabaseError, err)
}

func ErrInternalError(err error) *AppError {
	return New(ErrCodeInternalError, err)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError extracts AppError from error
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	// If not an AppError, wrap it as internal error
	return ErrInternalError(err)
}

// Standard Go errors for internal use
var (
	ErrEmptyFCMToken = errors.New("FCM token cannot be empty")
	ErrInvalidInput  = errors.New("invalid input")
)
