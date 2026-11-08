package main

import (
	"net/http"
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
	if err != nil {
		return "" // belum ada record
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
