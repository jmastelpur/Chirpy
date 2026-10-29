package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

/* Make sure that you can create and validate JWTs, and that expired tokens are rejected and JWTs signed with the wrong secret are rejected */

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "secret"
	expiresIn := time.Hour

	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to make JWT: %v", err)
	}
	if token == "" {
		t.Fatalf("Expected JWT token, got empty string")
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "secret"
	expiresIn := time.Hour

	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to make JWT: %v", err)
	}

	parsedUserID, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}
	if parsedUserID != userID {
		t.Fatalf("Expected user ID %v, got %v", userID, parsedUserID)
	}
}

func TestValidateExpiredJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "secret"
	expiresIn := -time.Hour // Token already expired

	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to make JWT: %v", err)
	}

	_, err = ValidateJWT(token, tokenSecret)
	if err == nil {
		t.Fatalf("Expected error for expired JWT, got nil")
	}
}

func TestValidateWrongSecretJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "secret"
	wrongSecret := "wrong"
	expiresIn := time.Hour

	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to make JWT: %v", err)
	}

	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Fatalf("Expected error for JWT signed with wrong secret, got nil")
	}
}

func TestGetBearerToken(t *testing.T) {
	headers := http.Header{}
	_, err := GetBearerToken(headers)
	if err == nil {
		t.Fatalf("Expected error for missing authorization header, got nil")
	}

	headers.Set("Authorization", "Bearer mytoken")
	token, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("Failed to get bearer token: %v", err)
	}
	if token != "mytoken" {
		t.Fatalf("Expected bearer token 'mytoken', got '%s'", token)
	}
}
