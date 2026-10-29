package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {

	/* It should extract the api key from the Authorization header, which is expected to be in this format:

	Authorization: ApiKey THE_KEY_HERE

	You'll need to strip out the ApiKey part and the whitespace and return just the key. */
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "ApiKey" {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	return parts[1], nil
}
