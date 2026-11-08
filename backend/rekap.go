package main

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
)


var bulanIndo = map[string]string{"Jan":"Jan","Feb":"Feb","Mar":"Mar","Apr":"Apr","May":"Mei","Jun":"Jun","Jul":"Jul","Aug":"Ags","Sep":"Sep","Oct":"Okt","Nov":"Nov","Dec":"Des"}

func tglIndo(s string) string {
	for en, id := range bulanIndo {
		s = strings.ReplaceAll(s, "TM"+en, id)
	}
	return s
}

// ---------- LAPORAN REKAP BULANAN (Fase C) ----------
// Rekap satu bulan: absensi per kelas, SPP (lunas/belum + nominal), pengeluaran,
// hafalan, nilai rata-rata — semuanya terfilter tenant.

func handleRekapBulanan(w http.ResponseWriter, r *http.Request, c *Claims) {
	bulan := r.URL.Query().Get("bulan") // YYYY-MM
	if len(bulan) != 7 {
		bulan = fmt.Sprintf("%s-%02d", r.URL.Query().Get("tahun"), bulanNum(r.URL.Query().Get("bulan")))
	}
	if len(bulan) != 7 || bulan[4] != '-' {
		jsonErr(w, 400, "Format bulan harus YYYY-MM, mis. 2026-10")
		return
	}
	if _, err := strconv.Atoi(bulan[:4]); err != nil {
		jsonErr(w, 400, "Tahun tidak valid")
		return
	}
	if _, err := strconv.Atoi(bulan[5:]); err != nil || bulan[5:] == "00" || bulan[5:] > "12" {
		jsonErr(w, 400, "Bulan tidak valid")
		return
	}

	out := map[string]interface{}{"bulan": bulan}

	// 1. Rekap absensi siswa per kelas
	rows, err := db.Query(`SELECT s.kelas,
			COUNT(*) FILTER (WHERE a.status='hadir'),
			COUNT(*) FILTER (WHERE a.status='sakit'),
			COUNT(*) FILTER (WHERE a.status='izin'),
			COUNT(*) FILTER (WHERE a.status='alpa')
		FROM absensi_siswa a JOIN siswa s ON s.id=a.siswa_id
		WHERE s.sekolah_id=$1 AND to_char(a.tanggal,'YYYY-MM')=$2
		GROUP BY s.kelas ORDER BY s.kelas`, c.SekolahID, bulan)
	if err == nil {
		rekap := []map[string]interface{}{}
		for rows.Next() {
			var k string
			var h, sk, iz, al int
			rows.Scan(&k, &h, &sk, &iz, &al)
			rekap = append(rekap, map[string]interface{}{"kelas": k, "hadir": h, "sakit": sk, "izin": iz, "alpa": al})
		}
		rows.Close()
		out["absensi"] = rekap
	}

	// 2. Rekap SPP
	var lunas, belum, totalLunas int
	db.QueryRow(`SELECT COUNT(*) FILTER (WHERE sp.status='lunas'), COUNT(*) FILTER (WHERE sp.status!='lunas'),
			COALESCE(SUM(sp.jumlah) FILTER (WHERE sp.status='lunas'),0)
		FROM spp sp JOIN siswa s ON s.id=sp.siswa_id
		WHERE s.sekolah_id=$1 AND sp.bulan=$2`, c.SekolahID, bulanNumName(bulan)).
		Scan(&lunas, &belum, &totalLunas)
	out["spp"] = map[string]interface{}{"lunas": lunas, "belum": belum, "total_lunas": totalLunas}

	// 3. Keuangan: pemasukan vs pengeluaran
	var pemasukan, pengeluaran int
	db.QueryRow(`SELECT COALESCE(SUM(sp.jumlah),0) FROM spp sp JOIN siswa s ON s.id=sp.siswa_id
		WHERE s.sekolah_id=$1 AND sp.status='lunas' AND to_char(sp.tgl_bayar,'YYYY-MM')=$2`,
		c.SekolahID, bulan).Scan(&pemasukan)
	db.QueryRow(`SELECT COALESCE(SUM(jumlah),0) FROM pengeluaran WHERE sekolah_id=$1 AND to_char(tanggal,'YYYY-MM')=$2`,
		c.SekolahID, bulan).Scan(&pengeluaran)
	out["keuangan"] = map[string]interface{}{"pemasukan": pemasukan, "pengeluaran": pengeluaran, "selisih": pemasukan - pengeluaran}

	// 4. Hafalan per status — hanya lembaga islam
	var jenis string
	db.QueryRow(`SELECT COALESCE(jenis,'umum') FROM sekolah WHERE id=$1`, c.SekolahID).Scan(&jenis)
	if jenis == "islam" {
	rows, err = db.Query(`SELECT h.status, COUNT(*) FROM hafalan h JOIN siswa s ON s.id=h.siswa_id
		WHERE s.sekolah_id=$1 AND to_char(h.tanggal,'YYYY-MM')=$2 GROUP BY h.status`, c.SekolahID, bulan)
	if err == nil {
		hf := map[string]int{}
		for rows.Next() {
			var st string
			var n int
			rows.Scan(&st, &n)
			hf[st] = n
		}
		rows.Close()
		out["hafalan"] = hf
		}
	}

	// 5. Rata-rata nilai per kelas
	rows, err = db.Query(`SELECT s.kelas, ROUND(AVG(n.nilai)::numeric,1)::float8 FROM nilai n JOIN siswa s ON s.id=n.siswa_id
		WHERE s.sekolah_id=$1 AND to_char(n.created_at,'YYYY-MM')=$2 GROUP BY s.kelas ORDER BY s.kelas`,
		c.SekolahID, bulan)
	if err == nil {
		nilai := []map[string]interface{}{}
		for rows.Next() {
			var k string
			var avg float64
			rows.Scan(&k, &avg)
			nilai = append(nilai, map[string]interface{}{"kelas": k, "rata": avg})
		}
		rows.Close()
		out["nilai_rata"] = nilai
	}

	logActivity(c.UserID, "laporan_rekap", "Rekap bulanan "+bulan)
	respond(w, 200, out)
}

func bulanNum(b string) int {
	n, _ := strconv.Atoi(b)
	if n < 1 || n > 12 {
		return 1
	}
	return n
}

func bulanNumName(bulan string) string {
	// kolom spp memakai nama bulan Indonesia ("November") — konversi dari YYYY-MM
	m, err := strconv.Atoi(bulan[5:])
	if err != nil || m < 1 || m > 12 {
		return ""
	}
	return bulanID[m-1]
}

// ---------- KWITANSI SPP (Fase C) ----------
// Halaman HTML siap cetak (print-friendly, thermal & A5), token di URL seperti rapor-cetak.

func handleKwitansi(w http.ResponseWriter, r *http.Request) {
	sekolahID := 0
	if c := userFromContext(r); c != nil {
		sekolahID = c.SekolahID
	} else {
		sekolahID = shareFromContext(r) // token bagikan (?share=), bukan JWT
	}
	if sekolahID == 0 {
		http.Error(w, "Unauthorized", 401)
		return
	}
	id, _ := strconv.Atoi(r.PathValue("id"))
	var nama, kelas, bulan, tahun, tglBayar, metode, petugas, namaSekolah, alamat, kepala string
	var jumlah float64
	var sid int
	err := db.QueryRow(`SELECT s.nama, s.kelas, sp.bulan, sp.tahun, COALESCE(to_char(sp.tgl_bayar,'DD TMBln YYYY'),'-'),
			COALESCE(sp.metode,'tunai'), COALESCE(u.nama,'-'), sk.nama_sekolah, COALESCE(sk.alamat,''), COALESCE(sk.kepala_sekolah,''), sp.sekolah_id, sp.jumlah
		FROM spp sp
		JOIN siswa s ON s.id=sp.siswa_id
		LEFT JOIN users u ON u.id=sp.petugas_id
		JOIN pengaturan sk ON sk.id=$2
		WHERE sp.id=$1 AND sp.status='lunas' AND s.sekolah_id=$2`, id, sekolahID).
		Scan(&nama, &kelas, &bulan, &tahun, &tglBayar, &metode, &petugas, &namaSekolah, &alamat, &kepala, &sid, &jumlah)
	if err != nil {
		http.Error(w, "Kwitansi tidak ditemukan (SPP belum lunas?)", 404)
		return
	}
	e := html.EscapeString
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html><html lang="id"><head><meta charset="utf-8"><title>Kwitansi SPP %s</title>
<style>
 @media print { .noprint{display:none} body{margin:0} }
 body{font-family:'Segoe UI',Arial,sans-serif;background:#f1f2f7;display:flex;justify-content:center;padding:24px;margin:0}
 .kw{background:#fff;width:420px;max-width:94vw;border:1px solid #e5e7f0;border-radius:12px;padding:26px 30px;box-shadow:0 4px 24px rgba(30,27,75,.08)}
 .kop{display:flex;justify-content:space-between;align-items:flex-start;border-bottom:3px double #4f46e5;padding-bottom:10px;margin-bottom:16px}
 .kop h1{font-size:17px;margin:0;color:#1e1b4b}
 .kop small{color:#6b7280;font-size:11px;display:block;margin-top:2px}
 .no{font-size:11px;color:#6b7280;text-align:right}
 h2{font-size:14px;text-align:center;color:#4f46e5;letter-spacing:2px;margin:0 0 14px}
 table{width:100%%;font-size:13px;border-collapse:collapse}
 td{padding:6px 0;vertical-align:top}
 td.k{color:#6b7280;width:120px}
 td.v b{font-size:15px}
 .jml{border-top:1px dashed #d1d5db;margin-top:8px;padding-top:10px;font-size:15px}
 .jml b{color:#1e1b4b;font-size:18px}
 .ttd{margin-top:36px;display:flex;justify-content:space-between;font-size:12px;color:#374151}
 .ttd .box{text-align:center;width:160px}
 .ttd .sp{display:block;height:48px}
 .btn{position:fixed;top:14px;right:14px;background:#4f46e5;color:#fff;border:none;padding:10px 18px;border-radius:8px;font-size:13px;cursor:pointer}
 .foot{margin-top:14px;font-size:10px;color:#9ca3af;text-align:center}
</style></head><body>
<button class="btn noprint" onclick="print()">🖨️ Cetak</button>
<div class="kw">
 <div class="kop"><div><h1>%s</h1><small>%s</small></div><div class="no">No. KW/%s/%s</div></div>
 <h2>KWITANSI SPP</h2>
 <table>
  <tr><td class="k">Telah diterima dari</td><td class="v"><b>%s</b><br><small>Kelas %s</small></td></tr>
  <tr><td class="k">Untuk pembayaran</td><td class="v">SPP bulan %s %s</td></tr>
  <tr><td class="k">Tanggal bayar</td><td class="v">%s</td></tr>
  <tr><td class="k">Metode</td><td class="v">%s</td></tr>
  <tr><td class="k">Petugas</td><td class="v">%s</td></tr>
 </table>
 <div class="jml">Jumlah &nbsp; <b>Rp %s</b></div>
 <div class="ttd"><div class="box">Penerima<br><span class="sp"></span><u>%s</u></div>
 <div class="box">%s, ____________________<br>Kepala Sekolah</div></div>
 <div class="foot">Kwitansi ini sah dan dibuat otomatis oleh SchoolVia</div>
</div></body></html>`,
		e(nama), e(namaSekolah), e(alamat), itoa(id), e(bulan), e(nama), e(kelas), e(bulan), e(tahun), e(tglIndo(tglBayar)), e(metode), e(petugas),
		fnumFmt(jumlah), e(petugas), e(namaSekolah))
}

func fnumFmt(v float64) string {
	s := strconv.FormatFloat(v, 'f', 0, 64)
	// format ribuan Indonesia: 1.250.000
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += "."
		}
		out += string(c)
	}
	return out
}

// handleExportRekap: rekap bulanan dalam CSV (butuh plan export)
func handleExportRekap(w http.ResponseWriter, r *http.Request, c *Claims) {
	bulan := r.URL.Query().Get("bulan")
	if len(bulan) != 7 || bulan[4] != '-' {
		jsonErr(w, 400, "Format bulan harus YYYY-MM")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=rekap-"+bulan+".csv")
	w.Write([]byte("\xEF\xBB\xBF")) // BOM agar Excel baca UTF-8
	// Ringkasan
	var lunas, belum, totalLunas, pemasukan, pengeluaran int
	db.QueryRow(`SELECT COUNT(*) FILTER (WHERE sp.status='lunas'), COUNT(*) FILTER (WHERE sp.status!='lunas'),
			COALESCE(SUM(sp.jumlah) FILTER (WHERE sp.status='lunas'),0)
		FROM spp sp JOIN siswa s ON s.id=sp.siswa_id WHERE s.sekolah_id=$1 AND sp.bulan=$2`,
		c.SekolahID, bulanNumName(bulan)).Scan(&lunas, &belum, &totalLunas)
	db.QueryRow(`SELECT COALESCE(SUM(sp.jumlah),0) FROM spp sp JOIN siswa s ON s.id=sp.siswa_id
		WHERE s.sekolah_id=$1 AND sp.status='lunas' AND to_char(sp.tgl_bayar,'YYYY-MM')=$2`, c.SekolahID, bulan).Scan(&pemasukan)
	db.QueryRow(`SELECT COALESCE(SUM(jumlah),0) FROM pengeluaran WHERE sekolah_id=$1 AND to_char(tanggal,'YYYY-MM')=$2`, c.SekolahID, bulan).Scan(&pengeluaran)
	w.Write([]byte("REKAP BULANAN " + bulan + " — " + bulanID[atoiSafe(bulan[5:])-1] + "\n\n"))
	w.Write([]byte("Ringkasan,nilai\n"))
	w.Write([]byte("SPP lunas," + itoa(lunas) + "\n"))
	w.Write([]byte("SPP belum lunas," + itoa(belum) + "\n"))
	w.Write([]byte("Total SPP diterima," + itoa(totalLunas) + "\n"))
	w.Write([]byte("Pemasukan (dibayar bulan ini)," + itoa(pemasukan) + "\n"))
	w.Write([]byte("Pengeluaran," + itoa(pengeluaran) + "\n"))
	w.Write([]byte("Selisih kas," + itoa(pemasukan-pengeluaran) + "\n\n"))
	// Absensi per kelas
	w.Write([]byte("Absensi per kelas\nKelas,Hadir,Sakit,Izin,Alpa\n"))
	rows, _ := db.Query(`SELECT s.kelas,
			COUNT(*) FILTER (WHERE a.status='hadir'), COUNT(*) FILTER (WHERE a.status='sakit'),
			COUNT(*) FILTER (WHERE a.status='izin'), COUNT(*) FILTER (WHERE a.status='alpa')
		FROM absensi_siswa a JOIN siswa s ON s.id=a.siswa_id
		WHERE s.sekolah_id=$1 AND to_char(a.tanggal,'YYYY-MM')=$2 GROUP BY s.kelas ORDER BY s.kelas`, c.SekolahID, bulan)
	if rows != nil {
		for rows.Next() {
			var k string
			var h, sk, iz, al int
			rows.Scan(&k, &h, &sk, &iz, &al)
			w.Write([]byte(k + "," + itoa(h) + "," + itoa(sk) + "," + itoa(iz) + "," + itoa(al) + "\n"))
		}
		rows.Close()
	}
	// Rata-rata nilai
	w.Write([]byte("\nRata-rata nilai per kelas\nKelas,Rata-rata\n"))
	rows, _ = db.Query(`SELECT s.kelas, ROUND(AVG(n.nilai)::numeric,1)::float8 FROM nilai n JOIN siswa s ON s.id=n.siswa_id
		WHERE s.sekolah_id=$1 AND to_char(n.created_at,'YYYY-MM')=$2 GROUP BY s.kelas ORDER BY s.kelas`, c.SekolahID, bulan)
	if rows != nil {
		for rows.Next() {
			var k string
			var avg float64
			rows.Scan(&k, &avg)
			w.Write([]byte(k + "," + strconv.FormatFloat(avg, 'f', 1, 64) + "\n"))
		}
		rows.Close()
	}
	logActivity(c.UserID, "export_rekap", "Ekspor rekap "+bulan)
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(s)
	if n < 1 || n > 12 {
		return 1
	}
	return n
}

// planMWClaimsExport: adopsi Claims dari context lalu panggil handler Claims
func planMWClaimsExport(h func(http.ResponseWriter, *http.Request, *Claims)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := userFromContext(r)
		if c == nil {
			jsonErr(w, 401, "Unauthorized")
			return
		}
		h(w, r, c)
	}
}
