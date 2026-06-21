package authtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Claims struct {
	UserID int64  `json:"uid"`
	Login  string `json:"login"`
	Exp    int64  `json:"exp"`
}

func Generate(secret []byte, userID int64, login string, ttl time.Duration) (string, error) {
	claims := Claims{UserID: userID, Login: login, Exp: time.Now().Add(ttl).Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return encodedPayload + "." + sign(secret, encodedPayload), nil
}

func Verify(secret []byte, token string) (Claims, error) {
	dot := strings.LastIndex(token, ".")
	if dot < 0 {
		return Claims{}, errors.New("malformed token")
	}
	encodedPayload, sig := token[:dot], token[dot+1:]
	if subtle.ConstantTimeCompare([]byte(sig), []byte(sign(secret, encodedPayload))) != 1 {
		return Claims{}, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return Claims{}, err
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, err
	}
	if time.Now().Unix() > claims.Exp {
		return Claims{}, errors.New("token expired")
	}
	return claims, nil
}

func sign(secret []byte, data string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
