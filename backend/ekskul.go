package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ---------- P3: Ekstrakurikuler, kenaikan kelas & kelulusan, label iuran ----------

// EKSKUL: tabel ekskul (per sekolah) + anggota + penilaian deskriptif per siswa.
// Penilaian masuk rapor via kolom ekskul di rapor (dipakai handleCetakRapor).

func initEkskulSchema() {
	db.Exec(`CREATE TABLE IF NOT EXISTS ekskul (
		id SERIAL PRIMARY KEY,
		sekolah_id INT NOT NULL REFERENCES sekolah(id) ON DELETE CASCADE,
		nama TEXT NOT NULL,
		pembina_id INT REFERENCES users(id),
		UNIQUE(sekolah_id, nama)
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS ekskul_siswa (
		id SERIAL PRIMARY KEY,
		ekskul_id INT NOT NULL REFERENCES ekskul(id) ON DELETE CASCADE,
		siswa_id INT NOT NULL REFERENCES siswa(id) ON DELETE CASCADE,
		siswa_sekolah_id INT NOT NULL REFERENCES sekolah(id) ON DELETE CASCADE,
		nilai TEXT DEFAULT '', -- deskriptif: Sangat Baik / Baik / Cukup / Perlu Bimbingan
		catatan TEXT DEFAULT '',
		UNIQUE(ekskul_id, siswa_id)
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_ekskul_siswa_siswa ON ekskul_siswa(siswa_id)`)
	db.Exec(`GRANT ALL ON ekskul, ekskul_siswa TO simaka`)
	db.Exec(`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO simaka`)
}

// handleListEkskul: GET /api/ekskul → daftar ekskul + jumlah anggota
func handleListEkskul(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	rows, err := db.Query(`SELECT e.id, e.nama, COALESCE(u.nama,''), COUNT(es.id)
		FROM ekskul e LEFT JOIN users u ON u.id=e.pembina_id
		LEFT JOIN ekskul_siswa es ON es.ekskul_id=e.id
		WHERE e.sekolah_id=$1 GROUP BY e.id, e.nama, u.nama ORDER BY e.id`, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var nama, pembina string
		var n int
		rows.Scan(&id, &nama, &pembina, &n)
		list = append(list, map[string]interface{}{"id": id, "nama": nama, "pembina": pembina, "jumlah": n})
	}
	respond(w, 200, list)
}

// handleSaveEkskul: POST /api/ekskul {nama, pembina_id}
func handleSaveEkskul(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	var in struct {
		Nama      string `json:"nama"`
		PembinaID int    `json:"pembina_id"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Nama == "" {
		jsonErr(w, 400, "Nama ekskul wajib diisi")
		return
	}
	_, err := db.Exec(`INSERT INTO ekskul (sekolah_id, nama, pembina_id) VALUES ($1,$2,NULLIF($3,0)) ON CONFLICT (sekolah_id,nama) DO NOTHING`,
		c.SekolahID, in.Nama, in.PembinaID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 201, map[string]string{"ok": "Ekskul tersimpan"})
}

// handleDelEkskul: DELETE /api/ekskul/{id}
func handleDelEkskul(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	id, _ := strconv.Atoi(r.PathValue("id"))
	var sid int
	if err := db.QueryRow(`SELECT sekolah_id FROM ekskul WHERE id=$1`, id).Scan(&sid); err != nil || sid != c.SekolahID {
		jsonErr(w, 404, "Ekskul tidak ditemukan")
		return
	}
	db.Exec(`DELETE FROM ekskul WHERE id=$1`, id)
	respond(w, 200, map[string]string{"ok": "Ekskul dihapus"})
}

// handleEkskulAnggota: GET /api/ekskul/{id}/anggota
func handleEkskulAnggota(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	id, _ := strconv.Atoi(r.PathValue("id"))
	var sid int
	if err := db.QueryRow(`SELECT sekolah_id FROM ekskul WHERE id=$1`, id).Scan(&sid); err != nil || sid != c.SekolahID {
		jsonErr(w, 404, "Ekskul tidak ditemukan")
		return
	}
	rows, err := db.Query(`SELECT s.id, s.nama, s.kelas, COALESCE(es.nilai,''), COALESCE(es.catatan,'')
		FROM ekskul_siswa es JOIN siswa s ON s.id=es.siswa_id
		WHERE es.ekskul_id=$1 AND es.siswa_sekolah_id=$2 ORDER BY s.kelas, s.nama`, id, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var sid2 int
		var nama, kelas, nilai, catatan string
		rows.Scan(&sid2, &nama, &kelas, &nilai, &catatan)
		list = append(list, map[string]interface{}{"siswa_id": sid2, "nama": nama, "kelas": kelas, "nilai": nilai, "catatan": catatan})
	}
	respond(w, 200, list)
}

// handleAddEkskulSiswa: POST /api/ekskul/{id}/anggota {siswa_id, nilai, catatan}
func handleAddEkskulSiswa(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	id, _ := strconv.Atoi(r.PathValue("id"))
	var sid int
	if err := db.QueryRow(`SELECT sekolah_id FROM ekskul WHERE id=$1`, id).Scan(&sid); err != nil || sid != c.SekolahID {
		jsonErr(w, 404, "Ekskul tidak ditemukan")
		return
	}
	var in struct {
		SiswaID int    `json:"siswa_id"`
		Nilai   string `json:"nilai"`
		Catatan string `json:"catatan"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	var sSekolah int
	if err := db.QueryRow(`SELECT sekolah_id FROM siswa WHERE id=$1`, in.SiswaID).Scan(&sSekolah); err != nil || sSekolah != c.SekolahID {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return
	}
	_, err := db.Exec(`INSERT INTO ekskul_siswa (ekskul_id, siswa_id, siswa_sekolah_id, nilai, catatan)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (ekskul_id, siswa_id) DO UPDATE SET nilai=EXCLUDED.nilai, catatan=EXCLUDED.catatan`,
		id, in.SiswaID, c.SekolahID, in.Nilai, in.Catatan)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 200, map[string]string{"ok": "Penilaian ekskul tersimpan"})
}

// handleRaporEkskul: digabung ke /api/rapor/{id} — siswa: semua ekskul + nilai
func raporEkskulList(siswaID int) []map[string]interface{} {
	rows, err := db.Query(`SELECT e.nama, COALESCE(es.nilai,''), COALESCE(es.catatan,'')
		FROM ekskul_siswa es JOIN ekskul e ON e.id=es.ekskul_id
		WHERE es.siswa_id=$1 ORDER BY e.nama`, siswaID)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var nama, nilai, catatan string
		rows.Scan(&nama, &nilai, &catatan)
		list = append(list, map[string]interface{}{"nama": nama, "nilai": nilai, "catatan": catatan})
	}
	return list
}

// KENAIKAN KELAS & KELULUSAN: naikkan semua siswa aktif satu tingkat.
// handleNaikKelas: POST /api/naik-kelas {dari:"7", ke:"8"} — kelas berformat "7A"→"8A"
func handleNaikKelas(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	var in struct {
		Dari string `json:"dari"`
		Ke   string `json:"ke"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Dari == "" || in.Ke == "" {
		jsonErr(w, 400, "Tingkat asal dan tingkat tujuan wajib diisi (mis. 7 → 8)")
		return
	}
	if !isDigits(in.Dari) || !isDigits(in.Ke) {
		jsonErr(w, 400, "Tingkat asal dan tujuan harus berupa angka (mis. 7 dan 8)")
		return
	}
	// update kelas: ganti prefix tingkat, sisakan huruf paralel
	tag, err := db.Exec(`UPDATE siswa SET kelas = $2 || SUBSTRING(kelas FROM LENGTH($1)+1)
		WHERE sekolah_id=$3 AND status='aktif' AND kelas LIKE $1 || '%'`,
		in.Dari, in.Ke, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	n, _ := tag.RowsAffected()
	respond(w, 200, map[string]interface{}{"ok": "Kenaikan kelas selesai", "jumlah": n})
}

// handleLuluskan: POST /api/luluskan {tingkat:"6"} — arsipkan siswa kelas akhir sebagai lulus
func handleLuluskan(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	var in struct {
		Tingkat string `json:"tingkat"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Tingkat == "" {
		jsonErr(w, 400, "Tingkat kelas akhir wajib diisi (mis. 6 untuk SD)")
		return
	}
	if !isDigits(in.Tingkat) {
		jsonErr(w, 400, "Tingkat kelas akhir harus berupa angka (mis. 6)")
		return
	}
	tag, err := db.Exec(`UPDATE siswa SET status='lulus'
		WHERE sekolah_id=$1 AND status='aktif' AND kelas LIKE $2 || '%'`,
		c.SekolahID, in.Tingkat)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	n, _ := tag.RowsAffected()
	respond(w, 200, map[string]interface{}{"ok": "Kelulusan tercatat", "jumlah": n})
}
