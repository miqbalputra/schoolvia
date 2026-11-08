package main

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

// ---------- PENGATURAN SEKOLAH ----------

func handleGetPengaturan(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	if c == nil {
		jsonErr(w, 401, "Unauthorized")
		return
	}
	var nama, alamat, npsn, kepala, ta, sem, waPeng string
	err := db.QueryRow(`SELECT nama_sekolah, COALESCE(alamat,''), COALESCE(npsn,''), COALESCE(kepala_sekolah,''), tahun_ajaran_aktif, semester_aktif, COALESCE(wa_pengasuh,'') FROM pengaturan WHERE id=$1`, c.SekolahID).
		Scan(&nama, &alamat, &npsn, &kepala, &ta, &sem, &waPeng)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 200, map[string]string{
		"nama_sekolah": nama, "alamat": alamat, "npsn": npsn, "kepala_sekolah": kepala,
		"tahun_ajaran_aktif": ta, "semester_aktif": sem, "wa_pengasuh": waPeng,
	})
}

func handleUpdatePengaturan(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in map[string]string
	json.NewDecoder(r.Body).Decode(&in)
	_, err := db.Exec(`UPDATE pengaturan SET nama_sekolah=$1, alamat=$2, npsn=$3, kepala_sekolah=$4, tahun_ajaran_aktif=$5, semester_aktif=$6 WHERE id=$7`,
		in["nama_sekolah"], in["alamat"], in["npsn"], in["kepala_sekolah"], in["tahun_ajaran_aktif"], in["semester_aktif"], c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	logActivity(c.UserID, "update_pengaturan", "Memperbarui pengaturan sekolah")
	respond(w, 200, map[string]string{"ok": "1"})
}

// ---------- RESET PASSWORD ----------

func randPassword() string {
	const chars = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	n := big.NewInt(int64(len(chars)))
	b := make([]byte, 10)
	for i := range b {
		x, _ := rand.Int(rand.Reader, n)
		b[i] = chars[x.Int64()]
	}
	return string(b)
}

func handleResetPassword(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	// hanya boleh reset user dari sekolah yang sama
	var sid int
	var role string
	if err := db.QueryRow(`SELECT COALESCE(sekolah_id,0), role FROM users WHERE id=$1`, id).Scan(&sid, &role); err != nil || sid != c.SekolahID {
		jsonErr(w, 404, "Pengguna tidak ditemukan")
		return
	}
	// Kepala sekolah tidak boleh di-reset oleh TU
	if role == "kepala_sekolah" && c.Role != "kepala_sekolah" {
		jsonErr(w, 403, "Hanya kepala sekolah yang boleh me-reset akun kepala sekolah")
		return
	}
	// Tidak boleh me-reset kepala sekolah lain
	if role == "kepala_sekolah" && id != c.UserID {
		jsonErr(w, 403, "Tidak boleh me-reset kepala sekolah lain")
		return
	}
	newPass := randPassword()
	hash, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	res, err := db.Exec(`UPDATE users SET password=$1, token_version=token_version+1 WHERE id=$2`, string(hash), id)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, 404, "Pengguna tidak ditemukan")
		return
	}
	var nama string
	db.QueryRow(`SELECT nama FROM users WHERE id=$1`, id).Scan(&nama)
	logActivity(c.UserID, "reset_password", "Reset password untuk user ID "+itoa(id)+" ("+nama+")")
	respond(w, 200, map[string]string{"password_baru": newPass})
}

func handleUbahPassword(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		Lama, Baru string
	}
	json.NewDecoder(r.Body).Decode(&in)
	if len(in.Baru) < 8 {
		jsonErr(w, 400, "Password baru minimal 8 karakter")
		return
	}
	if c.UserID == 0 {
		jsonErr(w, 401, "Tidak terautentikasi")
		return
	}
	var hash string
	if err := db.QueryRow(`SELECT password FROM users WHERE id=$1`, c.UserID).Scan(&hash); err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	if !checkPass(hash, in.Lama) {
		jsonErr(w, 400, "Password lama salah")
		return
	}
	newHash, _ := bcrypt.GenerateFromPassword([]byte(in.Baru), bcrypt.DefaultCost)
	db.Exec(`UPDATE users SET password=$1, token_version=token_version+1 WHERE id=$2`, string(newHash), c.UserID)
	logActivity(c.UserID, "ubah_password", "Mengubah password sendiri")
	respond(w, 200, map[string]string{"ok": "1"})
}

// ---------- CATATAN RAPOR ----------

func getCatatan(sid int, ta string) (cw, ek, pr string) {
	db.QueryRow(`SELECT COALESCE(catatan_wali,''), COALESCE(ekstrakurikuler,''), COALESCE(prestasi,'')
		FROM rapor_catatan WHERE siswa_id=$1 AND tahun_ajaran=$2`, sid, ta).Scan(&cw, &ek, &pr)
	return
}

func cekSiswaTenant(w http.ResponseWriter, sid int, sekolahID int) bool {
	var sSekolah int
	if err := db.QueryRow(`SELECT sekolah_id FROM siswa WHERE id=$1`, sid).Scan(&sSekolah); err != nil || sSekolah != sekolahID {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return false
	}
	return true
}

func handleGetCatatanRapor(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	if c == nil {
		jsonErr(w, 401, "Unauthorized")
		return
	}
	sid, _ := strconv.Atoi(r.PathValue("id"))
	if !cekSiswaTenant(w, sid, c.SekolahID) {
		return
	}
	ta := r.URL.Query().Get("tahun_ajaran")
	if ta == "" {
		ta = tahunAjaranSekolah(c.SekolahID)
	}
	cw, ek, pr := getCatatan(sid, ta)
	respond(w, 200, map[string]string{"catatan_wali": cw, "ekstrakurikuler": ek, "prestasi": pr})
}

func handleSaveCatatanRapor(w http.ResponseWriter, r *http.Request, c *Claims) {
	sid, _ := strconv.Atoi(r.PathValue("id"))
	if !cekSiswaTenant(w, sid, c.SekolahID) {
		return
	}
	var in struct {
		TahunAjaran    string `json:"tahun_ajaran"`
		CatatanWali    string `json:"catatan_wali"`
		Ekstrakurikuler string `json:"ekstrakurikuler"`
		Prestasi       string `json:"prestasi"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.TahunAjaran == "" {
		in.TahunAjaran = tahunAjaranSekolah(c.SekolahID)
	}
	_, err := db.Exec(`INSERT INTO rapor_catatan (siswa_id, tahun_ajaran, catatan_wali, ekstrakurikuler, prestasi, wali_id)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (siswa_id, tahun_ajaran) DO UPDATE SET catatan_wali=EXCLUDED.catatan_wali, ekstrakurikuler=EXCLUDED.ekstrakurikuler, prestasi=EXCLUDED.prestasi, wali_id=EXCLUDED.wali_id`,
		sid, in.TahunAjaran, in.CatatanWali, in.Ekstrakurikuler, in.Prestasi, c.UserID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	logActivity(c.UserID, "update_rapor", "Menyimpan catatan rapor siswa ID "+itoa(sid))
	respond(w, 200, map[string]string{"ok": "1"})
}

// ---------- ACTIVITY LOG ----------

func handleListLog(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT l.id, COALESCE(u.nama,'-'), l.aksi, COALESCE(l.detail,''), l.waktu::text
		FROM activity_log l LEFT JOIN users u ON u.id=l.user_id WHERE l.sekolah_id=$1 ORDER BY l.waktu DESC, l.id DESC LIMIT 200`, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var nama, aksi, detail, waktu string
		rows.Scan(&id, &nama, &aksi, &detail, &waktu)
		list = append(list, map[string]interface{}{"id": id, "user": nama, "aksi": aksi, "detail": detail, "waktu": waktu})
	}
	respond(w, 200, list)
}

// ---------- DELETE (Fase B: CRUD lengkap dengan cek tenant) ----------

func deleteOwned(w http.ResponseWriter, table string, id int, sekolahID int, col string) bool {
	res, err := db.Exec(`DELETE FROM `+table+` WHERE id=$1 AND `+col+`=$2`, id, sekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return false
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, 404, "Data tidak ditemukan")
		return false
	}
	return true
}

func handleDeleteNilai(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	// nilai Join ke siswa untuk cek tenant
	res, err := db.Exec(`DELETE FROM nilai n USING siswa s WHERE n.id=$1 AND n.siswa_id=s.id AND s.sekolah_id=$2`, id, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, 404, "Nilai tidak ditemukan")
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

func handleDeleteHafalan(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	res, err := db.Exec(`DELETE FROM hafalan h USING siswa s WHERE h.id=$1 AND h.siswa_id=s.id AND s.sekolah_id=$2`, id, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, 404, "Hafalan tidak ditemukan")
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

func handleDeleteJadwal(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	if !deleteOwned(w, "jadwal", id, c.SekolahID, "sekolah_id") {
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

func handleDeleteAgenda(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	if !deleteOwned(w, "agenda", id, c.SekolahID, "sekolah_id") {
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

func handleDeletePengeluaran(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	if !deleteOwned(w, "pengeluaran", id, c.SekolahID, "sekolah_id") {
		return
	}
	logActivity(c.UserID, "hapus_pengeluaran", "Menghapus pengeluaran ID "+itoa(id))
	respond(w, 200, map[string]string{"ok": "1"})
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	if id == c.UserID {
		jsonErr(w, 400, "Tidak bisa menghapus akun sendiri")
		return
	}
	var sid int
	var role string
	if err := db.QueryRow(`SELECT COALESCE(sekolah_id,0), role FROM users WHERE id=$1`, id).Scan(&sid, &role); err != nil || sid != c.SekolahID {
		jsonErr(w, 404, "Pengguna tidak ditemukan")
		return
	}
	if role == "kepala_sekolah" {
		jsonErr(w, 403, "Kepala sekolah tidak bisa dihapus")
		return
	}
	res, err := db.Exec(`DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, 404, "Pengguna tidak ditemukan")
		return
	}
	logActivity(c.UserID, "hapus_user", "Menghapus pengguna ID "+itoa(id))
	respond(w, 200, map[string]string{"ok": "1"})
}

// ---------- WALI KELAS (Fase C) ----------

func handleListWali(w http.ResponseWriter, r *http.Request, c *Claims) {
	rows, err := db.Query(`SELECT w.kelas, u.nama, u.id FROM wali_kelas w JOIN users u ON u.id=w.user_id WHERE w.sekolah_id=$1 ORDER BY w.kelas`, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var kelas, nama string
		var uid int
		rows.Scan(&kelas, &nama, &uid)
		list = append(list, map[string]interface{}{"kelas": kelas, "wali": nama, "user_id": uid})
	}
	respond(w, 200, list)
}

func handleSetWali(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		Kelas  string `json:"kelas"`
		UserID int    `json:"user_id"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Kelas == "" || in.UserID == 0 {
		jsonErr(w, 400, "Kelas dan user wajib diisi")
		return
	}
	// user harus sekolah sama & bukan kepala_sekolah (wali = guru/TU)
	var sid int
	var role string
	if err := db.QueryRow(`SELECT COALESCE(sekolah_id,0), role FROM users WHERE id=$1`, in.UserID).Scan(&sid, &role); err != nil || sid != c.SekolahID {
		jsonErr(w, 404, "Pengguna tidak ditemukan")
		return
	}
	if role == "kepala_sekolah" {
		jsonErr(w, 400, "Kepala sekolah tidak bisa jadi wali kelas")
		return
	}
	db.Exec(`INSERT INTO wali_kelas (sekolah_id, kelas, user_id) VALUES ($1,$2,$3)
		ON CONFLICT (sekolah_id, kelas) DO UPDATE SET user_id=EXCLUDED.user_id`, c.SekolahID, in.Kelas, in.UserID)
	logActivity(c.UserID, "set_wali", "Wali kelas "+in.Kelas)
	respond(w, 200, map[string]string{"ok": "1"})
}

// rotateLog: panggil periodik — simpan maks. 5000 entri activity_log
func rotateLog() {
	db.Exec(`DELETE FROM activity_log WHERE id < (SELECT COALESCE(MAX(id),0) - 5000 FROM activity_log)`)
}

// ---------- EDIT (PUT) — jadwal, agenda, pengumuman (Fase polish) ----------

func handleUpdateJadwal(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var in struct {
		Kelas, Hari, JamMulai, JamSelesai, Mapel string
		GuruID                                   int
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Kelas == "" || in.Hari == "" || in.Mapel == "" {
		jsonErr(w, 400, "Kelas, hari, dan mapel wajib")
		return
	}
	if in.JamMulai == "" {
		in.JamMulai = "07:00"
	}
	if in.JamSelesai == "" {
		in.JamSelesai = "08:00"
	}
	res, err := db.Exec(`UPDATE jadwal SET kelas=$1, hari=$2, jam_mulai=$3::TIME, jam_selesai=$4::TIME, mapel=$5, guru_id=NULLIF($6,0)
		WHERE id=$7 AND sekolah_id=$8`, in.Kelas, in.Hari, in.JamMulai, in.JamSelesai, in.Mapel, in.GuruID, id, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, 404, "Jadwal tidak ditemukan")
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

func handleUpdateAgenda(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var in struct {
		Judul, Deskripsi, Tanggal, Waktu, Lokasi string
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Judul == "" || in.Tanggal == "" {
		jsonErr(w, 400, "Judul dan tanggal wajib")
		return
	}
	res, err := db.Exec(`UPDATE agenda SET judul=$1, deskripsi=NULLIF($2,''), tanggal=$3, waktu=NULLIF($4,'')::TIME, lokasi=NULLIF($5,'')
		WHERE id=$6 AND sekolah_id=$7`, in.Judul, in.Deskripsi, in.Tanggal, in.Waktu, in.Lokasi, id, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, 404, "Agenda tidak ditemukan")
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

func handleUpdatePengumuman(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var in struct {
		Judul, Isi, Target string
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Judul == "" || in.Isi == "" {
		jsonErr(w, 400, "Judul dan isi wajib")
		return
	}
	if in.Target == "" {
		in.Target = "semua"
	}
	res, err := db.Exec(`UPDATE pengumuman SET judul=$1, isi=$2, target=$3 WHERE id=$4 AND sekolah_id=$5`,
		in.Judul, in.Isi, in.Target, id, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, 404, "Pengumuman tidak ditemukan")
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}
