package westack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost es el coste de trabajo de bcrypt para hashear contraseñas.
const bcryptCost = 11

// pepperedPassword deriva una entrada de LONGITUD FIJA (44 bytes, base64 de un HMAC-SHA256)
// a partir del secreto de la app (pepper) y la contraseña, antes de pasarla a bcrypt.
//
// El esquema anterior concatenaba el secreto en claro delante de la contraseña
// (`secret+password`) y lo pasaba directo a bcrypt. Como bcrypt SOLO considera los primeros
// 72 bytes de su entrada, el secreto consumía los bytes iniciales y a la contraseña le
// quedaban `72 - len(secret)`: con un secreto largo la entropía útil de la contraseña se
// reducía drásticamente (y si el secreto superaba 72 bytes, la contraseña se ignoraba por
// completo). Pre-derivar con HMAC-SHA256 produce siempre 32 bytes (44 en base64), muy por
// debajo del límite de bcrypt, eliminando la truncación sin sacrificar el pepper.
func pepperedPassword(secret []byte, password string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(password))
	sum := mac.Sum(nil)
	out := make([]byte, base64.StdEncoding.EncodedLen(len(sum)))
	base64.StdEncoding.Encode(out, sum)
	return out
}

// hashPassword genera el hash bcrypt de una contraseña usando el esquema HMAC+bcrypt.
func hashPassword(secret []byte, password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword(pepperedPassword(secret, password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// verifyPassword comprueba una contraseña contra el hash almacenado.
//
// Acepta tanto el esquema nuevo (HMAC+bcrypt) como el legacy (bcrypt(secret+password)) para
// no invalidar los hashes existentes. Si el hash es legacy y la contraseña es correcta,
// needsUpgrade=true para que el llamante lo re-hashee de forma transparente al nuevo esquema.
func verifyPassword(secret []byte, storedHash, password string) (ok bool, needsUpgrade bool) {
	if bcrypt.CompareHashAndPassword([]byte(storedHash), pepperedPassword(secret, password)) == nil {
		return true, false
	}
	// Legacy: pepper concatenado en claro (bcrypt trunca a 72 bytes).
	legacy := []byte(string(secret) + password)
	if bcrypt.CompareHashAndPassword([]byte(storedHash), legacy) == nil {
		return true, true
	}
	return false, false
}
