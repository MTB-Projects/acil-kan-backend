package model

import "testing"

func TestHasInstitutionalEmail(t *testing.T) {
	cases := map[string]bool{
		"kanmerkezi@saglik.gov.tr":  true,
		"ali.veli@hacettepe.edu.tr": true,
		"bagis@kizilay.org.tr":      true,
		"x@kan.kizilay.org.tr":      true,
		"x@.gov.tr":                 false,
		"Personel@ANKARA.BEL.TR":    true,
		"ali@gmail.com":             false,
		"sahte@kizilay.org.tr.co":   false,
		"x@notkizilay.org.tr":       false,
		"eksik-adres":               false,
		"":                          false,
	}
	for email, want := range cases {
		a := &InstitutionApplication{OfficialEmail: email}
		if got := a.HasInstitutionalEmail(); got != want {
			t.Errorf("HasInstitutionalEmail(%q) = %v; want %v", email, got, want)
		}
	}
}

func TestVerifiedInstitution(t *testing.T) {
	u := &User{Roles: []UserRole{RoleUser}}
	if u.IsVerifiedInstitution() {
		t.Fatal("regular user must not be verified")
	}
	u.AddRole(RoleInstitution)
	if u.IsVerifiedInstitution() {
		t.Fatal("role without institution info must not count")
	}
	u.Institution = &Institution{Name: "Ankara Şehir Hastanesi Kan Merkezi", Type: InstitutionBloodCenter}
	if !u.IsVerifiedInstitution() {
		t.Fatal("role + institution should be verified")
	}
	u.AddRole(RoleInstitution)
	if len(u.Roles) != 2 {
		t.Errorf("AddRole duplicated role: %v", u.Roles)
	}
}

func TestPublicViewCarriesVerifiedBadge(t *testing.T) {
	r := &BloodRequest{ID: "r", RequesterUID: "uid", VerifiedInstitution: "Hacettepe Kan Merkezi"}
	if r.ToPublic().VerifiedInstitution != "Hacettepe Kan Merkezi" {
		t.Error("public view lost the verified badge")
	}
	if r.ToContactView().VerifiedInstitution != "Hacettepe Kan Merkezi" {
		t.Error("contact view lost the verified badge")
	}
}
