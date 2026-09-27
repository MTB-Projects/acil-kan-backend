package service

import (
	"testing"

	"acilkan.backend/internal/model"
	appErrors "acilkan.backend/pkg/errors"
)

func TestIsValidPhone(t *testing.T) {
	for _, p := range []string{"5551234567", "0555 123 45 67", "+90 (555) 123-45-67", "+905551234567"} {
		if !IsValidPhone(p) {
			t.Errorf("IsValidPhone(%q) = false; want true", p)
		}
	}
	for _, p := range []string{"", "12345", "555abc4567", "5+551234567", "+90555123456789"} {
		if IsValidPhone(p) {
			t.Errorf("IsValidPhone(%q) = true; want false", p)
		}
	}
}

func validInput() *CreateRequestInput {
	return &CreateRequestInput{
		PatientName:  "Hasta Adı",
		BloodType:    "A Rh+",
		City:         "İstanbul",
		HospitalName: "Hastane",
		ContactPhone: "0555 123 45 67",
		UnitsNeeded:  2,
	}
}

func TestCreateRequestInputValidate(t *testing.T) {
	in := validInput()
	if err := in.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	if in.parsedBloodType != model.BloodTypeAPositive {
		t.Errorf("blood type not normalized: %q", in.parsedBloodType)
	}
	if in.ProductType != string(model.ProductWholeBlood) || in.Urgency != string(model.UrgencyNormal) {
		t.Errorf("defaults not applied: %q %q", in.ProductType, in.Urgency)
	}

	cases := map[string]func(*CreateRequestInput){
		appErrors.ErrCodeInvalidBloodType:     func(i *CreateRequestInput) { i.BloodType = "C+" },
		appErrors.ErrCodeMissingRequiredField: func(i *CreateRequestInput) { i.PatientName = "  " },
		appErrors.ErrCodeInvalidPhoneNumber:   func(i *CreateRequestInput) { i.ContactPhone = "123" },
		appErrors.ErrCodeInvalidRequestData:   func(i *CreateRequestInput) { i.Urgency = "PANIC" },
	}
	for wantCode, mutate := range cases {
		in := validInput()
		mutate(in)
		err := in.Validate()
		if err == nil || appErrors.GetAppError(err).Code != wantCode {
			t.Errorf("want %s, got %v", wantCode, err)
		}
	}
}
