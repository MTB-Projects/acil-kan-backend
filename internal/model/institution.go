package model

import (
	"strings"
	"time"
)

// RoleInstitution marks a user whose institution (hospital, blood center, NGO)
// has been verified by an admin. Their blood requests carry a verified badge.
const RoleInstitution UserRole = "INSTITUTION"

// InstitutionType is the kind of organisation behind a verified account
type InstitutionType string

const (
	InstitutionHospital    InstitutionType = "HOSPITAL"
	InstitutionBloodCenter InstitutionType = "BLOOD_CENTER"
	InstitutionNGO         InstitutionType = "NGO"
)

// IsValid reports whether t is a known institution type
func (t InstitutionType) IsValid() bool {
	switch t {
	case InstitutionHospital, InstitutionBloodCenter, InstitutionNGO:
		return true
	}
	return false
}

// Institution is the verified organisation attached to a user
type Institution struct {
	Name       string          `firestore:"name" json:"name"`
	Type       InstitutionType `firestore:"type" json:"type"`
	City       string          `firestore:"city" json:"city"`
	District   string          `firestore:"district,omitempty" json:"district,omitempty"`
	VerifiedAt time.Time       `firestore:"verified_at" json:"verified_at"`
	VerifiedBy string          `firestore:"verified_by" json:"-"` // admin UID, internal
}

// ApplicationStatus is the review state of an institution application
type ApplicationStatus string

const (
	ApplicationPending  ApplicationStatus = "PENDING"
	ApplicationApproved ApplicationStatus = "APPROVED"
	ApplicationRejected ApplicationStatus = "REJECTED"
)

// IsValid reports whether s is a known application status
func (s ApplicationStatus) IsValid() bool {
	switch s {
	case ApplicationPending, ApplicationApproved, ApplicationRejected:
		return true
	}
	return false
}

// InstitutionApplication is a request by a user to be verified as an institution.
// An admin verifies it out of band (calling the institution's official number)
// before approving.
type InstitutionApplication struct {
	ID              string            `firestore:"id" json:"id"`
	UID             string            `firestore:"uid" json:"uid"`
	ApplicantName   string            `firestore:"applicant_name" json:"applicant_name"`
	ApplicantTitle  string            `firestore:"applicant_title" json:"applicant_title"`
	InstitutionName string            `firestore:"institution_name" json:"institution_name"`
	Type            InstitutionType   `firestore:"type" json:"type"`
	City            string            `firestore:"city" json:"city"`
	District        string            `firestore:"district,omitempty" json:"district,omitempty"`
	OfficialPhone   string            `firestore:"official_phone" json:"official_phone"`
	OfficialEmail   string            `firestore:"official_email,omitempty" json:"official_email,omitempty"`
	Note            string            `firestore:"note,omitempty" json:"note,omitempty"`
	Status          ApplicationStatus `firestore:"status" json:"status"`
	RejectReason    string            `firestore:"reject_reason,omitempty" json:"reject_reason,omitempty"`
	CreatedAt       time.Time         `firestore:"created_at" json:"created_at"`
	ReviewedAt      *time.Time        `firestore:"reviewed_at,omitempty" json:"reviewed_at,omitempty"`
	ReviewedBy      string            `firestore:"reviewed_by,omitempty" json:"-"`
}

// institutionalDomains are e-mail domains that only official organisations can hold.
// A match is a hint for the reviewer, not proof on its own.
var institutionalDomains = []string{".gov.tr", ".edu.tr", ".bel.tr", ".tsk.tr", "kizilay.org.tr"}

// HasInstitutionalEmail reports whether the application's e-mail uses an
// official Turkish institution domain (e.g. saglik.gov.tr, hacettepe.edu.tr).
func (a *InstitutionApplication) HasInstitutionalEmail() bool {
	at := strings.LastIndex(a.OfficialEmail, "@")
	if at < 0 {
		return false
	}
	domain := strings.ToLower(strings.TrimSpace(a.OfficialEmail[at+1:]))
	for _, d := range institutionalDomains {
		if strings.HasPrefix(d, ".") {
			// Kamu uzantısı: saglik.gov.tr, ankara.bel.tr ...
			if strings.HasSuffix(domain, d) && len(domain) > len(d) {
				return true
			}
			continue
		}
		// Belirli bir kurum: yalnızca kendisi ya da gerçek alt alan adı
		// ("kan.kizilay.org.tr" evet, "notkizilay.org.tr" hayır)
		if domain == d || strings.HasSuffix(domain, "."+d) {
			return true
		}
	}
	return false
}
