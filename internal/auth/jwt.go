package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	/* Use jwt.NewWithClaims
	Use jwt.SigningMethodHS256 as the signing method.
	Use jwt.RegisteredClaims as the claims.
	Set the Issuer to "chirpy-access"
	Set IssuedAt to the current time in UTC
	Set ExpiresAt to the current time plus the expiration time (expiresIn)
	Set the Subject to a stringified version of the user's id
	*/
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn).UTC()),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	/* Use the jwt.ParseWithClaims function to validate the signature of the JWT and extract the claims into a *jwt.Token struct.
	The keyFunc callback must return the same key type ([]byte) used when the token was signed. An error will be returned if the token is invalid or has expired.
	If all is well with the token, use the token.Claims interface to get access to the user's id from the claims (which should be stored in the Subject field).
	Return the id as a uuid.UUID.
	*/
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	/* This function should look for the Authorization header in the headers parameter and return the TOKEN_STRING if it exists (stripping off the Bearer prefix and whitespace).
	If the header doesn't exist, return an error. */
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}
	return parts[1], nil
}
