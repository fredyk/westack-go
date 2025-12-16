package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateOauthStateString(cookie string) string {
	oauthState := primitive.NewObjectID().Hex()
	hash := sha256.New()
	hash.Write([]byte(oauthState + cookie))
	return fmt.Sprintf("%s.%x", oauthState, hash.Sum(nil))
}

func VerifyOauthState(cookie, stateWithHash string) bool {
	oauthState := stateWithHash[:24]
	hash := sha256.New()
	hash.Write([]byte(oauthState + cookie))
	return stateWithHash[25:] == hex.EncodeToString(hash.Sum(nil))
}
