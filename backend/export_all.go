package main

// Anti-sandera data: satu klik → ZIP berisi CSV semua tabel milik sekolah.
// Terbuka untuk SEMUA plan, termasuk gratis. Ekspor = jaminan, bukan fitur premium.

import (
	"archive/zip"
	"fmt"
	"database/sql"
	"net/http"
	"strings"
	"time"
)

func csvRow(rows *sql.Rows, cols []string) ([]string, error) {
	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	out := make([]string, len(cols))
	for i, v := range vals {
		if v == nil {
			out[i] = ""
		} else {
			switch t := v.(type) {
			case []byte:
				out[i] = string(t)
			default:
				out[i] = fmt.Sprintf("%v", t)
			}
		}
	}
	return out, nil
}

func csvEscape(s string) string {
	s = csvSafe(s)
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// handleExportAll: ZIP berisi CSV semua data sekolah (anti-sandera)
func handleExportAll(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	if c == nil {
		jsonErr(w, 401, "Unauthorized")
		return
	}
	sid := c.SekolahID

	// tabel + kolom yang diekspor (semua terfilter sekolah_id)
	tables := []struct {
		nama  string
		query string
	}{
		{"siswa", `SELECT id, nis, nama, jk, kelas, nama_ortu, hp_ortu, status FROM siswa WHERE sekolah_id=$1 ORDER BY id`},
		{"nilai", `SELECT n.id, s.nama AS siswa, n.mapel, n.jenis, n.nilai, n.tahun_ajaran, n.semester FROM nilai n JOIN siswa s ON s.id=n.siswa_id WHERE n.sekolah_id=$1 ORDER BY n.id`},
		{"hafalan", `SELECT h.id, s.nama AS siswa, h.surah, h.ayat_dari, h.ayat_sampai, h.status, h.catatan, to_char(h.tanggal,'YYYY-MM-DD') FROM hafalan h JOIN siswa s ON s.id=h.siswa_id WHERE h.sekolah_id=$1 ORDER BY h.id`},
		{"absensi", `SELECT a.id, s.nama AS siswa, a.tanggal, a.status, COALESCE(a.keterangan,'') FROM absensi_siswa a JOIN siswa s ON s.id=a.siswa_id WHERE a.sekolah_id=$1 ORDER BY a.id`},
		{"spp", `SELECT sp.id, s.nama AS siswa, sp.bulan, sp.tahun, sp.jumlah, sp.status, COALESCE(to_char(sp.tgl_bayar,'YYYY-MM-DD'),'-') FROM spp sp JOIN siswa s ON s.id=sp.siswa_id WHERE sp.sekolah_id=$1 ORDER BY sp.id`},
		{"pengeluaran", `SELECT id, uraian, kategori, jumlah, to_char(tanggal,'YYYY-MM-DD') FROM pengeluaran WHERE sekolah_id=$1 ORDER BY id`},
		{"users", `SELECT id, nama, email, role FROM users WHERE sekolah_id=$1 ORDER BY id`},
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"simaka-data-sekolah-"+itoa(sid)+".zip\"")

	zw := zip.NewWriter(w)
	defer zw.Close()

	var gagal []string
	for _, t := range tables {
		rows, err := db.Query(t.query, sid)
		if err != nil {
			gagal = append(gagal, t.nama+": "+err.Error())
			continue
		}
		cols, _ := rows.Columns()
		var sb strings.Builder
		// header
		for i, cl := range cols {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(csvEscape(cl))
		}
		sb.WriteString("\n")
		n := 0
		for rows.Next() {
			row, err := csvRow(rows, cols)
			if err != nil {
				break
			}
			for i, v := range row {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(csvEscape(v))
			}
			sb.WriteString("\n")
			n++
		}
		rows.Close()
		fw, err := zw.Create(t.nama + ".csv")
		if err != nil {
			continue
		}
		fw.Write([]byte(sb.String()))
		_ = n
	}
	if len(gagal) > 0 {
		fe, _ := zw.Create("ERROR.txt")
		fe.Write([]byte("PERHATIAN: beberapa tabel gagal diekspor (seharusnya tidak terjadi setelah migrasi):\n" + strings.Join(gagal, "\n") + "\n"))
	}
	// README kecil di dalam ZIP
	rd, _ := zw.Create("BACA-SAYA.txt")
	rd.Write([]byte("Data ini milik sekolah Anda. Bisa dibuka dengan Excel/Google Sheets.\nDiunduh dari SchoolVia pada " + time.Now().Format("2 Jan 2006") + ".\n"))
}
