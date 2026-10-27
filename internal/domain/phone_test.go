package domain

import "testing"

func TestNormalizePhone(t *testing.T) {
	for _, raw := range []string{"11999990000", "+55 (11) 99999-0000", "5511999990000"} {
		phone, err := NormalizePhone(raw)
		if err != nil || phone != "11999990000" {
			t.Fatal(phone, err)
		}
	}
	for _, raw := range []string{"999990000", "0011999990000", "+1 11999990000", "xx11999990000", "", "1199999000099"} {
		if _, err := NormalizePhone(raw); err == nil {
			t.Fatalf("accepted ambiguous phone %q", raw)
		}
	}
}
