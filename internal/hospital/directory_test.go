package hospital

import "testing"

const sample = `[
 {"id":"w1","name":"Ankara Şehir Hastanesi","city":"Ankara","district":"Çankaya"},
 {"id":"w2","name":"Gazi Üniversitesi Hastanesi","city":"Ankara","district":"Yenimahalle"},
 {"id":"w3","name":"Hacettepe Üniversitesi Hastanesi","city":"Ankara","district":"Altındağ"},
 {"id":"w4","name":"Özel Çankaya Hastanesi","city":"Ankara","district":"Çankaya"},
 {"id":"w5","name":"İzmir Şehir Hastanesi","city":"İzmir","district":"Bayraklı"}
]`

func names(hs []Hospital) []string {
	var out []string
	for _, h := range hs {
		out = append(out, h.Name)
	}
	return out
}

func TestSearch(t *testing.T) {
	d, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		city, district, q string
		want              []string
	}{
		// Türkçe karakter duyarsız, kelime başı eşleşmesi
		{"Ankara", "", "sehir", []string{"Ankara Şehir Hastanesi"}},
		{"ankara", "", "GAZİ", []string{"Gazi Üniversitesi Hastanesi"}},
		// Adı sorguyla başlayanlar önce
		{"Ankara", "", "hac", []string{"Hacettepe Üniversitesi Hastanesi"}},
		// Birden çok kelime: hepsi eşleşmeli
		{"Ankara", "", "univ gaz", []string{"Gazi Üniversitesi Hastanesi"}},
		// İlçe filtresi, boş sorgu: alfabetik liste
		{"Ankara", "Çankaya", "", []string{"Ankara Şehir Hastanesi", "Özel Çankaya Hastanesi"}},
		// Şehir filtresi
		{"İzmir", "", "sehir", []string{"İzmir Şehir Hastanesi"}},
		// Kelimenin ortası eşleşmez
		{"Ankara", "", "ehir", nil},
	}
	for _, c := range cases {
		got := names(d.Search(c.city, c.district, c.q, 10))
		if len(got) != len(c.want) {
			t.Errorf("Search(%q,%q,%q) = %v; want %v", c.city, c.district, c.q, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("Search(%q,%q,%q) = %v; want %v", c.city, c.district, c.q, got, c.want)
				break
			}
		}
	}
}

func TestSearchRanksNamePrefixFirst(t *testing.T) {
	d, _ := Parse([]byte(sample))
	// "cankaya": "Özel Çankaya Hastanesi" (ikinci kelime) ile "Ankara Şehir" (ilçe Çankaya) eşleşir;
	// hiçbirinin adı "cankaya" ile başlamaz, ilk kelime eşleşmesi de yok -> alfabetik
	got := names(d.Search("Ankara", "", "cankaya", 10))
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	got = names(d.Search("Ankara", "", "ozel", 10))
	if len(got) != 1 || got[0] != "Özel Çankaya Hastanesi" {
		t.Errorf("got %v", got)
	}
}

func TestSearchLimit(t *testing.T) {
	d, _ := Parse([]byte(sample))
	if got := d.Search("Ankara", "", "", 2); len(got) != 2 {
		t.Errorf("limit: got %d", len(got))
	}
}

func TestEmbeddedDataLoads(t *testing.T) {
	d, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if d.Len() < 1000 {
		t.Errorf("embedded directory has only %d hospitals", d.Len())
	}
}
