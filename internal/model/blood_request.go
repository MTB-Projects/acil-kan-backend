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

// ProductType represents the requested blood product
type ProductType string

const (
	ProductWholeBlood  ProductType = "WHOLE_BLOOD"
	ProductErythrocyte ProductType = "ERYTHROCYTE"
	ProductThrombocyte ProductType = "THROMBOCYTE"
	ProductPlasma      ProductType = "PLASMA"
	ProductGranulocyte ProductType = "GRANULOCYTE"
)

// IsValid reports whether p is a known product type
func (p ProductType) IsValid() bool {
	switch p {
	case ProductWholeBlood, ProductErythrocyte, ProductThrombocyte, ProductPlasma, ProductGranulocyte:
		return true
	}
	return false
}

// Urgency represents how urgent a blood request is
type Urgency string

const (
	UrgencyNormal   Urgency = "NORMAL"
	UrgencyHigh     Urgency = "HIGH"
	UrgencyCritical Urgency = "CRITICAL"
)

// IsValid reports whether u is a known urgency level
func (u Urgency) IsValid() bool {
	switch u {
	case UrgencyNormal, UrgencyHigh, UrgencyCritical:
		return true
	}
	return false
}

// BloodRequest represents an emergency blood donation request
type BloodRequest struct {
	ID               string        `firestore:"id" json:"id"`
	RequesterUID     string        `firestore:"requester_uid" json:"requester_uid"`
	RequesterName    string        `firestore:"requester_name" json:"requester_name"`
	PatientName      string        `firestore:"patient_name" json:"patient_name"`
	BloodType        BloodType     `firestore:"blood_type" json:"blood_type"`
	ProductType      ProductType   `firestore:"product_type" json:"product_type"`
	Urgency          Urgency       `firestore:"urgency" json:"urgency"`
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
	// Name of the verified institution that posted the request; empty for regular users
	VerifiedInstitution string `firestore:"verified_institution,omitempty" json:"verified_institution,omitempty"`
}

// PublicBloodRequest is the view of a request that anyone can see without logging in.
// It deliberately omits personal data (patient name, contact phone, user IDs).
type PublicBloodRequest struct {
	ID           string      `json:"id"`
	BloodType    BloodType   `json:"blood_type"`
	ProductType  ProductType `json:"product_type"`
	Urgency      Urgency     `json:"urgency"`
	City         string      `json:"city"`
	District     string      `json:"district,omitempty"`
	HospitalName string      `json:"hospital_name"`
	UnitsNeeded  int         `json:"units_needed"`
	Description  string      `json:"description"`
	ExpiresAt    time.Time   `json:"expires_at"`
	CreatedAt    time.Time   `json:"created_at"`
	// Verified institution badge, e.g. "Ankara Şehir Hastanesi Kan Merkezi"
	VerifiedInstitution string `json:"verified_institution,omitempty"`
}

// ToPublic strips personal data from the request
func (r *BloodRequest) ToPublic() *PublicBloodRequest {
	return &PublicBloodRequest{
		ID:           r.ID,
		BloodType:    r.BloodType,
		ProductType:  r.ProductType,
		Urgency:      r.Urgency,
		City:         r.City,
		District:     r.District,
		HospitalName: r.HospitalName,
		UnitsNeeded:  r.UnitsNeeded,
		Description:  r.Description,
		ExpiresAt:    r.ExpiresAt,
		CreatedAt:    r.CreatedAt,

		VerifiedInstitution: r.VerifiedInstitution,
	}
}

// ContactView is shown to authenticated users who want to help: it adds
// the contact details needed to reach the requester, but no internal IDs.
type ContactView struct {
	*PublicBloodRequest
	PatientName     string `json:"patient_name"`
	HospitalAddress string `json:"hospital_address"`
	ContactPhone    string `json:"contact_phone"`
}

// ToContactView returns the request with contact details for logged-in users
func (r *BloodRequest) ToContactView() *ContactView {
	return &ContactView{
		PublicBloodRequest: r.ToPublic(),
		PatientName:        r.PatientName,
		HospitalAddress:    r.HospitalAddress,
		ContactPhone:       r.ContactPhone,
	}
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
