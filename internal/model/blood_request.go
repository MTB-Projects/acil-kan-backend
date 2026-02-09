package model

import "time"

// RequestStatus represents the status of a blood request
type RequestStatus string

const (
	RequestStatusActive    RequestStatus = "ACTIVE"
	RequestStatusFulfilled RequestStatus = "FULFILLED"
	RequestStatusExpired   RequestStatus = "EXPIRED"
	RequestStatusCancelled RequestStatus = "CANCELLED"
)

// BloodRequest represents an emergency blood donation request
type BloodRequest struct {
	ID               string        `firestore:"id" json:"id"`
	RequesterUID     string        `firestore:"requester_uid" json:"requester_uid"`
	RequesterName    string        `firestore:"requester_name" json:"requester_name"`
	PatientName      string        `firestore:"patient_name" json:"patient_name"`
	BloodType        BloodType     `firestore:"blood_type" json:"blood_type"`
	City             string        `firestore:"city" json:"city"`
	District         string        `firestore:"district,omitempty" json:"district,omitempty"` // Optional district for more specific targeting
	HospitalName     string        `firestore:"hospital_name" json:"hospital_name"`
	HospitalAddress  string        `firestore:"hospital_address" json:"hospital_address"`
	ContactPhone     string        `firestore:"contact_phone" json:"contact_phone"`
	UnitsNeeded      int           `firestore:"units_needed" json:"units_needed"`
	Description      string        `firestore:"description" json:"description"`
	Status           RequestStatus `firestore:"status" json:"status"`
	ExpiresAt        time.Time     `firestore:"expires_at" json:"expires_at"`
	CreatedAt        time.Time     `firestore:"created_at" json:"created_at"`
	UpdatedAt        time.Time     `firestore:"updated_at" json:"updated_at"`
	NotifiedUserUIDs []string      `firestore:"notified_user_uids,omitempty" json:"notified_user_uids,omitempty"` // Track who was notified
}

// IsExpired checks if the request has expired
func (r *BloodRequest) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsActive checks if the request is still active
func (r *BloodRequest) IsActive() bool {
	return r.Status == RequestStatusActive && !r.IsExpired()
}

// CanBeCancelledBy checks if a user can cancel this request
func (r *BloodRequest) CanBeCancelledBy(userUID string) bool {
	return r.RequesterUID == userUID && r.Status == RequestStatusActive
}

// CanBeUpdatedBy checks if a user can update this request
func (r *BloodRequest) CanBeUpdatedBy(userUID string) bool {
	return r.RequesterUID == userUID && r.Status == RequestStatusActive && !r.IsExpired()
}
