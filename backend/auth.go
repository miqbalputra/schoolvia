package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("diisi-saat-startup-oleh-loadOrCreateSecret")

// dummyHash: dipakai saat email tidak ditemukan agar waktu respons sama
// (mencegah user-enumeration via timing).
var dummyHash = hashPass("schoolvia-timing-guard-0000")

func hashPass(p string) string {
	b, _ := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(b)
}

func checkPass(hash, p string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)) == nil
}

type Claims struct {
	UserID    int    `json:"uid"`
	Nama      string `json:"nama"`
	Role      string `json:"role"`
	SekolahID int    `json:"sid"`
	jwt.RegisteredClaims
}

func makeToken(u *User) (string, error) {
	c := Claims{UserID: u.ID, Nama: u.Nama, Role: u.Role, SekolahID: u.SekolahID,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour))}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(jwtSecret)
}

// claimsFromRequest: parse JWT HANYA dari header Authorization.
// Token tidak lagi diterima via query string (?token=) agar tidak terekam di log server/history browser.
func claimsFromRequest(r *http.Request) *Claims {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil
	}
	tok, err := jwt.ParseWithClaims(strings.TrimPrefix(h, "Bearer "), &Claims{},
		func(t *jwt.Token) (interface{}, error) { return jwtSecret, nil })
	if err != nil || !tok.Valid {
		return nil
	}
	return tok.Claims.(*Claims)
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := claimsFromRequest(r)
		if c == nil {
			jsonErr(w, 401, "Token tidak valid")
			return
		}
		r = r.WithContext(contextWithUser(r, c))
		next.ServeHTTP(w, r)
	})
}

// requireRole membatasi akses: mis. requireRole(w,r,"kepala_sekolah","tu")
func requireRole(w http.ResponseWriter, r *http.Request, roles ...string) *Claims {
	c := userFromContext(r)
	if c == nil {
		jsonErr(w, 401, "Unauthorized")
		return nil
	}
	for _, role := range roles {
		if c.Role == role {
			return c
		}
	}
	jsonErr(w, 403, "Akses ditolak untuk peran "+c.Role)
	return nil
}

func respond(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	respond(w, code, map[string]string{"error": msg})
}
