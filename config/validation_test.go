package config

import "testing"

func TestExplicitLocalCapabilities(t *testing.T) {
	c := &Config{Env: "development", JWTSecret: "local-test-secret-with-at-least-32-characters", LocalTestAccounts: true, LocalStorage: true}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Env = "production"
	if c.Validate() == nil {
		t.Fatal("production must reject local test capabilities")
	}
	c.Env = ""
	if c.Validate() == nil {
		t.Fatal("missing environment must not activate development")
	}
}

func TestMissingIdentityAndStorageFailClosed(t *testing.T) {
	c := &Config{Env: "production", JWTSecret: "strong-configured-secret-at-least-32-characters"}
	if c.Validate() == nil {
		t.Fatal("missing providers must fail configuration validation")
	}
	c.FirebaseProjectID = "project"
	if c.Validate() == nil {
		t.Fatal("missing R2 must not activate mock storage")
	}
}
