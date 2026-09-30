package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestUserModel_JSONSerialization(t *testing.T) {
	now := time.Now()
	user := User{
		ID:                   "usr_123",
		FullName:             "Emerson Torres",
		Email:                "emerson@bikko.com.br",
		Phone:                "(11) 99999-9999",
		CPF:                  "123.456.789-00",
		PasswordHash:         "super_secret_hash_not_exposed",
		ProfilePhotoURL:      "https://example.com/photo.jpg",
		Rating:               4.9,
		TotalRatings:         18,
		IsBikker:             true,
		NotificationsEnabled: true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("failed to marshal User: %v", err)
	}

	jsonStr := string(data)

	// PasswordHash must be omitted from JSON
	if strings.Contains(jsonStr, "super_secret_hash_not_exposed") || strings.Contains(jsonStr, "password_hash") {
		t.Errorf("PasswordHash must not be exposed in JSON, got: %s", jsonStr)
	}

	if !strings.Contains(jsonStr, `"id":"usr_123"`) {
		t.Errorf("expected id in JSON, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"full_name":"Emerson Torres"`) {
		t.Errorf("expected full_name in JSON, got: %s", jsonStr)
	}

	var unmarshaled User
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal User: %v", err)
	}

	if unmarshaled.ID != user.ID || unmarshaled.Email != user.Email || unmarshaled.Rating != user.Rating {
		t.Errorf("unmarshaled user mismatch: %+v", unmarshaled)
	}
	if unmarshaled.PasswordHash != "" {
		t.Errorf("unmarshaled PasswordHash should be empty, got: %s", unmarshaled.PasswordHash)
	}
}

func TestBikkerModel_JSONSerialization(t *testing.T) {
	bikker := Bikker{
		ID:              "bikker_456",
		Profession:      "Eletricista",
		ExperienceYears: "5",
		Description:     "Especialista em instalações residenciais",
		LocationName:    "São Paulo, SP",
		Latitude:        -23.5505,
		Longitude:       -46.6333,
		Rating:          4.8,
		TotalReviews:    42,
		IsActive:        true,
	}

	data, err := json.Marshal(bikker)
	if err != nil {
		t.Fatalf("failed to marshal Bikker: %v", err)
	}

	var unmarshaled Bikker
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal Bikker: %v", err)
	}

	if unmarshaled.ID != "bikker_456" || unmarshaled.Profession != "Eletricista" {
		t.Errorf("unmarshaled bikker mismatch: %+v", unmarshaled)
	}
	if unmarshaled.Latitude != -23.5505 || unmarshaled.Longitude != -46.6333 {
		t.Errorf("unmarshaled coordinates mismatch: lat=%f, lon=%f", unmarshaled.Latitude, unmarshaled.Longitude)
	}
}
