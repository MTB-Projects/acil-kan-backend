package model

import "time"

// BloodType represents valid blood types
type BloodType string

const (
	BloodTypeAPositive  BloodType = "A+"
	BloodTypeANegative  BloodType = "A-"
	BloodTypeBPositive  BloodType = "B+"
	BloodTypeBNegative  BloodType = "B-"
	BloodTypeABPositive BloodType = "AB+"
	BloodTypeABNegative BloodType = "AB-"
	BloodTypeOPositive  BloodType = "O+"
	BloodTypeONegative  BloodType = "O-"
)

// UserRole represents user roles in the system
type UserRole string

const (
	RoleUser  UserRole = "USER"
	RoleAdmin UserRole = "ADMIN"
)

// User represents a user in the system
// Firebase Auth stores only UID and authentication data
// All domain-specific data is stored here in backend database
type User struct {
	UID              string     `firestore:"uid" json:"uid"` // Firebase UID
	Email            string     `firestore:"email" json:"email"`
	PhoneNumber      string     `firestore:"phone_number" json:"phone_number"`
	FullName         string     `firestore:"full_name" json:"full_name"`
	BloodType        BloodType  `firestore:"blood_type" json:"blood_type"`
	City             string     `firestore:"city" json:"city"`
	District         string     `firestore:"district" json:"district"`
	IsDonor          bool       `firestore:"is_donor" json:"is_donor"` // Whether user is willing to donate
	LastDonationDate *time.Time `firestore:"last_donation_date,omitempty" json:"last_donation_date,omitempty"`
	FCMToken         string     `firestore:"fcm_token,omitempty" json:"fcm_token,omitempty"` // For push notifications
	Roles            []UserRole `firestore:"roles" json:"roles"`
	CreatedAt        time.Time  `firestore:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `firestore:"updated_at" json:"updated_at"`
}

// CanDonate checks if user can donate blood (at least 3 months since last donation)
func (u *User) CanDonate() bool {
	if !u.IsDonor {
		return false
	}
	if u.LastDonationDate == nil {
		return true
	}
	// Blood donation is allowed every 3 months (90 days)
	return time.Since(*u.LastDonationDate) >= 90*24*time.Hour
}

// HasRole checks if user has a specific role
func (u *User) HasRole(role UserRole) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsCompatibleDonor checks if this user can donate to a specific blood type
func (u *User) IsCompatibleDonor(recipientBloodType BloodType) bool {
	compatibility := map[BloodType][]BloodType{
		BloodTypeONegative:  {BloodTypeONegative, BloodTypeOPositive, BloodTypeANegative, BloodTypeAPositive, BloodTypeBNegative, BloodTypeBPositive, BloodTypeABNegative, BloodTypeABPositive},
		BloodTypeOPositive:  {BloodTypeOPositive, BloodTypeAPositive, BloodTypeBPositive, BloodTypeABPositive},
		BloodTypeANegative:  {BloodTypeANegative, BloodTypeAPositive, BloodTypeABNegative, BloodTypeABPositive},
		BloodTypeAPositive:  {BloodTypeAPositive, BloodTypeABPositive},
		BloodTypeBNegative:  {BloodTypeBNegative, BloodTypeBPositive, BloodTypeABNegative, BloodTypeABPositive},
		BloodTypeBPositive:  {BloodTypeBPositive, BloodTypeABPositive},
		BloodTypeABNegative: {BloodTypeABNegative, BloodTypeABPositive},
		BloodTypeABPositive: {BloodTypeABPositive},
	}

	compatibleTypes, exists := compatibility[u.BloodType]
	if !exists {
		return false
	}

	for _, compatible := range compatibleTypes {
		if compatible == recipientBloodType {
			return true
		}
	}
	return false
}
