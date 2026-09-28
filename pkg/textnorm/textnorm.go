// Package textnorm normalizes Turkish text for matching: "İstanbul", "istanbul"
// and "ISTANBUL" all become "istanbul"; "Şehir" becomes "sehir".
package textnorm

import "strings"

var turkishReplacer = strings.NewReplacer(
	"ç", "c", "Ç", "c",
	"ğ", "g", "Ğ", "g",
	"ı", "i", "I", "i", "İ", "i",
	"ö", "o", "Ö", "o",
	"ş", "s", "Ş", "s",
	"ü", "u", "Ü", "u",
	"â", "a", "Â", "a",
	"î", "i", "Î", "i",
	"û", "u", "Û", "u",
)

// Fold lowercases s and maps Turkish letters to ASCII. Other characters are kept.
func Fold(s string) string {
	return strings.ToLower(turkishReplacer.Replace(s))
}

// Words folds s and splits it into runs of ASCII letters and digits.
func Words(s string) []string {
	return strings.FieldsFunc(Fold(s), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
}

// Slug folds s into a lowercase ASCII token joined by underscores:
// "İstanbul Avrupa" -> "istanbul_avrupa".
func Slug(s string) string {
	return strings.Join(Words(s), "_")
}
