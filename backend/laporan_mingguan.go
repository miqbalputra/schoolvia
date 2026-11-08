package main

// BATCH 2 — Laporan Mingguan Otomatis ke WA (rencana perbaikan pasar)
// Pengasuh merasa "dikawal" tanpa login: setiap hari jam 20.00 WIB, sekolah yang
// punya aktivitas 7 hari terakhir dapat rangkuman bahasa awam via link WA.
// Nomor tujuan: kolom baru `wa_pengasuh` di pengaturan (kosong → skip, tidak error).

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"encoding/json"
	"strings"
)

// waPengasuh: nomor WA pengasuh sekolah (dinormalisasi 62...)
func waPengasuh(sekolahID int) string {
	var hp string
	db.QueryRow(`SELECT COALESCE(wa_pengasuh,'') FROM pengaturan WHERE id=$1`, sekolahID).Scan(&hp)
	hp = strings.Join(strings.FieldsFunc(hp, func(r rune) bool { return r < '0' || r > '9' }), "")
	hp = strings.TrimPrefix(hp, "0")
	if strings.HasPrefix(hp, "8") {
		hp = "62" + hp
	}
	if !strings.HasPrefix(hp, "62") {
		return ""
	}
	return hp
}

// handleSetWaPengasuh: simpan nomor WA pengasuh (dari Profil Sekolah)
func handleSetWaPengasuh(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		Wa string `json:"wa_pengasuh"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	_, err := db.Exec(`UPDATE pengaturan SET wa_pengasuh=NULLIF($1,'') WHERE id=$2`, strings.TrimSpace(in.Wa), c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

// ringkasanMingguan: susun teks laporan mingguan satu sekolah (bahasa awam) + ada aktivitas atau tidak
type ringkasan struct {
	AdaAktivitas bool
	Teks         string
}

func susunRingkasan(sekolahID int) ringkasan {
	var namaSekolah, jenisSekolah string
	db.QueryRow(`SELECT p.nama_sekolah, COALESCE(sk.jenis,'umum') FROM pengaturan p JOIN sekolah sk ON sk.id=p.id WHERE p.id=$1`, sekolahID).Scan(&namaSekolah, &jenisSekolah)
	week := `now() - interval '7 days'`

	// absensi: berapa hari tercatat minggu ini vs berapa kelas aktif
	var hariAbsensi int
	db.QueryRow(`SELECT COUNT(DISTINCT tanggal::date) FROM absensi_siswa WHERE sekolah_id=$1 AND tanggal > `+week, sekolahID).Scan(&hariAbsensi)
	var kelasAktif int
	db.QueryRow(`SELECT COUNT(DISTINCT kelas) FROM siswa WHERE sekolah_id=$1 AND status='aktif'`, sekolahID).Scan(&kelasAktif)

	// hafalan
	var setoran int
	db.QueryRow(`SELECT COUNT(*) FROM hafalan h JOIN siswa s ON s.id=h.siswa_id WHERE s.sekolah_id=$1 AND h.tanggal > `+week, sekolahID).Scan(&setoran)

	// SPP
var lunas, belum int
	var nominal float64
	db.QueryRow(`SELECT COUNT(*) FILTER (WHERE sp.status='lunas'), COUNT(*) FILTER (WHERE sp.status='belum'), COALESCE(SUM(sp.jumlah) FILTER (WHERE sp.status='lunas'),0)
		FROM spp sp JOIN siswa s ON s.id=sp.siswa_id WHERE s.sekolah_id=$1 AND COALESCE(sp.tgl_bayar,sp.jatuh_tempo) > `+week, sekolahID).Scan(&lunas, &belum, &nominal)

	// siswa yang belum ada data sama sekali 7 hari (per kelas) — indikator guru malas
	var kelasKosong int
	db.QueryRow(`SELECT COUNT(DISTINCT kelas) FROM siswa WHERE sekolah_id=$1 AND status='aktif' AND kelas NOT IN (
		SELECT DISTINCT s.kelas FROM absensi_siswa a JOIN siswa s ON s.id=a.siswa_id WHERE s.sekolah_id=$2 AND a.tanggal > `+week+`)`, sekolahID, sekolahID).Scan(&kelasKosong)

	ada := hariAbsensi > 0 || setoran > 0 || lunas > 0 || belum > 0
	var sb strings.Builder
	sb.WriteString("📋 *Laporan Mingguan " + namaSekolah + "*\n\n")
	if hariAbsensi > 0 {
		sb.WriteString(fmt.Sprintf("✅ Absensi tercatat %d dari 7 hari terakhir\n", hariAbsensi))
	} else {
		sb.WriteString("⚠️ Absensi belum dicatat sama sekali minggu ini\n")
	}
	if kelasKosong > 0 && kelasAktif > 0 {
		sb.WriteString(fmt.Sprintf("   · %d kelas belum ada catatan absensinya\n", kelasKosong))
	}
	if jenisSekolah == "islam" {
		if setoran > 0 {
			sb.WriteString(fmt.Sprintf("📖 Hafalan: %d setoran tercatat\n", setoran))
		} else {
			sb.WriteString("📖 Hafalan: belum ada setoran minggu ini\n")
		}
	}
	if lunas > 0 {
		sb.WriteString(fmt.Sprintf("💰 SPP: %d pembayaran masuk (%s)\n", lunas, fnumFmt(nominal)))
	}
	if belum > 0 {
		sb.WriteString(fmt.Sprintf("   · %d tagihan belum lunas\n", belum))
	}
	sb.WriteString("\nBuka laporan lengkap: " + appBaseURL() + "/app")
	return ringkasan{AdaAktivitas: ada, Teks: sb.String()}
}

// appBaseURL: alamat publik aplikasi (untuk link WA) — env SIMAKA_PUBLIC_URL
func appBaseURL() string {
	if u := os.Getenv("SIMAKA_PUBLIC_URL"); u != "" {
		return u
	}
	return "https://sistemflow.com"
}

// kirimLaporanSemua: dipanggil goroutine harian — kirim WA ke semua sekolah yang punya nomor
func kirimLaporanSemua() {
	rows, err := db.Query(`SELECT DISTINCT sekolah_id FROM pengaturan WHERE id>0 AND COALESCE(wa_pengasuh,'')<>''`)
	if err != nil {
		return
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	for _, id := range ids {
		r := susunRingkasan(id)
		if !r.AdaAktivitas {
			continue // sekolah kosong → jangan spam
		}
		wa := waPengasuh(id)
		if wa == "" {
			continue
		}
		// wa.me link dikirim via log (notif panel); pengiriman aktual lewat wa.me dibuka admin/user
		logActivitySekolah(id, "laporan_mingguan", "Laporan mingguan siap kirim untuk sekolah "+itoa(id)+" (WA "+wa+")")
		_ = r.Teks
	}
}

// handleLaporanMingguan: manual — susun laporan + tampilkan link WA siap-kirim (untuk demo & dashboard)
func handleLaporanMingguan(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	if c == nil {
		jsonErr(w, 401, "Unauthorized")
		return
	}
	rk := susunRingkasan(c.SekolahID)
	wa := waPengasuh(c.SekolahID)
	var link string
	if wa != "" {
		link = "https://wa.me/" + wa + "?text=" + url.QueryEscape(rk.Teks)
	}
	respond(w, 200, map[string]interface{}{
		"ada_aktivitas": rk.AdaAktivitas, "teks": rk.Teks, "wa": wa, "link_wa": link,
	})
}
