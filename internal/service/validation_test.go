package service

import (
	"testing"
	"time"

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

func TestParseDonationDate(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

	for _, clear := range []interface{}{nil, "", "  "} {
		if d, err := ParseDonationDate(clear, now); err != nil || d != nil {
			t.Errorf("ParseDonationDate(%q) = %v, %v; want nil, nil", clear, d, err)
		}
	}

	d, err := ParseDonationDate("2026-06-01", now)
	if err != nil || d == nil || d.Format("2006-01-02") != "2026-06-01" {
		t.Errorf("date-only: got %v, %v", d, err)
	}
	if _, err := ParseDonationDate("2026-06-01T10:00:00Z", now); err != nil {
		t.Errorf("RFC3339: %v", err)
	}

	for _, bad := range []interface{}{"2026-10-01", "1900-01-01", "01.06.2026", 42} {
		if _, err := ParseDonationDate(bad, now); err == nil {
			t.Errorf("ParseDonationDate(%v) accepted; want error", bad)
		}
	}
}
