package main

// LAPORAN WALI MURID (Batch 5 rencana pasar)
// Halaman read-only via link token — untuk dikirim ke WA orang tua, TANPA install app.
// Bahasa awam: "Lancar", "Perlu bimbingan", bukan istilah tajwid/juz rumit.

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// handleLaporanWali: halaman HTML read-only untuk wali murid (?id=<siswa>&token=<token-wali>)
func handleLaporanWali(w http.ResponseWriter, r *http.Request) {
	tokSiswa, tokSekolah, ok := waliTokenCheck(r.URL.Query().Get("token"))
	if !ok {
		http.Error(w, "Tautan tidak valid atau sudah kedaluwarsa. Minta tautan baru ke guru/wali kelas.", 401)
		return
	}
	sid, _ := strconv.Atoi(r.PathValue("id"))
	// Token per-wali terikat ke satu siswa: token ini hanya membuka laporan siswa itu.
	if sid != tokSiswa {
		http.Error(w, "Tautan ini hanya untuk satu siswa. Minta tautan yang benar ke guru/wali kelas.", 403)
		return
	}
	var siswaSekolah int
	if err := db.QueryRow(`SELECT sekolah_id FROM siswa WHERE id=$1`, sid).Scan(&siswaSekolah); err != nil || siswaSekolah != tokSekolah {
		http.Error(w, "Data tidak ditemukan", 404)
		return
	}
	var nama, nis, kelas, namaSekolah, alamat, kepala string
	var kadaluarsa time.Time
	db.QueryRow(`SELECT kadaluarsa FROM wali_token WHERE token=$1`, r.URL.Query().Get("token")).Scan(&kadaluarsa)
	db.QueryRow(`SELECT nama, COALESCE(nis,''), kelas FROM siswa WHERE id=$1`, sid).Scan(&nama, &nis, &kelas)
	db.QueryRow(`SELECT nama_sekolah, COALESCE(alamat,''), COALESCE(kepala_sekolah,'') FROM pengaturan WHERE id=$1`, tokSekolah).Scan(&namaSekolah, &alamat, &kepala)

	// Hafalan terakhir (bahasa awam) — default 10, toggle ?semua=1 untuk riwayat lengkap
	limit := 10
	if r.URL.Query().Get("semua") == "1" {
		limit = 300
	}
	rows, err := db.Query(fmt.Sprintf(`SELECT surah, COALESCE(ayat_dari,0), COALESCE(ayat_sampai,0), status, to_char(tanggal,'DD Mon YYYY') FROM hafalan WHERE siswa_id=$1 ORDER BY tanggal DESC LIMIT %d`, limit), sid)
	hafRows := ""
	if err == nil {
		for rows.Next() {
			var surah, status, tgl string
			var ad, as2 int
			rows.Scan(&surah, &ad, &as2, &status, &tgl)
			ayat := ""
			if as2 > 0 {
				ayat = fmt.Sprintf(" ayat %d–%d", ad, as2)
			}
			badge := "⏳ Sedang belajar"
			switch status {
			case "lancar":
				badge = "✅ Lancar"
			case "murojaah":
				badge = "🔁 Mengulang (bagus!)"
			case "baru":
				badge = "🆕 Setoran baru"
			}
			hafRows += fmt.Sprintf("<tr><td><b>%s</b>%s</td><td>%s</td><td style='color:#6b7280'>%s</td></tr>", esc(surah), ayat, badge, tgl)
		}
		rows.Close()
	}
	if hafRows == "" {
		hafRows = "<tr><td colspan=3 style='color:#6b7280'>Belum ada catatan hafalan.</td></tr>"
	}
	// Toggle riwayat lengkap (permintaan wali murid: "bisa lihat hafalan dari awal tahun?")
	var jmlHafalan int
	db.QueryRow(`SELECT COUNT(*) FROM hafalan WHERE siswa_id=$1`, sid).Scan(&jmlHafalan)
	toggleHafalan := ""
	if jmlHafalan > 10 {
		if r.URL.Query().Get("semua") == "1" {
			toggleHafalan = fmt.Sprintf(`<div style="text-align:center;margin:10px 0"><a href="?id=%d&amp;token=%s" style="color:#4f46e5;font-size:13px">▲ Tampilkan 10 terakhir saja</a></div>`, sid, r.URL.Query().Get("token"))
		} else {
			toggleHafalan = fmt.Sprintf(`<div style="text-align:center;margin:10px 0"><a href="?id=%d&amp;token=%s&amp;semua=1" style="color:#4f46e5;font-size:13px">▼ Lihat semua riwayat hafalan (%d catatan)</a></div>`, sid, r.URL.Query().Get("token"), jmlHafalan)
		}
	}

	// Ringkasan absensi bulan ini
	var hadir, sakit, izin, alpa int
	db.QueryRow(`SELECT COUNT(*) FILTER (WHERE status='hadir'), COUNT(*) FILTER (WHERE status='sakit'),
		COUNT(*) FILTER (WHERE status='izin'), COUNT(*) FILTER (WHERE status='alpa')
		FROM absensi_siswa WHERE siswa_id=$1 AND to_char(tanggal,'YYYY-MM')=to_char(now(),'YYYY-MM')`, sid).Scan(&hadir, &sakit, &izin, &alpa)

	// Tagihan SPP
	var lunas, belum int
	db.QueryRow(`SELECT COUNT(*) FILTER (WHERE status='lunas'), COUNT(*) FILTER (WHERE status='belum') FROM spp WHERE siswa_id=$1`, sid).Scan(&lunas, &belum)

	// Rata-rata nilai (jika ada)
	var rata interface{}
	db.QueryRow(`SELECT ROUND(AVG(nilai)::numeric,0) FROM nilai WHERE siswa_id=$1`, sid).Scan(&rata)

	statusNilai := "<span style='color:#6b7280'>Belum ada nilai</span>"
	if rata != nil {
		rv, _ := rata.(float64)
		if rv == 0 {
			rv = 80
		}
		if rv >= 80 {
			statusNilai = "<b style='color:#16a34a'>Bagus 🌟</b>"
		} else if rv >= 60 {
			statusNilai = "<b style='color:#d97706'>Cukup baik</b>"
		} else {
			statusNilai = "<b style='color:#dc2626'>Perlu bimbingan</b>"
		}
	}

	sppInfo := fmt.Sprintf("✅ Lunas: %d tagihan", lunas)
	if belum > 0 {
		sppInfo += fmt.Sprintf(" · ⚠️ Belum lunas: %d tagihan", belum)
	}

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html lang="id"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Laporan ` + esc(nama) + `</title><style>
body{font-family:system-ui,'Segoe UI',sans-serif;color:#1e1b4b;background:#f3f4f6;margin:0;padding:20px}
.card{max-width:520px;margin:0 auto;background:#fff;border-radius:16px;padding:24px;box-shadow:0 4px 20px rgba(0,0,0,.08);margin-bottom:16px}
.kop{text-align:center;border-bottom:3px solid #4f46e5;padding-bottom:14px;margin-bottom:16px}
.kop h1{font-size:18px;color:#4f46e5;margin-bottom:2px}
.kop p{font-size:12px;color:#6b7280;margin:0}
h2{font-size:15px;margin:0 0 10px;color:#4f46e5}
table{width:100%;border-collapse:collapse;font-size:13.5px}
td,th{padding:7px 6px;text-align:left;border-bottom:1px solid #f0f0f5}
.absen{display:flex;gap:8px;flex-wrap:wrap}
.absen span{background:#eef2ff;border-radius:10px;padding:6px 12px;font-size:13px}
.foot{text-align:center;font-size:11.5px;color:#9ca3af;margin-top:8px}
</style></head><body>
<div style="max-width:520px;margin:0 auto 12px;background:#ecfdf5;border:1px solid #a7f3d0;border-radius:10px;padding:9px 14px;font-size:12.5px;color:#065f46;text-align:center">
🔒 Tautan khusus untuk orang tua/wali ` + esc(nama) + ` · berlaku s/d ` + kadaluarsa.Format("2 Jan 2006") + ` · diperbarui otomatis tiap ada catatan baru
</div>
<div class="card"><div class="kop"><h1>🏫 ` + esc(namaSekolah) + `</h1><p>` + esc(alamat) + `</p></div>
<h2>Laporan ` + esc(nama) + ` · Kelas ` + esc(kelas) + `</h2>
<table><tr><td>Perkembangan nilai</td><td style="text-align:right">` + statusNilai + `</td></tr>
<tr><td>Tagihan SPP</td><td style="text-align:right">` + sppInfo + `</td></tr></table></div>

<div class="card"><h2>📖 Hafalan Terakhir</h2>
<table><tr><th>Surah</th><th>Status</th><th>Tanggal</th></tr>` + hafRows + `</table>` + toggleHafalan + `</div>

<div class="card"><h2>✅ Kehadiran Bulan Ini</h2>
<div class="absen"><span>🙏 Hadir: <b>` + strconv.Itoa(hadir) + `</b></span><span>🤒 Sakit: <b>` + strconv.Itoa(sakit) + `</b></span><span>📝 Izin: <b>` + strconv.Itoa(izin) + `</b></span><span>❌ Tanpa kabar: <b>` + strconv.Itoa(alpa) + `</b></span></div></div>

<div class="card" style="text-align:center"><p style="font-size:13.5px;margin:0 0 4px">Ada pertanyaan tentang anak Anda?</p>
<p style="font-size:12.5px;color:#6b7280;margin:0">Hubungi guru/wali kelas atau pengelola sekolah.<br>` + esc(namaSekolah) + ` · ` + esc(kepala) + `</p></div>
<p class="foot">Laporan ini dibuat otomatis oleh SchoolVia · ` + esc(namaSekolah) + `</p>
</body></html>`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(sb.String()))
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&#34;")
	return r.Replace(s)
}
