package main

import (
	"net/http"
	"strconv"
	"time"
)

// Token bagikan (share) untuk kwitansi: token acak mandiri (BUKAN JWT),
// kadaluarsa 30 hari, bisa di-regenerate kapan saja. Aman dikirim via WhatsApp
// ke orang tua karena tidak membuka akses API.

func migrateShares() error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS kwitansi_share (
		id SERIAL PRIMARY KEY,
		spp_id INT NOT NULL UNIQUE REFERENCES spp(id) ON DELETE CASCADE,
		token TEXT NOT NULL UNIQUE,
		kadaluarsa TIMESTAMPTZ NOT NULL,
		dibuat_oleh INT REFERENCES users(id),
		dibuat TIMESTAMPTZ DEFAULT now()
	)`)
	return err
}

// kwitansiShareCheck: validasi token bagikan → return sekolah_id pemilik spp.
func kwitansiShareCheck(tok string, sppID int) (int, bool) {
	if tok == "" {
		return 0, false
	}
	var sid int
	err := db.QueryRow(`SELECT sp.sekolah_id FROM kwitansi_share k JOIN spp sp ON sp.id=k.spp_id
		WHERE k.token=$1 AND k.spp_id=$2 AND k.kadaluarsa > now()`, tok, sppID).Scan(&sid)
	if err != nil {
		return 0, false
	}
	return sid, true
}

// authOrShare: lolos bila ada JWT valid di header, ATAU token bagikan kwitansi valid (?share=).
func authOrShare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c := claimsFromRequest(r); c != nil && tokenVersionOK(c.UserID, c.TokenVersion) {
			r = r.WithContext(contextWithUser(r, c))
			next.ServeHTTP(w, r)
			return
		}
		id, _ := strconv.Atoi(r.PathValue("id"))
		if sid, ok := kwitansiShareCheck(r.URL.Query().Get("share"), id); ok {
			r = r.WithContext(contextWithShare(r, sid))
			next.ServeHTTP(w, r)
			return
		}
		jsonErr(w, 401, "Unauthorized")
	})
}

// handleKwitansiShare: POST /api/kwitansi/{id}/share → buat/regenerate token bagikan (30 hari).
func handleKwitansiShare(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var sid int
	if err := db.QueryRow(`SELECT sekolah_id FROM spp WHERE id=$1 AND status='lunas'`, id).Scan(&sid); err != nil || sid != c.SekolahID {
		jsonErr(w, 404, "Kwitansi tidak ditemukan (SPP belum lunas?)")
		return
	}
	tok, err := waliTokenNew()
	if err != nil {
		jsonErr(w, 500, "Gagal membuat token")
		return
	}
	kadaluarsa := time.Now().AddDate(0, 1, 0) // 30 hari
	if _, err := db.Exec(`INSERT INTO kwitansi_share (spp_id, token, kadaluarsa, dibuat_oleh) VALUES ($1,$2,$3,$4)
		ON CONFLICT (spp_id) DO UPDATE SET token=EXCLUDED.token, kadaluarsa=EXCLUDED.kadaluarsa, dibuat_oleh=EXCLUDED.dibuat_oleh`,
		id, tok, kadaluarsa, c.UserID); err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	logActivity(c.UserID, "kwitansi_share", "Membuat token bagikan kwitansi SPP ID "+itoa(id))
	respond(w, 200, map[string]string{"token": tok, "kadaluarsa": kadaluarsa.Format("2006-01-02")})
}
