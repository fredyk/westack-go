package model

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"math/rand"
	"strings"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp/totp"
)

type EnableMfaBody struct {
	VerificationCode string `json:"verificationCode"`
}

type EnableMfaResponse struct {
	SecretKey   string `json:"secretKey"`
	BackupCodes string `json:"backupCodes"`
	Enabled     bool   `json:"enabled"`
	QrCode      string `json:"qrCode"`
}

// This method is a 2-step operation.
// The first steps setups MFA for the user and returns a secret key.
// Required attributes 1st time:
// - accountId
// Required attributes 2nd time:
// - accountId
// - mfaBody.VerificationCode
// The second step enables MFA for the user, and requires the secret key and a verification code.
func EnableMfa(mfaModel *StatefulModel, ctx *EventContext, mfaBody EnableMfaBody) (EnableMfaResponse, error) {

	systemContext := &EventContext{
		Bearer: &BearerToken{Account: &BearerAccount{System: true}},
	}
	accountId := ctx.ModelID

	account, err := ctx.Model.FindById(accountId, nil, systemContext)
	if err != nil {
		return EnableMfaResponse{}, err
	}

	if account == nil {
		return EnableMfaResponse{}, wst.CreateError(fiber.ErrUnauthorized, "ERR_UNAUTHORIZED", fiber.Map{"message": "Account not found"}, "Error")
	}

	username := account.GetString("username")
	if username == "" {
		username = account.GetString("email")
	}

	isStep1 := true
	// find an existing mfa record with status "SETUP"
	mfaRecord, err := mfaModel.FindOne(&wst.Filter{
		Where: &wst.Where{
			"accountId": accountId,
		},
		Include: &wst.Include{
			{Relation: "account"},
		},
	}, systemContext)
	if err != nil {
		return EnableMfaResponse{}, err
	}

	if mfaRecord != nil && mfaRecord.GetString("status") == "ENABLED" {
		return EnableMfaResponse{}, wst.CreateError(fiber.ErrUnauthorized, "ERR_UNAUTHORIZED", fiber.Map{"message": "MFA already enabled"}, "Error")
	}

	if mfaRecord != nil && mfaBody.VerificationCode != "" {
		isStep1 = false
	}

	if isStep1 {
		// Step 1: Setup MFA
		// Generate a secret key
		secretKey, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "WeStack",
			AccountName: username,
		})
		if err != nil {
			return EnableMfaResponse{}, err
		}

		if mfaRecord != nil {
			// Delete all existing MFA records
			deleteResult, err := mfaModel.DeleteMany(&wst.Where{"accountId": accountId}, systemContext)
			if err != nil {
				return EnableMfaResponse{}, err
			}
			fmt.Printf("[INFO] Deleted %v MFA records\n", deleteResult.DeletedCount)
		}

		// backup codes
		backupCodes := generateBackupCodes()

		// Create a new MFA record
		secretKeyString := secretKey.Secret()
		_, err = mfaModel.Create(&wst.M{
			"accountId":   accountId,
			"status":      "SETUP",
			"secretKey":   secretKeyString,
			"backupCodes": generateBackupCodes(),
		}, systemContext)
		if err != nil {
			return EnableMfaResponse{}, err
		}

		// Convert TOTP key into a QR code encoded as a PNG image.
		var buf bytes.Buffer
		img, err := secretKey.Image(256, 256)
		if err != nil {
			return EnableMfaResponse{}, err
		}
		png.Encode(&buf, img)

		base64Image := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

		// Return the secret key
		return EnableMfaResponse{
			SecretKey:   secretKeyString,
			Enabled:     false,
			BackupCodes: backupCodes,
			QrCode:      base64Image,
		}, nil
	} else {
		// Step 2: Enable MFA

		// Check if the secret key matches
		ok := totp.Validate(mfaBody.VerificationCode, mfaRecord.GetString("secretKey"))
		if !ok {
			return EnableMfaResponse{}, wst.CreateError(fiber.ErrUnauthorized, "ERR_UNAUTHORIZED", fiber.Map{"message": "Invalid verification code"}, "Error")
		}

		// Update the MFA record
		_, err = mfaRecord.UpdateAttributes(wst.M{
			"status": "ENABLED",
		}, systemContext)
		if err != nil {
			return EnableMfaResponse{}, err
		}

		fmt.Printf("[INFO] MFA enabled for account %v\n", accountId)

		// Return success
		return EnableMfaResponse{
			Enabled: true,
		}, nil
	}

}

func HasMfaEnabled(mfaModel *StatefulModel, accountId any) (bool, error) {
	systemContext := &EventContext{
		Bearer: &BearerToken{Account: &BearerAccount{System: true}},
	}
	mfaRecord, err := mfaModel.FindOne(&wst.Filter{
		Where: &wst.Where{
			"accountId": accountId,
		},
	}, systemContext)
	if err != nil {
		return false, err
	}

	if mfaRecord == nil {
		return false, nil
	}

	return mfaRecord.GetString("status") == "ENABLED", nil
}

func LoginVerifyMfa(mfaModel *StatefulModel, accountId any, verificationCode string) (bool, error) {
	systemContext := &EventContext{
		Bearer: &BearerToken{Account: &BearerAccount{System: true}},
	}
	// Find the MFA record
	mfaRecord, err := mfaModel.FindOne(&wst.Filter{
		Where: &wst.Where{
			"accountId": accountId,
		},
	}, systemContext)
	if err != nil {
		return false, err
	}

	if mfaRecord == nil {
		return false, wst.CreateError(fiber.ErrUnauthorized, "ERR_UNAUTHORIZED", fiber.Map{"message": "MFA not enabled"}, "Error")
	}

	// Check if the verification code matches
	ok := totp.Validate(verificationCode, mfaRecord.GetString("secretKey"))
	if !ok {
		return false, nil
	}

	return true, nil
}

// Composed of 80 digits
func generateBackupCodes() string {
	rows := 8
	cols := 10
	backupCodes := make([]string, rows)
	for i := 0; i < rows; i++ {
		backupCodes[i] = ""
		for j := 0; j < cols; j++ {
			n := 10000000 + rand.Intn(89999999)
			backupCodes[i] += fmt.Sprintf("%d", n)
		}
	}
	return strings.Join(backupCodes, "")
}
