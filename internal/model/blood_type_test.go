package model

import "testing"

func TestParseBloodType(t *testing.T) {
	cases := map[string]BloodType{
		"A+":     BloodTypeAPositive,
		"a+":     BloodTypeAPositive,
		" A Rh+": BloodTypeAPositive,
		"0 Rh-":  BloodTypeONegative,
		"0+":     BloodTypeOPositive,
		"AB Rh-": BloodTypeABNegative,
		"ab+":    BloodTypeABPositive,
		"B Rh+":  BloodTypeBPositive,
	}
	for in, want := range cases {
		got, ok := ParseBloodType(in)
		if !ok || got != want {
			t.Errorf("ParseBloodType(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}

	for _, in := range []string{"", "C+", "A", "Rh+", "AB", "A++"} {
		if got, ok := ParseBloodType(in); ok {
			t.Errorf("ParseBloodType(%q) = %q; want invalid", in, got)
		}
	}
}

func TestCompatibleDonorTypes(t *testing.T) {
	cases := map[BloodType][]BloodType{
		BloodTypeONegative:  {BloodTypeONegative},
		BloodTypeAPositive:  {BloodTypeAPositive, BloodTypeANegative, BloodTypeOPositive, BloodTypeONegative},
		BloodTypeABPositive: AllBloodTypes,
		BloodTypeBNegative:  {BloodTypeBNegative, BloodTypeONegative},
	}
	for recipient, want := range cases {
		got := CompatibleDonorTypes(recipient)
		if !sameSet(got, want) {
			t.Errorf("CompatibleDonorTypes(%s) = %v; want %v", recipient, got, want)
		}
	}
}

func TestUserIsCompatibleDonor(t *testing.T) {
	u := &User{BloodType: BloodTypeONegative}
	if !u.IsCompatibleDonor(BloodTypeABPositive) {
		t.Error("O- should donate to AB+")
	}
	u.BloodType = BloodTypeABPositive
	if u.IsCompatibleDonor(BloodTypeONegative) {
		t.Error("AB+ should not donate to O-")
	}
}

func TestToPublicOmitsPersonalData(t *testing.T) {
	r := &BloodRequest{
		ID: "r1", RequesterUID: "uid", PatientName: "Hasta", ContactPhone: "5551112233",
		HospitalAddress: "adres", BloodType: BloodTypeAPositive, NotifiedUserUIDs: []string{"x"},
	}
	p := r.ToPublic()
	if p.ID != "r1" || p.BloodType != BloodTypeAPositive {
		t.Fatalf("unexpected public view: %+v", p)
	}
	c := r.ToContactView()
	if c.ContactPhone != "5551112233" || c.PatientName != "Hasta" {
		t.Fatalf("contact view missing contact data: %+v", c)
	}
}

func sameSet(a, b []BloodType) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[BloodType]bool{}
	for _, x := range a {
		seen[x] = true
	}
	for _, x := range b {
		if !seen[x] {
			return false
		}
	}
	return true
}
