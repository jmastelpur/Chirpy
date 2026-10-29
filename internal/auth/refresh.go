package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func MakeRefreshToken() string {
	/* generate a random 256-bit (32-byte) hex-encoded string:
	rand.Read to generate 32 bytes (256 bits) of random data from the crypto/rand package (math/rand's Read function is deprecated).
	hex.EncodeToString to convert the random data to a hex string */
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	return hex.EncodeToString(tokenBytes)
}
