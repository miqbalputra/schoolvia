package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ---------- CSV EXPORT ----------
// CSV dipisah ";" dan pakai BOM agar langsung rapi dibuka di Excel Indonesia.

func csvResponse(w http.ResponseWriter, name string, header []string, rows [][]string) {
	var b strings.Builder
	b.WriteString("\uFEFF") // UTF-8 BOM for Excel
	b.WriteString(strings.Join(header, ";"))
	b.WriteString("\r\n")
	for _, r := range rows {
		quoted := make([]string, len(r))
		for i, c := range r {
			quoted[i] = `"` + strings.ReplaceAll(c, `"`, `""`) + `"`
		}
		b.WriteString(strings.Join(quoted, ";"))
		b.WriteString("\r\n")
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Write([]byte(b.String()))
}

func handleExportAbsensi(w http.ResponseWriter, r *http.Request) {
	bulan := r.URL.Query().Get("bulan") // "2026-11" atau kosong = bulan ini
	where := `WHERE s.sekolah_id=$2 AND ($1 = '' OR to_char(a.tanggal,'YYYY-MM') = $1)`
	rows, err := db.Query(`SELECT s.nis, s.nama, s.kelas, a.tanggal::text, a.status, COALESCE(a.keterangan,'')
		FROM absensi_siswa a JOIN siswa s ON s.id=a.siswa_id `+where+` ORDER BY a.tanggal, s.kelas, s.nama`, bulan, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	data := [][]string{}
	for rows.Next() {
		var nis, nama, kelas, tgl, st, ket string
		rows.Scan(&nis, &nama, &kelas, &tgl, &st, &ket)
		data = append(data, []string{nis, nama, kelas, tgl, st, ket})
	}
	csvResponse(w, "rekap-absensi.csv", []string{"NIS", "Nama", "Kelas", "Tanggal", "Status", "Keterangan"}, data)
}

func handleExportSPP(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT s.nis, s.nama, s.kelas, sp.bulan, sp.tahun, sp.jumlah::text, sp.status, COALESCE(sp.tgl_bayar::text,'-'), COALESCE(sp.metode,'-')
		FROM spp sp JOIN siswa s ON s.id=sp.siswa_id WHERE s.sekolah_id=$1 ORDER BY sp.tahun, sp.bulan, s.kelas, s.nama`, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	data := [][]string{}
	for rows.Next() {
		var nis, nama, kelas, bulan, tahun, jumlah, st, tgl, metode string
		rows.Scan(&nis, &nama, &kelas, &bulan, &tahun, &jumlah, &st, &tgl, &metode)
		data = append(data, []string{nis, nama, kelas, bulan, tahun, jumlah, st, tgl, metode})
	}
	csvResponse(w, "rekap-spp.csv", []string{"NIS", "Nama", "Kelas", "Bulan", "Tahun", "Jumlah", "Status", "Tgl Bayar", "Metode"}, data)
}

func handleExportNilai(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT s.nis, s.nama, n.kelas, n.mapel, n.jenis, n.semester, n.tahun_ajaran, n.nilai::text
		FROM nilai n JOIN siswa s ON s.id=n.siswa_id WHERE s.sekolah_id=$1 ORDER BY n.kelas, s.nama, n.mapel`, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	data := [][]string{}
	for rows.Next() {
		var nis, nama, kelas, mapel, jenis, sem, ta, nilai string
		rows.Scan(&nis, &nama, &kelas, &mapel, &jenis, &sem, &ta, &nilai)
		data = append(data, []string{nis, nama, kelas, mapel, strings.ToUpper(jenis), sem, ta, nilai})
	}
	csvResponse(w, "rekap-nilai.csv", []string{"NIS", "Nama", "Kelas", "Mapel", "Jenis", "Semester", "Tahun Ajaran", "Nilai"}, data)
}

func handleExportKeuangan(w http.ResponseWriter, r *http.Request) {
	data := [][]string{}
	// Pemasukan
	rows, err := db.Query(`SELECT 'PEMASUKAN', s.nama || ' - SPP ' || sp.bulan || ' ' || sp.tahun, COALESCE(sp.metode,''), sp.tgl_bayar::text, sp.jumlah::text
		FROM spp sp JOIN siswa s ON s.id=sp.siswa_id WHERE sp.status='lunas' AND s.sekolah_id=$1 ORDER BY sp.tgl_bayar`, userFromContext(r).SekolahID)
	if err == nil {
		for rows.Next() {
			var j, ket, met, tgl, jml string
			rows.Scan(&j, &ket, &met, &tgl, &jml)
			data = append(data, []string{j, ket, met, tgl, jml})
		}
		rows.Close()
	}
	rows2, err := db.Query(`SELECT 'PENGELUARAN', uraian, COALESCE(kategori,''), tanggal::text, jumlah::text FROM pengeluaran WHERE sekolah_id=$1 ORDER BY tanggal`, userFromContext(r).SekolahID)
	if err == nil {
		for rows2.Next() {
			var j, ket, met, tgl, jml string
			rows2.Scan(&j, &ket, &met, &tgl, &jml)
			data = append(data, []string{j, ket, met, tgl, jml})
		}
		rows2.Close()
	}
	csvResponse(w, "buku-kas.csv", []string{"Jenis", "Keterangan", "Kategori/Metode", "Tanggal", "Jumlah (Rp)"}, data)
}

// ---------- RAPOR ----------

func handleRaporData(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	if c == nil {
		jsonErr(w, 401, "Unauthorized")
		return
	}
	sid, _ := strconv.Atoi(r.PathValue("id"))
	var siswaSekolah int
	if err := db.QueryRow(`SELECT sekolah_id FROM siswa WHERE id=$1`, sid).Scan(&siswaSekolah); err != nil || siswaSekolah != c.SekolahID {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return
	}
	ta := r.URL.Query().Get("tahun_ajaran")
	if ta == "" {
		ta = tahunAjaranSekolah(c.SekolahID)
	}
	var nama, nis, kelas, ortu, hp string
	err := db.QueryRow(`SELECT nama, COALESCE(nis,''), kelas, COALESCE(nama_ortu,''), COALESCE(hp_ortu,'') FROM siswa WHERE id=$1`, sid).
		Scan(&nama, &nis, &kelas, &ortu, &hp)
	if err != nil {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return
	}
	// Nilai per mapel: rata-rata tugas/uh, plus pts/pas
	type mrow struct {
		Mapel string
		Tugas, UH, PTS, PAS *float64
	}
	mmap := map[string]*mrow{}
	rows, err := db.Query(`SELECT mapel, jenis, AVG(nilai) FROM nilai WHERE siswa_id=$1 AND tahun_ajaran=$2 GROUP BY mapel, jenis`, sid, ta)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	for rows.Next() {
		var mapel, jenis string
		var av float64
		rows.Scan(&mapel, &jenis, &av)
		m, ok := mmap[mapel]
		if !ok {
			m = &mrow{Mapel: mapel}
			mmap[mapel] = m
		}
		p := &av
		switch jenis {
		case "tugas":
			m.Tugas = p
		case "uh":
			m.UH = p
		case "pts":
			m.PTS = p
		case "pas":
			m.PAS = p
		}
	}
	rows.Close()
	// Absensi rekap
	var hadir, sakit, izin, alpa int
	db.QueryRow(`SELECT COUNT(*) FILTER (WHERE status='hadir'), COUNT(*) FILTER (WHERE status='sakit'),
		COUNT(*) FILTER (WHERE status='izin'), COUNT(*) FILTER (WHERE status='alpa')
		FROM absensi_siswa WHERE siswa_id=$1`, sid).Scan(&hadir, &sakit, &izin, &alpa)
	// Hafalan
	hafalan := []map[string]interface{}{}
	r3, err := db.Query(`SELECT surah, ayat_dari, ayat_sampai, status, tanggal::text FROM hafalan WHERE siswa_id=$1 ORDER BY tanggal DESC LIMIT 50`, sid)
	if err == nil {
		for r3.Next() {
			var surah, st, tgl string
			var d, s int
			r3.Scan(&surah, &d, &s, &st, &tgl)
			hafalan = append(hafalan, map[string]interface{}{"surah": surah, "dari": d, "sampai": s, "status": st, "tanggal": tgl})
		}
		r3.Close()
	}
	// SPP rekap
	var sppLunas, sppBelum int
	db.QueryRow(`SELECT COUNT(*) FILTER (WHERE status='lunas'), COUNT(*) FILTER (WHERE status='belum') FROM spp WHERE siswa_id=$1`, sid).Scan(&sppLunas, &sppBelum)

	list := []map[string]interface{}{}
	var total float64
	n := 0
	for _, m := range mmap {
		var na float64
		cnt := 0
		sum := 0.0
		for _, p := range []*float64{m.Tugas, m.UH, m.PTS, m.PAS} {
			if p != nil {
				sum += *p
				cnt++
			}
		}
		if cnt > 0 {
			na = sum / float64(cnt)
			total += na
			n++
		}
		pred, ket := predikatMerdeka(na)
		list = append(list, map[string]interface{}{
			"mapel": m.Mapel, "tugas": m.Tugas, "uh": m.UH, "pts": m.PTS, "pas": m.PAS, "na": na,
			"predikat": pred, "ket": ket,
		})
	}
	var rataRata float64
	if n > 0 {
		rataRata = total / float64(n)
	}
	respond(w, 200, map[string]interface{}{
		"siswa": map[string]interface{}{"nama": nama, "nis": nis, "kelas": kelas, "nama_ortu": ortu, "hp_ortu": hp},
		"tahun_ajaran": ta, "nilai": list, "rata_rata": rataRata,
		"absensi": map[string]int{"hadir": hadir, "sakit": sakit, "izin": izin, "alpa": alpa},
		"hafalan": hafalan, "ekskul": raporEkskulList(sid), "spp_lunas": sppLunas, "spp_belum": sppBelum,
	})
}

func handleCetakRapor(w http.ResponseWriter, r *http.Request) {
	// Token boleh lewat query (?token=) karena dibuka sebagai tab cetak baru
	if userFromContext(r) == nil {
		tokStr := r.URL.Query().Get("token")
		if tokStr != "" {
			if tok, err := jwt.ParseWithClaims(tokStr, &Claims{}, func(t *jwt.Token) (interface{}, error) { return jwtSecret, nil }); err == nil && tok.Valid {
				r = r.WithContext(context.WithValue(r.Context(), userKey, tok.Claims.(*Claims)))
			}
		}
	}
	c := userFromContext(r)
	if c == nil {
		http.Error(w, "Unauthorized", 401)
		return
	}
	sid, _ := strconv.Atoi(r.PathValue("id"))
	// cek tenant: siswa harus milik sekolah user
	var siswaSekolah int
	if err := db.QueryRow(`SELECT sekolah_id FROM siswa WHERE id=$1`, sid).Scan(&siswaSekolah); err != nil || siswaSekolah != c.SekolahID {
		http.Error(w, "Rapor tidak ditemukan", 404)
		return
	}
	ta := r.URL.Query().Get("tahun_ajaran")
	if ta == "" {
		ta = tahunAjaranSekolah(c.SekolahID)
	}
	var nama, nis, kelas, ortu string
	db.QueryRow(`SELECT nama, COALESCE(nis,''), kelas, COALESCE(nama_ortu,'') FROM siswa WHERE id=$1`, sid).Scan(&nama, &nis, &kelas, &ortu)
	var namaSekolah, alamatSekolah, kepalaSekolah string
	db.QueryRow(`SELECT nama_sekolah, COALESCE(alamat,''), COALESCE(kepala_sekolah,'') FROM pengaturan WHERE id=$1`, c.SekolahID).Scan(&namaSekolah, &alamatSekolah, &kepalaSekolah)
	cw, ek, pr := getCatatan(sid, ta)
	rows, err := db.Query(`SELECT mapel, jenis, AVG(nilai) FROM nilai WHERE siswa_id=$1 AND tahun_ajaran=$2 GROUP BY mapel, jenis`, sid, ta)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	type mrow struct{ t, u, p, pas *float64 }
	mm := map[string]*mrow{}
	var order []string
	for rows.Next() {
		var mapel, jenis string
		var av float64
		rows.Scan(&mapel, &jenis, &av)
		m, ok := mm[mapel]
		if !ok {
			m = &mrow{}
			mm[mapel] = m
			order = append(order, mapel)
		}
		av2 := av
		switch jenis {
		case "tugas":
			m.t = &av2
		case "uh":
			m.u = &av2
		case "pts":
			m.p = &av2
		case "pas":
			m.pas = &av2
		}
	}
	rows.Close()
	f := func(p *float64) string {
		if p == nil {
			return "-"
		}
		return strconv.FormatFloat(*p, 'f', 1, 64)
	}
	var b strings.Builder
	total, n := 0.0, 0
	for _, mapel := range order {
		m := mm[mapel]
		na, cnt, sum := 0.0, 0, 0.0
		for _, p := range []*float64{m.t, m.u, m.p, m.pas} {
			if p != nil {
				sum += *p
				cnt++
			}
		}
		if cnt > 0 {
			na = sum / float64(cnt)
			total += na
			n++
		}
		predRow, _ := predikatMerdeka(na)
		fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td><b>%s</b></td><td style="text-align:center"><b>%s</b></td></tr>`,
			html.EscapeString(mapel), f(m.t), f(m.u), f(m.p), f(m.pas), strconv.FormatFloat(na, 'f', 1, 64), predRow)
	}
	rata := 0.0
	if n > 0 {
		rata = total / float64(n)
	}
	predikat := "C"
	ket := "Cukup"
	if rata >= 90 {
		predikat, ket = "A", "Sangat Baik"
	} else if rata >= 80 {
		predikat, ket = "B", "Baik"
	} else if rata >= 70 {
		predikat, ket = "C", "Cukup"
	} else if rata > 0 {
		predikat, ket = "D", "Perlu Bimbingan"
	}
	var hadir, sakit, izin, alpa int
	db.QueryRow(`SELECT COUNT(*) FILTER (WHERE status='hadir'), COUNT(*) FILTER (WHERE status='sakit'),
		COUNT(*) FILTER (WHERE status='izin'), COUNT(*) FILTER (WHERE status='alpa')
		FROM absensi_siswa WHERE siswa_id=$1`, sid).Scan(&hadir, &sakit, &izin, &alpa)
	// Ekskul rows untuk cetak
	var eb strings.Builder
	for _, e := range raporEkskulList(sid) {
		nama, _ := e["nama"].(string)
		nilai, _ := e["nilai"].(string)
		if nilai == "" {
			nilai = "-"
		}
		fmt.Fprintf(&eb, `<tr><td width="50%%">%s</td><td>%s</td></tr>`, html.EscapeString(nama), html.EscapeString(nilai))
	}
	if eb.Len() == 0 {
		eb.WriteString(`<tr><td colspan="2">-</td></tr>`)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="id"><head><meta charset="utf-8"><title>Rapor %s</title>
<style>
body{font-family:Georgia,'Times New Roman',serif;max-width:800px;margin:24px auto;color:#111;line-height:1.5;padding:0 16px}
h1{font-size:20px;text-align:center;margin:0} h2{font-size:14px;text-align:center;font-weight:normal;margin:2px 0 20px}
table{width:100%%;border-collapse:collapse;font-size:13px;margin:10px 0}
th,td{border:1px solid #333;padding:6px 8px;text-align:left} th{background:#e8f5f4}
.right{text-align:right} .no-print{margin-bottom:18px}
.no-print button{padding:8px 16px;border:none;border-radius:8px;background:#0f766e;color:#fff;font-size:14px;cursor:pointer;margin-right:8px}
.no-print button.sec{background:#e2e8f0;color:#111}
.ket{display:flex;gap:40px;font-size:13px;margin:14px 0}
.ttd{display:flex;justify-content:space-between;margin-top:48px;font-size:13px}
@media print{.no-print{display:none} body{margin:0}}
</style></head><body>
<div class="no-print">
  <button onclick="window.print()">🖨️ Cetak / Simpan PDF</button>
  <button class="sec" onclick="window.close()">Tutup</button>
</div>
<h1>LAPORAN HASIL BELAJAR (RAPOR)</h1>
<h2 style="font-size:16px;font-weight:bold;margin-bottom:2px">%s</h2>
<h2>%s — Tahun Ajaran %s — Kelas %s</h2>
<table><tr><td width="50%%">Nama: <b>%s</b></td><td>NIS: <b>%s</b></td></tr>
<tr><td>Kelas: <b>%s</b></td><td>Orang Tua: <b>%s</b></td></tr></table>
<h3 style="font-size:14px">A. Nilai Akademik</h3>
<table><tr><th>Mata Pelajaran</th><th>Tugas</th><th>UH</th><th>PTS</th><th>PAS</th><th>NA</th><th>Predikat</th></tr>%s</table>
<p style="font-size:13px">Rata-rata: <b>%s</b> &nbsp;|&nbsp; Predikat: <b>%s</b> (%s)</p>
<h3 style="font-size:14px">B. Kehadiran</h3>
<table><tr><th>Hadir</th><th>Sakit</th><th>Izin</th><th>Alpa</th></tr>
<tr><td>%d hari</td><td>%d hari</td><td>%d hari</td><td>%d hari</td></tr></table>
<h3 style="font-size:14px">C. Ekstrakurikuler</h3>
<table>%s</table>
<h3 style="font-size:14px">D. Catatan Wali Kelas</h3>
<table>
<tr><td width="30%%">Ekstrakurikuler</td><td>%s</td></tr>
<tr><td>Prestasi</td><td>%s</td></tr>
<tr><td>Catatan Wali</td><td>%s</td></tr>
</table>
<div class="ttd">
  <div>Orang Tua/Wali<br><br><br>(%s)</div>
  <div>Kepala Sekolah<br><br><br><b>%s</b></div>
</div>
</body></html>`,
		html.EscapeString(nama), html.EscapeString(namaSekolah), html.EscapeString(alamatSekolah), html.EscapeString(ta), html.EscapeString(kelas),
		html.EscapeString(nama), html.EscapeString(nis),
		html.EscapeString(kelas), html.EscapeString(ortu), b.String(),
		strconv.FormatFloat(rata, 'f', 1, 64), predikat, ket,
		hadir, sakit, izin, alpa,
		eb.String(),
		html.EscapeString(ek), html.EscapeString(pr), html.EscapeString(cw),
		html.EscapeString(ortu), html.EscapeString(kepalaSekolah))
}

// ---------- NILAI MASSAL ----------

func handleBulkNilai(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		Kelas       string  `json:"kelas"`
		Mapel       string  `json:"mapel"`
		Jenis       string  `json:"jenis"`
		Semester    string  `json:"semester"`
		TahunAjaran string  `json:"tahun_ajaran"`
		Items       []struct {
			SiswaID int     `json:"siswa_id"`
			Nilai   float64 `json:"nilai"`
		} `json:"items"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Kelas == "" || in.Mapel == "" || in.Jenis == "" || len(in.Items) == 0 {
		jsonErr(w, 400, "Kelas, mapel, jenis, dan nilai wajib diisi")
		return
	}
	if in.TahunAjaran == "" {
		in.TahunAjaran = tahunAjaranSekolah(c.SekolahID)
	}
	n, skip := 0, 0
	for _, it := range in.Items {
		if it.Nilai <= 0 || it.Nilai > 100 {
			skip++
			continue
		}
		// pastikan siswa milik sekolah yang sama
		var sid int
		if err := db.QueryRow(`SELECT sekolah_id FROM siswa WHERE id=$1`, it.SiswaID).Scan(&sid); err != nil || sid != c.SekolahID {
			skip++
			continue
		}
		_, err := db.Exec(`INSERT INTO nilai (siswa_id,mapel,kelas,jenis,semester,tahun_ajaran,nilai,guru_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (siswa_id,mapel,jenis,semester,tahun_ajaran) DO UPDATE SET nilai=EXCLUDED.nilai, guru_id=EXCLUDED.guru_id`,
			it.SiswaID, in.Mapel, in.Kelas, in.Jenis, in.Semester, in.TahunAjaran, it.Nilai, c.UserID)
		if err == nil {
			n++
		} else {
			skip++
		}
	}
	respond(w, 200, map[string]interface{}{"saved": n, "skipped": skip})
}

func handleSiswaByKelas(w http.ResponseWriter, r *http.Request) {
	kelas := r.URL.Query().Get("kelas")
	rows, err := db.Query(`SELECT id, nama, COALESCE(nis,'') FROM siswa WHERE status='aktif' AND kelas=$1 AND sekolah_id=$2 ORDER BY nama`, kelas, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var nama, nis string
		rows.Scan(&id, &nama, &nis)
		list = append(list, map[string]interface{}{"id": id, "nama": nama, "nis": nis})
	}
	respond(w, 200, list)
}

// handleExportSiswa: CSV daftar siswa (anti-sandera — bebas semua plan)
func handleExportSiswa(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT nis, nama, jk, kelas, COALESCE(nama_ortu,''), COALESCE(hp_ortu,''), status
		FROM siswa WHERE sekolah_id=$1 ORDER BY kelas, nama`, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	data := [][]string{}
	for rows.Next() {
		var nis, nama, jk, kelas, ortu, hp, st string
		rows.Scan(&nis, &nama, &jk, &kelas, &ortu, &hp, &st)
		data = append(data, []string{nis, nama, jk, kelas, ortu, hp, st})
	}
	csvResponse(w, "siswa.csv", []string{"NIS", "Nama", "JK", "Kelas", "Nama Ortu", "HP Ortu", "Status"}, data)
}

// handleExportHafalan: CSV riwayat hafalan (anti-sandera — bebas semua plan)
func handleExportHafalan(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT s.nis, s.nama, s.kelas, h.surah, COALESCE(h.ayat_dari::text,''), COALESCE(h.ayat_sampai::text,''), h.status, COALESCE(h.catatan,''), h.tanggal::text
		FROM hafalan h JOIN siswa s ON s.id=h.siswa_id WHERE s.sekolah_id=$1 ORDER BY h.tanggal DESC`, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	data := [][]string{}
	for rows.Next() {
		var nis, nama, kelas, surah, ad, as2, st, ket, tgl string
		rows.Scan(&nis, &nama, &kelas, &surah, &ad, &as2, &st, &ket, &tgl)
		data = append(data, []string{nis, nama, kelas, surah, ad, as2, st, ket, tgl})
	}
	csvResponse(w, "hafalan.csv", []string{"NIS", "Nama", "Kelas", "Surah", "Ayat Dari", "Ayat Sampai", "Status", "Catatan", "Tanggal"}, data)
}
