package service

import (
	"errors"
	"strings"
	"time"
)

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

// ParseDonationDate reads the optional last donation date from a profile update.
// Accepts null or "" (clears it), "2006-01-02" or RFC 3339. The date may not be
// in the future or more than 100 years ago.
func ParseDonationDate(raw interface{}, now time.Time) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	s, ok := raw.(string)
	if !ok {
		return nil, errors.New("last_donation_date must be a date string")
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return nil, errors.New("last_donation_date must be YYYY-MM-DD")
	}
	if t.After(now) {
		return nil, errors.New("last_donation_date cannot be in the future")
	}
	if t.Before(now.AddDate(-100, 0, 0)) {
		return nil, errors.New("last_donation_date is too old")
	}
	return &t, nil
}
