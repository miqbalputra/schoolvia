package main

import (
	"database/sql"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Rate limit login: max 5 gagal per IP+email, lock 15 menit.
// Sumber kebenaran = tabel login_fails (persisten antar restart);
// cache in-memory dipakai untuk kecepatan baca.

const (
	maxFails     = 5
	lockDuration = 15 * time.Minute
)

var loginMu sync.Mutex

// loginRateLimit: return "" jika boleh, atau pesan lock.
func loginRateLimit(key string) string {
	loginMu.Lock()
	defer loginMu.Unlock()
	var cnt int
	var lockedUntil time.Time
	err := db.QueryRow(`SELECT count, locked_until FROM login_fails WHERE key=$1`, key).Scan(&cnt, &lockedUntil)
	if err == sql.ErrNoRows {
		return "" // belum ada record
	}
	if err != nil {
		return "Sistem sedang sibuk, coba lagi sebentar." // fail-closed: jangan izinkan saat DB error
	}
	if time.Now().Before(lockedUntil) {
		sisa := int(time.Until(lockedUntil).Minutes()) + 1
		return "Terlalu banyak percobaan gagal. Coba lagi dalam " + itoa(sisa) + " menit."
	}
	if cnt >= maxFails && lockedUntil.IsZero() {
		// capai batas tapi belum diberi lock (mis. restart lama) — set lock sekarang
		db.Exec(`UPDATE login_fails SET locked_until=$1 WHERE key=$2`, time.Now().Add(lockDuration), key)
		return "Terlalu banyak percobaan gagal. Coba lagi dalam 15 menit."
	}
	return ""
}

func loginRecordFail(key string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	var cnt int
	var lockedUntil time.Time
	db.QueryRow(`SELECT count, locked_until FROM login_fails WHERE key=$1`, key).Scan(&cnt, &lockedUntil)
	if !lockedUntil.IsZero() && time.Now().Before(lockedUntil) {
		return // sudah lock, jangan hitung lagi
	}
	cnt++
	if cnt >= maxFails {
		db.Exec(`INSERT INTO login_fails (key, count, locked_until) VALUES ($1,$2,$3)
			ON CONFLICT (key) DO UPDATE SET count=$2, locked_until=$3`,
			key, 0, time.Now().Add(lockDuration))
		return
	}
	db.Exec(`INSERT INTO login_fails (key, count) VALUES ($1,$2)
		ON CONFLICT (key) DO UPDATE SET count=$2`, key, cnt)
}

func loginRecordSuccess(key string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	db.Exec(`DELETE FROM login_fails WHERE key=$1`, key)
}

// clientIP: IP klien. X-Forwarded-For hanya dipercaya bila TRUSTED_PROXY=true
// (di belakang reverse proxy; ambil entry terakhir yang ditambahkan proxy kita).
func clientIP(r *http.Request) string {
	if os.Getenv("TRUSTED_PROXY") == "true" {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
				return ip
			}
		}
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

var daftarMu sync.Mutex
var daftarHits = map[string][]time.Time{}

const daftarMaxPerHour = 10

// daftarRateLimit: batasi pendaftaran sekolah baru per IP (in-memory, maks 10/jam).
func daftarRateLimit(ip string) bool {
	daftarMu.Lock()
	defer daftarMu.Unlock()
	now := time.Now()
	cut := now.Add(-time.Hour)
	kept := daftarHits[ip][:0]
	for _, t := range daftarHits[ip] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	daftarHits[ip] = kept
	if len(kept) >= daftarMaxPerHour {
		return false
	}
	daftarHits[ip] = append(kept, now)
	return true
}

// logActivity: catat aksi user; sekolah_id diambil dari user agar log terikat tenant.
func logActivity(userID int, aksi, detail string) {
	db.Exec(`INSERT INTO activity_log (user_id, aksi, detail, sekolah_id)
		SELECT $1, $2, $3, u.sekolah_id FROM users u WHERE u.id=$1`, userID, aksi, detail)
}

// logActivitySekolah: log tanpa user (mis. proses sistem), tetap terikat tenant.
func logActivitySekolah(sekolahID int, aksi, detail string) {
	db.Exec(`INSERT INTO activity_log (user_id, aksi, detail, sekolah_id) VALUES (NULL, $1, $2, $3)`, aksi, detail, sekolahID)
}

var _ = http.MethodGet
