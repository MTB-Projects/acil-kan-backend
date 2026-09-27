package service

import "strings"

// IsValidPhone accepts Turkish-style phone numbers in common formats:
// "5XX XXX XX XX", "05XX...", "+90 5XX..." (spaces, dashes and parentheses ignored).
func IsValidPhone(phone string) bool {
	digits := strings.Builder{}
	for i, r := range phone {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' && i == 0, r == ' ', r == '-', r == '(', r == ')':
		default:
			return false
		}
	}
	n := digits.Len()
	return n >= 10 && n <= 13
}
