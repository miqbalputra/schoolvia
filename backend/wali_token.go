package main

// Token per-wali untuk link laporan (Batch A rencana pasar ronde 2).
// Keberatan privasi dari simulasi: token JWT akun statis dianggap "longgar"
// karena selalu sama selamanya. Solusi: token acak per-siswa, kadaluarsa 90 hari,
// bisa di-reset (token lama mati seketika).

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// waliTokenGet: ambil token aktif siswa (buat kalau belum ada/kadaluarsa).
func waliTokenGet(sekolahID, siswaID int) (string, error) {
	var tok string
	var kad time.Time
	err := db.QueryRow(`SELECT token, kadaluarsa FROM wali_token WHERE siswa_id=$1 AND sekolah_id=$2`, siswaID, sekolahID).Scan(&tok, &kad)
	if err == nil && time.Now().Before(kad) {
		return tok, nil
	}
	// belum ada atau kedaluwarsa → buat baru (upsert per siswa, satu token aktif per siswa)
	tok, err = waliTokenNew()
	if err != nil {
		return "", err
	}
	kadaluarsa := time.Now().AddDate(0, 3, 0) // 90 hari
	_, err = db.Exec(`INSERT INTO wali_token (siswa_id, sekolah_id, token, kadaluarsa)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (siswa_id, sekolah_id) DO UPDATE SET token=EXCLUDED.token, kadaluarsa=EXCLUDED.kadaluarsa`,
		siswaID, sekolahID, tok, kadaluarsa)
	if err != nil {
		return "", err
	}
	return tok, nil
}

// waliTokenNew: token acak 32 byte hex.
func waliTokenNew() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// waliTokenCheck: validasi token untuk halaman laporan publik.
// Return (siswaID, sekolahID, ok).
func waliTokenCheck(tokStr string) (int, int, bool) {
	if tokStr == "" {
		return 0, 0, false
	}
	var sid, sekolahID int
	err := db.QueryRow(`SELECT siswa_id, sekolah_id FROM wali_token WHERE token=$1 AND kadaluarsa > now()`, tokStr).Scan(&sid, &sekolahID)
	if err != nil {
		return 0, 0, false
	}
	return sid, sekolahID, true
}

// handleWaliToken: GET /api/wali-token/{siswa_id} → {token} (admin/TU sekolah pemilik).
// POST /api/wali-token/{siswa_id} → reset (token lama mati).
func handleWaliTokenGet(w http.ResponseWriter, r *http.Request, c *Claims) {
	sid := pathInt(r)
	var ok int
	if err := db.QueryRow(`SELECT 1 FROM siswa WHERE id=$1 AND sekolah_id=$2`, sid, c.SekolahID).Scan(&ok); err != nil {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return
	}
	tok, err := waliTokenGet(c.SekolahID, sid)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 200, map[string]string{"token": tok})
}

func handleWaliTokenReset(w http.ResponseWriter, r *http.Request, c *Claims) {
	sid := pathInt(r)
	var ok int
	if err := db.QueryRow(`SELECT 1 FROM siswa WHERE id=$1 AND sekolah_id=$2`, sid, c.SekolahID).Scan(&ok); err != nil {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return
	}
	if _, err := db.Exec(`DELETE FROM wali_token WHERE siswa_id=$1 AND sekolah_id=$2`, sid, c.SekolahID); err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	tok, err := waliTokenGet(c.SekolahID, sid)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	logActivity(c.UserID, "reset_wali_token", "Reset token laporan wali siswa ID "+itoa(sid))
	respond(w, 200, map[string]string{"token": tok})
}

// pathInt: ambil {id} path value sebagai int.
func pathInt(r *http.Request) int {
	n := 0
	fmt.Sscanf(r.PathValue("id"), "%d", &n)
	return n
}
