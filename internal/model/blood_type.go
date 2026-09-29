package model

import "strings"

// AllBloodTypes lists every valid blood type in canonical API format ("A+", "O-", ...)
var AllBloodTypes = []BloodType{
	BloodTypeAPositive, BloodTypeANegative,
	BloodTypeBPositive, BloodTypeBNegative,
	BloodTypeABPositive, BloodTypeABNegative,
	BloodTypeOPositive, BloodTypeONegative,
}

// donorCompatibility maps a donor blood type to the recipient types it can donate to
var donorCompatibility = map[BloodType][]BloodType{
	BloodTypeONegative:  {BloodTypeONegative, BloodTypeOPositive, BloodTypeANegative, BloodTypeAPositive, BloodTypeBNegative, BloodTypeBPositive, BloodTypeABNegative, BloodTypeABPositive},
	BloodTypeOPositive:  {BloodTypeOPositive, BloodTypeAPositive, BloodTypeBPositive, BloodTypeABPositive},
	BloodTypeANegative:  {BloodTypeANegative, BloodTypeAPositive, BloodTypeABNegative, BloodTypeABPositive},
	BloodTypeAPositive:  {BloodTypeAPositive, BloodTypeABPositive},
	BloodTypeBNegative:  {BloodTypeBNegative, BloodTypeBPositive, BloodTypeABNegative, BloodTypeABPositive},
	BloodTypeBPositive:  {BloodTypeBPositive, BloodTypeABPositive},
	BloodTypeABNegative: {BloodTypeABNegative, BloodTypeABPositive},
	BloodTypeABPositive: {BloodTypeABPositive},
}

// IsValid reports whether b is one of the canonical blood types
func (b BloodType) IsValid() bool {
	_, ok := donorCompatibility[b]
	return ok
}

// CanDonateTo reports whether a donor with type b can donate to recipient
func (b BloodType) CanDonateTo(recipient BloodType) bool {
	for _, t := range donorCompatibility[b] {
		if t == recipient {
			return true
		}
	}
	return false
}

// CompatibleDonorTypes returns every donor blood type that can donate to recipient
func CompatibleDonorTypes(recipient BloodType) []BloodType {
	var donors []BloodType
	for _, donor := range AllBloodTypes {
		if donor.CanDonateTo(recipient) {
			donors = append(donors, donor)
		}
	}
	return donors
}

// ParseBloodType normalizes user input into a canonical blood type.
// Accepts "A+", "a +", "A Rh+", "0 Rh-", "ARh+" etc. Returns false if unrecognized.
func ParseBloodType(s string) (BloodType, bool) {
	v := strings.ToUpper(strings.TrimSpace(s))
	v = strings.ReplaceAll(v, " ", "")
	v = strings.ReplaceAll(v, "RH", "")
	// Turkish usage writes group O as zero
	v = strings.ReplaceAll(v, "0", "O")
	bt := BloodType(v)
	if !bt.IsValid() {
		return "", false
	}
	return bt, true
}
