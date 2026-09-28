package textnorm

import (
	"reflect"
	"testing"
)

func TestFold(t *testing.T) {
	cases := map[string]string{
		"İstanbul":  "istanbul",
		"ISTANBUL":  "istanbul",
		"Şehir":     "sehir",
		"Kâğıthane": "kagithane",
		"Hakkâri":   "hakkari",
		"GÜNGÖREN":  "gungoren",
	}
	for in, want := range cases {
		if got := Fold(in); got != want {
			t.Errorf("Fold(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestWords(t *testing.T) {
	got := Words("Ankara Şehir Hastanesi (MHH)")
	want := []string{"ankara", "sehir", "hastanesi", "mhh"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Words = %v; want %v", got, want)
	}
}

func TestSlug(t *testing.T) {
	if got := Slug("Gümüşhane  (Mrk)"); got != "gumushane_mrk" {
		t.Errorf("Slug = %q", got)
	}
}
