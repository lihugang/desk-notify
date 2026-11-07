package auth

import (
	"testing"
)

func TestBcryptPassword(t *testing.T) {
	password := "123456"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" || hash == password {
		t.Fatalf("HashPassword should return salted bcrypt hash, got %s", hash)
	}

	if !CheckPassword(hash, password) {
		t.Fatalf("CheckPassword should return true for correct password")
	}

	if CheckPassword(hash, "wrongpassword") {
		t.Fatalf("CheckPassword should return false for incorrect password")
	}
}

func TestAdminToken(t *testing.T) {
	token, err := GenerateAdminToken()
	if err != nil {
		t.Fatalf("GenerateAdminToken failed: %v", err)
	}

	if !ValidateAdminToken(token) {
		t.Fatalf("ValidateAdminToken failed for valid token")
	}

	if ValidateAdminToken("invalid.token.here") {
		t.Fatalf("ValidateAdminToken should reject invalid token")
	}

	// Normal user token should NOT validate as admin
	userToken, err := GenerateToken("pwd")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if ValidateAdminToken(userToken) {
		t.Fatalf("User token should not be valid as admin token")
	}
}
