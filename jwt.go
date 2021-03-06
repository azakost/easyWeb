// A light-weight JWT-manager written without any external dependencies except
// standard packages of Go provides features for JSON Web Token Creation and
// validation.
package easyWeb

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"
)

// Default configurations
var (
	Secret       = "defaultpassphrase"
	TokenRefresh = time.Hour
)

type JWT struct {
	User    JWTUser   `json:"user"`
	Expires time.Time `json:"expires"`
	Token   string    `json:"token"`
}

type JWTUser struct {
	Uid      int64  `json:"uid"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Role     string `json:"role"`
}

// CreateJWT consumes an empty JWT struct with pre-filled User.Id, User.Role
// & expiration time (Expires) and then returns its an encrypted version as a
// string.
func CreateJWT(data JWT) string {
	tok := []byte(strconv.FormatInt(data.User.Uid, 10) + data.Expires.String())
	data.Token = Encrypt(tok)
	marshaled, errorMarshal := json.Marshal(data)
	err(errorMarshal)
	return Encrypt(marshaled)
}

// ReadJWT decrypts a given JSON Web Token and returns two validation booleans.
// First is for general validation, second is a signal for token refreshment.
func ReadJWT(value string) (JWT, bool, bool) {
	var blank JWT
	data, errorDecrypt := Decrypt(value)
	if errorDecrypt != nil {
		return blank, false, false
	}
	errorUnmarshal := json.Unmarshal(data, &blank)
	if errorUnmarshal != nil {
		return blank, false, false
	}
	checkString := strconv.FormatInt(blank.User.Uid, 10) + blank.Expires.String()
	decrypted, errorDecryptToken := Decrypt(blank.Token)
	if errorDecryptToken != nil {
		return blank, false, false
	}
	if strings.Split(string(decrypted), " m=")[0] != checkString {
		return blank, false, false
	}
	if time.Now().After(blank.Expires) {
		return blank, false, false
	}
	if blank.Expires.Before(time.Now().Add(TokenRefresh)) {
		return blank, true, true
	}
	return blank, true, false
}

func Encrypt(data []byte) string {
	gcm := gcm()
	nonce := make([]byte, gcm.NonceSize())
	_, errorRead := io.ReadFull(rand.Reader, nonce)
	err(errorRead)
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString([]byte(ciphertext))
}

func Decrypt(value string) ([]byte, error) {
	data, errorBase64 := base64.StdEncoding.DecodeString(value)
	if errorBase64 != nil {
		return nil, errorBase64
	}
	gcm := gcm()
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	decrypted, errorOpen := gcm.Open(nil, nonce, ciphertext, nil)
	if errorOpen != nil {
		return nil, errorOpen
	}
	return decrypted, nil
}

func gcm() cipher.AEAD {
	hasher := md5.New()
	_, errorWrite := hasher.Write([]byte(Secret))
	err(errorWrite)
	hash := hex.EncodeToString(hasher.Sum(nil))
	block, errorAes := aes.NewCipher([]byte(hash))
	err(errorAes)
	gcm, errorGCM := cipher.NewGCM(block)
	err(errorGCM)
	return gcm
}
