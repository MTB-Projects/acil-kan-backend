package notification

import (
	"regexp"
	"testing"

	"acilkan.backend/internal/model"
)

var validTopic = regexp.MustCompile(`^[a-zA-Z0-9-_.~%]+$`)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"İstanbul":         "istanbul",
		"istanbul":         "istanbul",
		"Kadıköy":          "kadikoy",
		"Şanlıurfa":        "sanliurfa",
		"Çanakkale":        "canakkale",
		"Muğla":            "mugla",
		"Afyonkarahisar ":  "afyonkarahisar",
		"İstanbul Avrupa":  "istanbul_avrupa",
		"Gümüşhane  (Mrk)": "gumushane_mrk",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestTopicNamesAreValidFCMTopics(t *testing.T) {
	for _, bt := range model.AllBloodTypes {
		for _, city := range []string{"İstanbul", "Şırnak", "Kırklareli", "Iğdır"} {
			name := GenerateTopicName(bt, city)
			if !validTopic.MatchString(name) {
				t.Errorf("invalid topic name %q", name)
			}
		}
	}
	if got := GenerateTopicName(model.BloodTypeAPositive, "İstanbul"); got != "blood_apos_istanbul" {
		t.Errorf("GenerateTopicName = %q", got)
	}
	if got := GenerateTopicName(model.BloodTypeABNegative, "Ankara"); got != "blood_abneg_ankara" {
		t.Errorf("GenerateTopicName = %q", got)
	}
}

func TestRecipientTopicsCoverCompatibleDonors(t *testing.T) {
	r := &model.BloodRequest{BloodType: model.BloodTypeAPositive, City: "İzmir", District: "Bornova"}
	got := RecipientTopics(r)
	want := []string{"blood_apos_izmir", "blood_aneg_izmir", "blood_opos_izmir", "blood_oneg_izmir"}
	if len(got) != len(want) {
		t.Fatalf("RecipientTopics = %v; want %v", got, want)
	}
	set := map[string]bool{}
	for _, s := range got {
		set[s] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("missing topic %q in %v", w, got)
		}
	}
}
