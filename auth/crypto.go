package auth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
)

// verifyRS256 verifies an RS256 JWT signature.
func verifyRS256(signingInput, signatureB64 string, key *rsa.PublicKey) error {
	signature, err := base64URLDecode(signatureB64)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	hash := sha256.Sum256([]byte(signingInput))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], signature)
}
