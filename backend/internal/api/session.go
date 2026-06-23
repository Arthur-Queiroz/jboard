package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

// Sessão da web em cookie assinado (HMAC), stateless — single-user, sem store.
// Formato do valor: base64url(expiryUnix) "." base64url(HMAC(secret, expiry)).
// A chave HMAC é o APIToken (já é o segredo do servidor).
const (
	sessionCookieName = "jboard_session"
	sessionTTL        = 7 * 24 * time.Hour
)

func mintSession(secret string, now time.Time) string {
	exp := strconv.FormatInt(now.Add(sessionTTL).Unix(), 10)
	return b64([]byte(exp)) + "." + b64(signSession(secret, exp))
}

// validSession confere a assinatura (constant-time) e a expiração.
func validSession(secret, value string, now time.Time) bool {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return false
	}
	expRaw, err1 := unb64(parts[0])
	sigRaw, err2 := unb64(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	if !hmac.Equal(sigRaw, signSession(secret, string(expRaw))) {
		return false
	}
	exp, err := strconv.ParseInt(string(expRaw), 10, 64)
	if err != nil {
		return false
	}
	return now.Unix() < exp
}

func signSession(secret, payload string) []byte {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(payload))
	return m.Sum(nil)
}

func b64(b []byte) string            { return base64.RawURLEncoding.EncodeToString(b) }
func unb64(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }
