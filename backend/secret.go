package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// loadOrCreateSecret: pakai SIMAKA_JWT_SECRET dari env; kalau kosong, baca/simpan
// di /etc/simaka/jwt_secret; kalau tidak bisa, simpan di ~/.simaka/jwt_secret (0600).
// Secret tidak pernah ditulis ke direktori kerja (risiko ikut ter-commit).
func loadOrCreateSecret() ([]byte, error) {
	if s := os.Getenv("SIMAKA_JWT_SECRET"); s != "" {
		return []byte(s), nil
	}
	if b, err := os.ReadFile("/etc/simaka/jwt_secret"); err == nil && len(b) >= 32 {
		return b, nil
	}
	buf := make([]byte, 48)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	sec := []byte(base64.StdEncoding.EncodeToString(buf))
	if err := os.MkdirAll("/etc/simaka", 0700); err == nil {
		if err := os.WriteFile("/etc/simaka/jwt_secret", sec, 0600); err == nil {
			fmt.Println("JWT secret baru dibuat: /etc/simaka/jwt_secret")
			return sec, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("tidak bisa menentukan home dir untuk menyimpan jwt secret: %v", err)
	}
	dir := filepath.Join(home, ".simaka")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "jwt_secret")
	if b, err := os.ReadFile(path); err == nil && len(b) >= 32 {
		return b, nil
	}
	if err := os.WriteFile(path, sec, 0600); err != nil {
		return nil, err
	}
	fmt.Println("JWT secret baru dibuat:", path)
	return sec, nil
}
