package westack

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
)

// Helpers de gestión de ApiKeys (modelo interno ApiKey, ver setupmodels.go).
//
// La `key` es un token OPACO de alta entropía (32 bytes) que el cliente envía en el header
// X-Api-Key; se persiste tal cual y GetBearer la busca por igualdad (mismo modelo que los
// AccessToken de westack, cuyo id ES el token). `secretHash` guarda el SHA-256 de la key como
// salvaguarda adicional para auditoría/rotación; la autorización se basa en `key` + `enabled`.

// randomToken genera n bytes aleatorios en hex (2n chars).
func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ApiKeySecretHash devuelve el SHA-256 (hex) de una key. Es el valor por el que se persisten y
// buscan las ApiKeys (`secretHash`); la key en claro nunca se almacena. Útil para localizar una
// ApiKey a partir de la key en claro sin exponer el hashing interno.
func ApiKeySecretHash(key string) string {
	return sha256Hex(key)
}

// CreateApiKey crea una ApiKey con los roles dados, opcionalmente ligada a una cuenta.
// Devuelve la `key` en claro (mostrarla UNA vez; no se puede recuperar después).
func CreateApiKey(app *WeStack, name string, roles []string, accountId interface{}, ctx *model.EventContext) (key string, err error) {
	if app.apiKeyModel == nil {
		return "", fmt.Errorf("apiKeyModel not initialized")
	}
	key, err = randomToken(32)
	if err != nil {
		return "", err
	}
	// NO se persiste la key en claro: solo su SHA-256. La key en claro se devuelve UNA vez al
	// llamante (abajo) y no puede recuperarse después. La autenticación busca por `secretHash`.
	data := wst.M{
		"secretHash": sha256Hex(key),
		"name":       name,
		"roles":      roles,
		"enabled":    true,
	}
	if accountId != nil {
		data["accountId"] = accountId
	}
	if _, err = app.apiKeyModel.Create(data, ctx); err != nil {
		return "", err
	}
	return key, nil
}

// RevokeApiKey desactiva una ApiKey (enabled=false). Revocación atómica e inmediata.
func RevokeApiKey(app *WeStack, key string, ctx *model.EventContext) error {
	if app.apiKeyModel == nil {
		return fmt.Errorf("apiKeyModel not initialized")
	}
	inst, err := app.apiKeyModel.FindOne(&wst.Filter{Where: &wst.Where{"secretHash": sha256Hex(key)}}, ctx)
	if err != nil {
		return err
	}
	if inst == nil {
		return fmt.Errorf("api key not found")
	}
	_, err = inst.UpdateAttributes(wst.M{"enabled": false}, ctx)
	return err
}

// SetApiKeyRoles reemplaza los roles/permisos atómicos de una ApiKey.
func SetApiKeyRoles(app *WeStack, key string, roles []string, ctx *model.EventContext) error {
	if app.apiKeyModel == nil {
		return fmt.Errorf("apiKeyModel not initialized")
	}
	inst, err := app.apiKeyModel.FindOne(&wst.Filter{Where: &wst.Where{"secretHash": sha256Hex(key)}}, ctx)
	if err != nil {
		return err
	}
	if inst == nil {
		return fmt.Errorf("api key not found")
	}
	_, err = inst.UpdateAttributes(wst.M{"roles": roles}, ctx)
	return err
}
