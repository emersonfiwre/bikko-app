package domain

import (
	"fmt"
	"strings"
)

// NormalizePhone returns a Brazilian mobile number including its area code.
// Only the optional country prefix 55 is removed; arbitrary suffix matching is forbidden.
func NormalizePhone(raw string) (string, error) {
	var digits strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' || r == ' ' || r == '(' || r == ')' || r == '-':
		default:
			return "", fmt.Errorf("invalid phone number")
		}
	}
	phone := digits.String()
	if len(phone) == 13 && strings.HasPrefix(phone, "55") {
		phone = phone[2:]
	}
	if len(phone) != 11 || phone[0] == '0' || phone[2] != '9' {
		return "", fmt.Errorf("invalid Brazilian mobile number")
	}
	return phone, nil
}
