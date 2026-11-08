package main

import (
	"time"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func registerRoutes(mux *http.ServeMux) {
	// Public
	mux.HandleFunc("POST /api/login", handleLogin)
	mux.HandleFunc("POST /api/daftar", handleDaftar)
	mux.HandleFunc("GET /api/manifest", handleManifest)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]interface{}{"status": "ok", "demo": os.Getenv("SEED_DEMO") == "1"})
	})

	// Protected
	api := http.NewServeMux()
	api.Handle("GET /api/me", authMiddleware(http.HandlerFunc(handleMe)))
	api.Handle("POST /api/pengumuman", authMiddleware(writeGuard(roleMW(handleCreatePengumuman, "kepala_sekolah", "tu", "guru"))))
	api.Handle("GET /api/pengumuman", authMiddleware(http.HandlerFunc(handleListPengumuman)))
	api.Handle("DELETE /api/pengumuman/{id}", authMiddleware(writeGuard(roleMW(handleDeletePengumuman, "kepala_sekolah", "tu"))))
	api.Handle("PUT /api/pengumuman/{id}", authMiddleware(writeGuard(roleMW(handleUpdatePengumuman, "kepala_sekolah", "tu"))))

	// Siswa
	api.Handle("GET /api/siswa", authMiddleware(http.HandlerFunc(handleListSiswa)))
	api.Handle("POST /api/siswa", authMiddleware(writeGuard(roleMW(handleCreateSiswa, "tu", "kepala_sekolah"))))
	api.Handle("PUT /api/siswa/{id}", authMiddleware(writeGuard(roleMW(handleUpdateSiswa, "tu", "kepala_sekolah"))))
	api.Handle("DELETE /api/siswa/{id}", authMiddleware(writeGuard(roleMW(handleDeleteSiswa, "tu", "kepala_sekolah"))))

	// Absensi siswa
	api.Handle("GET /api/absensi-siswa", authMiddleware(http.HandlerFunc(handleListAbsensiSiswa)))
	api.Handle("POST /api/absensi-siswa", authMiddleware(writeGuard(roleMW(handleUpsertAbsensiSiswa, "guru", "tu", "kepala_sekolah"))))

	// Absensi guru
	api.Handle("GET /api/absensi-guru", authMiddleware(http.HandlerFunc(handleListAbsensiGuru)))
	api.Handle("POST /api/absensi-guru/masuk", authMiddleware(writeGuard(roleMW(handleAbsenMasuk, "guru", "tu", "kepala_sekolah"))))
	api.Handle("POST /api/absensi-guru/pulang", authMiddleware(writeGuard(roleMW(handleAbsenPulang, "guru", "tu", "kepala_sekolah"))))

	// Nilai
	api.Handle("GET /api/nilai", authMiddleware(http.HandlerFunc(handleListNilai)))
	api.Handle("POST /api/nilai", authMiddleware(writeGuard(roleMW(handleUpsertNilai, "guru", "kepala_sekolah"))))
	api.Handle("DELETE /api/nilai/{id}", authMiddleware(writeGuard(roleMW(handleDeleteNilai, "guru", "kepala_sekolah"))))

	// Hafalan
	api.Handle("GET /api/hafalan", authMiddleware(http.HandlerFunc(handleListHafalan)))
	api.Handle("POST /api/hafalan", authMiddleware(writeGuard(roleMW(handleCreateHafalan, "guru", "kepala_sekolah"))))
	api.Handle("DELETE /api/hafalan/{id}", authMiddleware(writeGuard(roleMW(handleDeleteHafalan, "guru", "kepala_sekolah"))))

	// SPP
	api.Handle("GET /api/spp", authMiddleware(http.HandlerFunc(handleListSPP)))
	api.Handle("POST /api/spp/generate", authMiddleware(writeGuard(roleMW(handleGenerateSPP, "tu", "kepala_sekolah"))))
	api.Handle("POST /api/spp/{id}/bayar", authMiddleware(writeGuard(roleMW(handleBayarSPP, "tu", "kepala_sekolah"))))

	// Keuangan
	api.Handle("GET /api/keuangan/ringkasan", authMiddleware(http.HandlerFunc(handleRingkasanKeuangan)))
	api.Handle("GET /api/pengeluaran", authMiddleware(http.HandlerFunc(handleListPengeluaran)))
	api.Handle("POST /api/pengeluaran", authMiddleware(writeGuard(roleMW(handleCreatePengeluaran, "tu", "kepala_sekolah"))))
	api.Handle("DELETE /api/pengeluaran/{id}", authMiddleware(writeGuard(roleMW(handleDeletePengeluaran, "tu", "kepala_sekolah"))))

	// Agenda & jadwal
	api.Handle("GET /api/agenda", authMiddleware(http.HandlerFunc(handleListAgenda)))
	api.Handle("POST /api/agenda", authMiddleware(writeGuard(roleMW(handleCreateAgenda, "kepala_sekolah", "tu", "guru"))))
	api.Handle("DELETE /api/agenda/{id}", authMiddleware(writeGuard(roleMW(handleDeleteAgenda, "kepala_sekolah", "tu"))))
	api.Handle("PUT /api/agenda/{id}", authMiddleware(writeGuard(roleMW(handleUpdateAgenda, "kepala_sekolah", "tu", "guru"))))
	api.Handle("GET /api/jadwal", authMiddleware(http.HandlerFunc(handleListJadwal)))
	api.Handle("GET /api/mapel", authMiddleware(http.HandlerFunc(handleListMapel)))
	api.Handle("GET /api/ekskul", authMiddleware(http.HandlerFunc(handleListEkskul)))
	api.Handle("POST /api/ekskul", authMiddleware(writeGuard(roleMWAuth(http.HandlerFunc(handleSaveEkskul), "kepala_sekolah", "tu"))))
	api.Handle("DELETE /api/ekskul/{id}", authMiddleware(writeGuard(roleMWAuth(http.HandlerFunc(handleDelEkskul), "kepala_sekolah", "tu"))))
	api.Handle("GET /api/ekskul/{id}/anggota", authMiddleware(http.HandlerFunc(handleEkskulAnggota)))
	api.Handle("POST /api/ekskul/{id}/anggota", authMiddleware(writeGuard(roleMWAuth(http.HandlerFunc(handleAddEkskulSiswa), "kepala_sekolah", "guru", "tu"))))
	api.Handle("POST /api/naik-kelas", authMiddleware(writeGuard(roleMWAuth(http.HandlerFunc(handleNaikKelas), "kepala_sekolah", "tu"))))
	api.Handle("POST /api/luluskan", authMiddleware(writeGuard(roleMWAuth(http.HandlerFunc(handleLuluskan), "kepala_sekolah", "tu"))))
	api.Handle("POST /api/jadwal", authMiddleware(writeGuard(roleMW(handleCreateJadwal, "tu", "kepala_sekolah"))))
	api.Handle("DELETE /api/jadwal/{id}", authMiddleware(writeGuard(roleMW(handleDeleteJadwal, "tu", "kepala_sekolah"))))
	api.Handle("PUT /api/jadwal/{id}", authMiddleware(writeGuard(roleMW(handleUpdateJadwal, "tu", "kepala_sekolah"))))

	// Dashboard
	api.Handle("GET /api/dashboard", authMiddleware(http.HandlerFunc(handleDashboard)))

	// Ekspor & rapor
	api.Handle("GET /api/export/absensi", authMiddleware(http.HandlerFunc(handleExportAbsensi)))
	api.Handle("GET /api/export/spp", authMiddleware(http.HandlerFunc(handleExportSPP)))
	api.Handle("GET /api/export/nilai", authMiddleware(http.HandlerFunc(handleExportNilai)))
	api.Handle("GET /api/export/keuangan", authMiddleware(http.HandlerFunc(handleExportKeuangan)))
	api.Handle("GET /api/export/hafalan", authMiddleware(http.HandlerFunc(handleExportHafalan)))
	api.Handle("GET /api/export/siswa", authMiddleware(roleMWAuth(http.HandlerFunc(handleExportSiswa), "kepala_sekolah", "tu")))
	api.Handle("GET /api/export/rekap", authMiddleware(roleMWAuth(planMWClaimsExport(handleExportRekap), "kepala_sekolah", "tu")))
	api.Handle("GET /api/export/all", authMiddleware(roleMWAuth(http.HandlerFunc(handleExportAll), "kepala_sekolah", "tu")))
	api.Handle("GET /api/kode-ref", authMiddleware(http.HandlerFunc(handleKodeRefSaya)))
	api.Handle("GET /api/laporan-mingguan", authMiddleware(http.HandlerFunc(handleLaporanMingguan)))
	api.Handle("POST /api/wa-pengasuh", authMiddleware(writeGuard(roleMW(handleSetWaPengasuh, "kepala_sekolah", "tu"))))
	api.Handle("GET /api/import/template", authMiddleware(http.HandlerFunc(handleTemplateImport)))
	api.Handle("POST /api/import/preview", authMiddleware(writeGuard(http.HandlerFunc(handleImportPreview))))
	api.Handle("POST /api/import/siswa", authMiddleware(writeGuard(roleMW(handleImportSiswa, "tu", "kepala_sekolah"))))
	api.Handle("GET /api/rapor/{id}", authMiddleware(http.HandlerFunc(handleRaporData)))
	api.Handle("GET /api/rapor-cetak/{id}", authMiddleware(http.HandlerFunc(handleCetakRapor)))
	api.Handle("GET /api/kwitansi/{id}", authOrShare(http.HandlerFunc(handleKwitansi)))
	api.Handle("POST /api/kwitansi/{id}/share", authMiddleware(writeGuard(roleMW(handleKwitansiShare, "kepala_sekolah", "tu"))))
	api.Handle("GET /api/laporan-wali/{id}", http.HandlerFunc(handleLaporanWali))
	api.Handle("GET /api/wali-token/{id}", authMiddleware(authedHandler(handleWaliTokenGet)))
	api.Handle("POST /api/wali-token/{id}", authMiddleware(writeGuard(roleMW(handleWaliTokenReset, "kepala_sekolah", "tu"))))
	api.Handle("GET /api/wali", authMiddleware(authedHandler(handleListWali)))
	api.Handle("POST /api/wali", authMiddleware(writeGuard(roleMW(handleSetWali, "kepala_sekolah", "tu"))))
	api.Handle("GET /api/rekap", authMiddleware(roleMWAuth(planMWClaims("rekap", handleRekapBulanan), "kepala_sekolah", "tu")))
	api.Handle("GET /api/siswa-by-kelas", authMiddleware(http.HandlerFunc(handleSiswaByKelas)))
	api.Handle("POST /api/nilai-bulk", authMiddleware(writeGuard(roleMW(handleBulkNilai, "guru", "kepala_sekolah"))))

	// Langganan & notifikasi
	api.Handle("GET /api/langganan", authMiddleware(http.HandlerFunc(handleLangganan)))
	api.Handle("GET /api/notif/spp", authMiddleware(http.HandlerFunc(handleNotifSPP)))

	// Pengaturan sekolah
	api.Handle("GET /api/pengaturan", authMiddleware(http.HandlerFunc(handleGetPengaturan)))
	api.Handle("PUT /api/pengaturan", authMiddleware(writeGuard(roleMW(handleUpdatePengaturan, "kepala_sekolah", "tu"))))

	// Reset password (kepala sekolah)
	api.Handle("POST /api/users/{id}/reset-password", authMiddleware(writeGuard(roleMW(handleResetPassword, "kepala_sekolah", "tu"))))

	// Ubah password sendiri
	api.Handle("POST /api/ubah-password", authMiddleware(writeGuard(roleMW(handleUbahPassword, "kepala_sekolah", "guru", "tu"))))

	// Log aktivitas (kepala sekolah)
	api.Handle("GET /api/activity-log", authMiddleware(roleMWAuth(http.HandlerFunc(handleListLog), "kepala_sekolah")))

	// Catatan rapor (wali kelas/guru/kepala)
	api.Handle("GET /api/rapor-catatan/{id}", authMiddleware(http.HandlerFunc(handleGetCatatanRapor)))
	api.Handle("PUT /api/rapor-catatan/{id}", authMiddleware(writeGuard(roleMW(handleSaveCatatanRapor, "guru", "kepala_sekolah"))))

	// User management (kepala sekolah)
	api.Handle("GET /api/users", authMiddleware(writeGuard(roleMW(handleListUsers, "kepala_sekolah", "tu", "platform"))))
	api.Handle("POST /api/users", authMiddleware(writeGuard(roleMW(handleCreateUser, "kepala_sekolah", "tu"))))
	api.Handle("DELETE /api/users/{id}", authMiddleware(writeGuard(roleMW(handleDeleteUser, "kepala_sekolah"))))

	// Panel admin platform (role khusus owner)
	api.Handle("GET /api/admin/sekolah", authMiddleware(writeGuard(roleMW(handleAdminSekolah, "platform"))))
	api.Handle("POST /api/admin/set-plan", authMiddleware(writeGuard(roleMW(handleAdminSetPlan, "platform"))))
	api.Handle("GET /api/admin/stats", authMiddleware(writeGuard(roleMW(handleAdminStats, "platform"))))

	mux.Handle("/api/", api)
}

func roleMW(h func(http.ResponseWriter, *http.Request, *Claims), roles ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := userFromContext(r)
		if c == nil {
			jsonErr(w, 401, "Unauthorized")
			return
		}
		for _, role := range roles {
			if c.Role == role {
				h(w, r, c)
				return
			}
		}
		jsonErr(w, 403, "Akses ditolak untuk peran "+c.Role)
	})
}

// authedHandler: handler dengan Claims, tanpa pembatasan role
func authedHandler(h func(http.ResponseWriter, *http.Request, *Claims)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := userFromContext(r)
		if c == nil {
			jsonErr(w, 401, "Unauthorized")
			return
		}
		h(w, r, c)
	})
}

// handleKodeRefSaya: kode rujukan sekolah user (untuk halaman Paket)
func handleKodeRefSaya(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	if c == nil {
		jsonErr(w, 401, "Unauthorized")
		return
	}
	var kode string
	db.QueryRow(`SELECT COALESCE(kode_ref,'') FROM sekolah WHERE id=$1`, c.SekolahID).Scan(&kode)
	respond(w, 200, map[string]string{"kode_ref": kode})
}

// writeGuard: tolak input baru saat mode baca (langganan habis) — data tetap bisa dilihat & diunduh
func writeGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c := userFromContext(r); c != nil && readMode(c.SekolahID) {
			jsonErr(w, 402, "Langganan sedang tidak aktif, jadi sementara hanya bisa melihat dan mengunduh data. Data Anda aman dan bisa diunduh kapan saja di menu Pengaturan.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// roleMWAuth: roleMW yang menerima http.Handler (untuk kombinasi dengan planMW)
func roleMWAuth(next http.Handler, roles ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := userFromContext(r)
		if c == nil {
			jsonErr(w, 401, "Unauthorized")
			return
		}
		for _, role := range roles {
			if c.Role == role {
				next.ServeHTTP(w, r)
				return
			}
		}
		jsonErr(w, 403, "Akses ditolak untuk peran "+c.Role)
	})
}

// ---------- AUTH ----------

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email, Password string
	}
	json.NewDecoder(r.Body).Decode(&in)
	email := strings.TrimSpace(strings.ToLower(in.Email))
	ip := clientIP(r)
	key := ip + "|" + email
	if msg := loginRateLimit(key); msg != "" {
		jsonErr(w, 429, msg)
		return
	}
	var u User
	var hash string
	var sekolahID int
	err := db.QueryRow(`SELECT id,nama,email,role,COALESCE(nip,'-'),password,COALESCE(sekolah_id,1) FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.Nama, &u.Email, &u.Role, &u.NIP, &hash, &sekolahID)
	u.SekolahID = sekolahID
	if err != nil {
		checkPass(dummyHash, in.Password) // anti user-enumeration via timing
		loginRecordFail(key)
		jsonErr(w, 401, "Email atau password salah")
		return
	}
	if !checkPass(hash, in.Password) {
		loginRecordFail(key)
		jsonErr(w, 401, "Email atau password salah")
		return
	}
	// Cek langganan sekolah
	// ANTI-SANDERA: langganan habis TIDAK memblokir login — user masih bisa
	// melihat & mengunduh datanya (input baru ditolak per-endpoint oleh writeGuard).
	// Users dengan role 'platform' tidak terikat sekolah — cekLangganan dilewati
	if u.Role != "platform" {
		if ok, msg := cekLangganan(u.SekolahID); !ok {
			// suspend = satu-satunya blokir penuh (pelanggaran)
			if strings.Contains(msg, "ditangguhkan") {
				jsonErr(w, 402, msg)
				return
			}
			// trial habis / langganan lewat: login tetap boleh (mode baca)
			_ = msg
		}
	}
	loginRecordSuccess(key)
	tok, _ := makeToken(&u)
	logActivity(u.ID, "login", "Login berhasil")
	respond(w, 200, map[string]interface{}{"token": tok, "user": u})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	var jenis, jenjang string
	db.QueryRow(`SELECT COALESCE(jenis,'umum'), COALESCE(jenjang,'') FROM sekolah WHERE id=$1`, c.SekolahID).Scan(&jenis, &jenjang)
	respond(w, 200, map[string]interface{}{"id": c.UserID, "nama": c.Nama, "role": c.Role, "jenis": jenis, "jenjang": jenjang})
}

// ---------- SISWA ----------

func handleListSiswa(w http.ResponseWriter, r *http.Request) {
	offset, limit := paginate(r)
	q := `SELECT id, COALESCE(nis,''), COALESCE(nisn,''), nama, COALESCE(jk,''), kelas, COALESCE(tgl_lahir, DATE '1900-01-01'), COALESCE(alamat,''), COALESCE(nama_ortu,''), COALESCE(hp_ortu,''), status FROM siswa`
	args := []interface{}{}
	where := ""
	if s := r.URL.Query().Get("kelas"); s != "" {
		where += ` WHERE kelas=$1`
		args = append(args, s)
	}
	search := r.URL.Query().Get("q")
	if search != "" {
		if where == "" {
			where = " WHERE "
		} else {
			where += " AND "
		}
		args = append(args, "%"+search+"%")
		where += `(nama ILIKE $` + itoa(len(args)) + ` OR COALESCE(nis,'') ILIKE $` + itoa(len(args)) + `)`
	}
	// filter sekolah untuk semua role (multi-tenant isolation)
	if userFromContext(r) != nil && userFromContext(r).SekolahID > 0 {
		if where == "" {
			where = " WHERE "
		} else {
			where += " AND "
		}
		where += `sekolah_id=` + itoa(userFromContext(r).SekolahID)
	}
	var total int
	db.QueryRow(`SELECT COUNT(*) FROM siswa`+where, args...).Scan(&total)
	q += where + ` ORDER BY kelas, nama LIMIT ` + itoa(limit) + ` OFFSET ` + itoa(offset)
	rows, err := db.Query(q, args...)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var nis, nisn, nama, jk, kelas, tgl, alamat, ortu, hp, status string
		rows.Scan(&id, &nis, &nisn, &nama, &jk, &kelas, &tgl, &alamat, &ortu, &hp, &status)
		list = append(list, map[string]interface{}{"id": id, "nis": nis, "nisn": nisn, "nama": nama, "jk": jk, "kelas": kelas, "tgl_lahir": tgl, "alamat": alamat, "nama_ortu": ortu, "hp_ortu": hp, "status": status})
	}
	resp := map[string]interface{}{"data": list}
	for k, v := range pageMeta(r, total) {
		resp[k] = v
	}
	respond(w, 200, resp)
}

func siswaFromRequest(r *http.Request) (map[string]interface{}, error) {
	var in map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return nil, err
	}
	for _, k := range []string{"nama", "kelas"} {
		if v, _ := in[k].(string); v == "" {
			return nil, errors.New("Field " + k + " wajib diisi")
		}
	}
	return in, nil
}

func sval(m map[string]interface{}, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func handleCreateSiswa(w http.ResponseWriter, r *http.Request, c *Claims) {
	if msg := cekKuota(c.SekolahID, "siswa"); msg != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(402)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}
	in, err := siswaFromRequest(r)
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	var id int
	err = db.QueryRow(`INSERT INTO siswa (nis,nisn,nama,jk,kelas,tgl_lahir,alamat,nama_ortu,hp_ortu,sekolah_id) VALUES (NULLIF($1,''),NULLIF($2,''),$3,NULLIF($4,''),$5,NULLIF($6,'')::DATE,NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),$10) RETURNING id`,
		sval(in, "nis"), sval(in, "nisn"), sval(in, "nama"), sval(in, "jk"), sval(in, "kelas"), sval(in, "tgl_lahir"), sval(in, "alamat"), sval(in, "nama_ortu"), sval(in, "hp_ortu"), c.SekolahID).Scan(&id)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 201, map[string]interface{}{"id": id})
}

func handleUpdateSiswa(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	in, err := siswaFromRequest(r)
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	res, err := db.Exec(`UPDATE siswa SET nis=NULLIF($1,''), nisn=NULLIF($2,''), nama=$3, jk=NULLIF($4,''), kelas=$5, tgl_lahir=NULLIF($6,'')::DATE, alamat=NULLIF($7,''), nama_ortu=NULLIF($8,''), hp_ortu=NULLIF($9,''), status=COALESCE(NULLIF($10,''),'aktif') WHERE id=$11 AND sekolah_id=$12`,
		sval(in, "nis"), sval(in, "nisn"), sval(in, "nama"), sval(in, "jk"), sval(in, "kelas"), sval(in, "tgl_lahir"), sval(in, "alamat"), sval(in, "nama_ortu"), sval(in, "hp_ortu"), sval(in, "status"), id, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

func handleDeleteSiswa(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var sid int
	if err := db.QueryRow(`SELECT sekolah_id FROM siswa WHERE id=$1`, id).Scan(&sid); err != nil || sid != c.SekolahID {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return
	}
	// ARSIP, bukan hapus: data nilai/SPP/hafalan tetap aman.
	// Siswa berstatus 'keluar' tidak tampil di daftar aktif & tidak ikut generate SPP.
	res, err := db.Exec(`UPDATE siswa SET status='keluar' WHERE id=$1 AND sekolah_id=$2`, id, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return
	}
	logActivity(c.UserID, "arsip_siswa", "Mengarsipkan siswa ID "+itoa(id)+" (data tetap tersimpan)")
	respond(w, 200, map[string]string{"ok": "1", "pesan": "Siswa diarsipkan. Semua nilai, tagihan, dan hafalannya tetap tersimpan."})
}

// ---------- ABSENSI SISWA ----------

func handleListAbsensiSiswa(w http.ResponseWriter, r *http.Request) {
	tanggal := r.URL.Query().Get("tanggal")
	kelas := r.URL.Query().Get("kelas")
	if tanggal == "" {
		tanggal = today()
	}
	q := `SELECT s.id, s.nama, s.kelas, COALESCE(a.status,''), COALESCE(a.keterangan,'')
		FROM siswa s LEFT JOIN absensi_siswa a ON a.siswa_id=s.id AND a.tanggal=$1`
	args := []interface{}{tanggal}
	where := ` WHERE s.sekolah_id=` + itoa(userFromContext(r).SekolahID) + ` AND s.status='aktif'`
	if kelas != "" {
		where += ` AND s.kelas=$2`
		args = append(args, kelas)
	}
	q += where + ` ORDER BY s.kelas, s.nama`
	rows, err := db.Query(q, args...)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var nama, kls, st, ket string
		rows.Scan(&id, &nama, &kls, &st, &ket)
		list = append(list, map[string]interface{}{"siswa_id": id, "nama": nama, "kelas": kls, "status": st, "keterangan": ket})
	}
	respond(w, 200, list)
}

func handleUpsertAbsensiSiswa(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		Items []struct {
			SiswaID    int    `json:"siswa_id"`
			Status     string `json:"status"`
			Keterangan string `json:"keterangan"`
		} `json:"items"`
		Tanggal string `json:"tanggal"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Tanggal == "" {
		in.Tanggal = today()
	}
	n := 0
	// verifikasi sekali: semua siswa dalam batch harus milik sekolah yang sama
	for _, it := range in.Items {
		if it.Status == "" {
			continue
		}
		var sid int
		if err := db.QueryRow(`SELECT sekolah_id FROM siswa WHERE id=$1`, it.SiswaID).Scan(&sid); err != nil || sid != c.SekolahID {
			continue
		}
		_, err := db.Exec(`INSERT INTO absensi_siswa (siswa_id,tanggal,status,keterangan,pencatat) VALUES ($1,$2,$3,NULLIF($4,''),$5)
			ON CONFLICT (siswa_id,tanggal) DO UPDATE SET status=EXCLUDED.status, keterangan=EXCLUDED.keterangan, pencatat=EXCLUDED.pencatat`,
			it.SiswaID, in.Tanggal, it.Status, it.Keterangan, c.UserID)
		if err == nil {
			n++
		}
	}
	respond(w, 200, map[string]interface{}{"saved": n})
}

// ---------- ABSENSI GURU ----------

func handleListAbsensiGuru(w http.ResponseWriter, r *http.Request) {
	tanggal := r.URL.Query().Get("tanggal")
	if tanggal == "" {
		tanggal = today()
	}
	rows, err := db.Query(`SELECT u.id, u.nama, u.role, COALESCE(g.jam_masuk::text,''), COALESCE(g.jam_pulang::text,''), COALESCE(g.status,''), COALESCE(g.keterangan,'')
		FROM users u LEFT JOIN absensi_guru g ON g.guru_id=u.id AND g.tanggal=$1 WHERE u.sekolah_id=$2 ORDER BY u.role, u.nama`, tanggal, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var nama, role, jm, jp, st, ket string
		rows.Scan(&id, &nama, &role, &jm, &jp, &st, &ket)
		list = append(list, map[string]interface{}{"guru_id": id, "nama": nama, "role": role, "jam_masuk": jm, "jam_pulang": jp, "status": st, "keterangan": ket})
	}
	respond(w, 200, list)
}

func handleAbsenMasuk(w http.ResponseWriter, r *http.Request, c *Claims) {
	_, err := db.Exec(`INSERT INTO absensi_guru (guru_id,tanggal,jam_masuk,status) VALUES ($1,CURRENT_DATE,CURRENT_TIME,'hadir')
		ON CONFLICT (guru_id,tanggal) DO UPDATE SET jam_masuk=CURRENT_TIME, status='hadir'`, c.UserID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

func handleAbsenPulang(w http.ResponseWriter, r *http.Request, c *Claims) {
	_, err := db.Exec(`INSERT INTO absensi_guru (guru_id,tanggal,jam_masuk,status) VALUES ($1,CURRENT_DATE,CURRENT_TIME,'hadir')
		ON CONFLICT (guru_id,tanggal) DO UPDATE SET jam_pulang=CURRENT_TIME`, c.UserID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

// ---------- NILAI ----------

func handleListNilai(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	base := `SELECT n.id, s.nama, n.kelas, n.mapel, n.jenis, n.semester, n.tahun_ajaran, n.nilai
		FROM nilai n JOIN siswa s ON s.id=n.siswa_id WHERE s.sekolah_id=` + itoa(userFromContext(r).SekolahID)
	args := []interface{}{}
	i := 1
	for _, f := range []string{"kelas", "mapel", "jenis", "semester", "tahun_ajaran"} {
		if v := q.Get(f); v != "" {
			base += " AND n." + f + "=$" + strconv.Itoa(i)
			args = append(args, v)
			i++
		}
	}
	var total int
	countWhere := strings.Replace(strings.TrimPrefix(base, `SELECT n.id, s.nama, n.kelas, n.mapel, n.jenis, n.semester, n.tahun_ajaran, n.nilai
		FROM nilai n JOIN siswa s ON s.id=n.siswa_id `), "ORDER BY", "", 1)
	db.QueryRow(`SELECT COUNT(*) FROM nilai n JOIN siswa s ON s.id=n.siswa_id `+countWhere, args...).Scan(&total)
	offset, limit := paginate(r)
	base += ` ORDER BY n.kelas, s.nama, n.mapel LIMIT ` + strconv.Itoa(limit) + ` OFFSET ` + strconv.Itoa(offset)
	rows, err := db.Query(base, args...)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var nama, kelas, mapel, jenis, sem, ta string
		var nilai float64
		rows.Scan(&id, &nama, &kelas, &mapel, &jenis, &sem, &ta, &nilai)
		list = append(list, map[string]interface{}{"id": id, "siswa": nama, "kelas": kelas, "mapel": mapel, "jenis": jenis, "semester": sem, "tahun_ajaran": ta, "nilai": nilai})
	}
	resp := map[string]interface{}{"data": list}
	for k, v := range pageMeta(r, total) {
		resp[k] = v
	}
	respond(w, 200, resp)
}

// tahunAjaranAktif membaca pengaturan sekolah; fallback 2026/2027
func tahunAjaranSekolah(sekolahID int) string {
	var ta string
	db.QueryRow(`SELECT COALESCE(tahun_ajaran_aktif,'') FROM pengaturan WHERE id=$1`, sekolahID).Scan(&ta)
	if ta == "" {
		ta = "2026/2027"
	}
	return ta
}

func handleUpsertNilai(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		SiswaID     int     `json:"siswa_id"`
		Mapel       string  `json:"mapel"`
		Kelas       string  `json:"kelas"`
		Jenis       string  `json:"jenis"`
		Semester    string  `json:"semester"`
		TahunAjaran string  `json:"tahun_ajaran"`
		Nilai       float64 `json:"nilai"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Nilai < 0 || in.Nilai > 100 {
		jsonErr(w, 400, "Nilai harus di antara 0 dan 100")
		return
	}
	// pastikan siswa milik sekolah yang sama
	var sid int
	if err := db.QueryRow(`SELECT sekolah_id FROM siswa WHERE id=$1`, in.SiswaID).Scan(&sid); err != nil || sid != c.SekolahID {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return
	}
	if in.TahunAjaran == "" {
		in.TahunAjaran = tahunAjaranSekolah(c.SekolahID)
	}
	_, err := db.Exec(`INSERT INTO nilai (siswa_id,mapel,kelas,jenis,semester,tahun_ajaran,nilai,guru_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (siswa_id,mapel,jenis,semester,tahun_ajaran) DO UPDATE SET nilai=EXCLUDED.nilai, guru_id=EXCLUDED.guru_id`,
		in.SiswaID, in.Mapel, in.Kelas, in.Jenis, in.Semester, in.TahunAjaran, in.Nilai, c.UserID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

// ---------- HAFALAN ----------

func handleListHafalan(w http.ResponseWriter, r *http.Request) {
	q := `SELECT h.id, s.nama, h.surah, h.ayat_dari, h.ayat_sampai, h.status, h.tanggal, COALESCE(h.catatan,'')
		FROM hafalan h JOIN siswa s ON s.id=h.siswa_id WHERE s.sekolah_id=` + itoa(userFromContext(r).SekolahID)
	args := []interface{}{}
	if v := r.URL.Query().Get("siswa_id"); v != "" {
		q += ` AND h.siswa_id=$1`
		args = append(args, v)
	}
	q += ` ORDER BY h.tanggal DESC LIMIT 200`
	rows, err := db.Query(q, args...)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id, dari, sampai int
		var nama, surah, status, tgl, catatan string
		rows.Scan(&id, &nama, &surah, &dari, &sampai, &status, &tgl, &catatan)
		list = append(list, map[string]interface{}{"id": id, "siswa": nama, "surah": surah, "ayat_dari": dari, "ayat_sampai": sampai, "status": status, "tanggal": tgl, "catatan": catatan})
	}
	respond(w, 200, list)
}

func handleCreateHafalan(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		SiswaID    int    `json:"siswa_id"`
		Surah      string `json:"surah"`
		AyatDari   int    `json:"ayat_dari"`
		AyatSampai int    `json:"ayat_sampai"`
		Status     string `json:"status"`
		Catatan    string `json:"catatan"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Surah == "" || in.Status == "" {
		jsonErr(w, 400, "Surah dan status wajib")
		return
	}
	var sid int
	if err := db.QueryRow(`SELECT sekolah_id FROM siswa WHERE id=$1`, in.SiswaID).Scan(&sid); err != nil || sid != c.SekolahID {
		jsonErr(w, 404, "Siswa tidak ditemukan")
		return
	}
	_, err := db.Exec(`INSERT INTO hafalan (siswa_id,surah,ayat_dari,ayat_sampai,status,penyetor_guru_id,catatan) VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''))`,
		in.SiswaID, in.Surah, in.AyatDari, in.AyatSampai, in.Status, c.UserID, in.Catatan)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 201, map[string]string{"ok": "1"})
}

// ---------- SPP ----------

func handleListSPP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	base := `SELECT sp.id, s.nama, s.kelas, sp.bulan, sp.tahun, sp.jumlah, COALESCE(sp.status,'belum'), COALESCE(sp.tgl_bayar::text,''), COALESCE(sp.metode,''), COALESCE(s.hp_ortu,'')
		FROM spp sp JOIN siswa s ON s.id=sp.siswa_id WHERE s.sekolah_id=` + itoa(userFromContext(r).SekolahID)
	args := []interface{}{}
	i := 1
	for _, f := range []string{"bulan", "tahun", "status"} {
		if v := q.Get(f); v != "" {
			base += " AND sp." + f + "=$" + strconv.Itoa(i)
			args = append(args, v)
			i++
		}
	}
	var total int
	countWhere := strings.Replace(strings.TrimPrefix(base, `SELECT sp.id, s.nama, s.kelas, sp.bulan, sp.tahun, sp.jumlah, COALESCE(sp.status,'belum'), COALESCE(sp.tgl_bayar::text,''), COALESCE(sp.metode,'')
		FROM spp sp JOIN siswa s ON s.id=sp.siswa_id `), "ORDER BY", "", 1)
	db.QueryRow(`SELECT COUNT(*) FROM spp sp JOIN siswa s ON s.id=sp.siswa_id `+countWhere, args...).Scan(&total)
	offset, limit := paginate(r)
	base += ` ORDER BY s.kelas, s.nama LIMIT ` + strconv.Itoa(limit) + ` OFFSET ` + strconv.Itoa(offset)
	rows, err := db.Query(base, args...)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var nama, kelas, bulan, tahun, status, tgl, metode, hp string
		var jumlah float64
		rows.Scan(&id, &nama, &kelas, &bulan, &tahun, &jumlah, &status, &tgl, &metode, &hp)
		list = append(list, map[string]interface{}{"id": id, "siswa": nama, "kelas": kelas, "bulan": bulan, "tahun": tahun, "jumlah": jumlah, "status": status, "tgl_bayar": tgl, "metode": metode, "hp": hp})
	}
	resp := map[string]interface{}{"data": list}
	for k, v := range pageMeta(r, total) {
		resp[k] = v
	}
	respond(w, 200, resp)
}

var bulanID = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func handleGenerateSPP(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		Bulan  string  `json:"bulan"`
		Tahun  string  `json:"tahun"`
		Jumlah float64 `json:"jumlah"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Bulan == "" || in.Tahun == "" {
		jsonErr(w, 400, "Bulan dan tahun wajib")
		return
	}
	if in.Jumlah == 0 {
		db.QueryRow(`SELECT COALESCE(spp_default,45000) FROM pengaturan WHERE id=$1`, c.SekolahID).Scan(&in.Jumlah)
	}
	rows, err := db.Query(`SELECT id FROM siswa WHERE status='aktif' AND sekolah_id=$1`, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer tx.Rollback()
	n := 0
	for _, id := range ids {
		_, err := tx.Exec(`INSERT INTO spp (siswa_id,bulan,tahun,jumlah,status) VALUES ($1,$2,$3,$4,'belum') ON CONFLICT (siswa_id,bulan,tahun) DO NOTHING`, id, in.Bulan, in.Tahun, in.Jumlah)
		if err == nil {
			n++
		}
	}
	if err := tx.Commit(); err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	logActivity(c.UserID, "generate_spp", "Generate SPP "+in.Bulan+" "+in.Tahun+": "+itoa(n)+" tagihan")
	respond(w, 200, map[string]interface{}{"generated": n})
}

func handleBayarSPP(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var in struct {
		Metode string `json:"metode"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Metode == "" {
		in.Metode = "tunai"
	}
	_, err := db.Exec(`UPDATE spp SET status='lunas', tgl_bayar=CURRENT_DATE, metode=$1, petugas_id=$2 WHERE id=$3 AND sekolah_id=$4`, in.Metode, c.UserID, id, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 200, map[string]string{"ok": "1"})
}

// ---------- KEUANGAN ----------

func handleRingkasanKeuangan(w http.ResponseWriter, r *http.Request) {
	sid := userFromContext(r).SekolahID
	var pemasukan, pengeluaran float64
	var sppBelum int
	db.QueryRow(`SELECT COALESCE(SUM(jumlah),0) FROM spp WHERE status='lunas' AND sekolah_id=$1`, sid).Scan(&pemasukan)
	db.QueryRow(`SELECT COALESCE(SUM(jumlah),0) FROM pengeluaran WHERE sekolah_id=$1`, sid).Scan(&pengeluaran)
	db.QueryRow(`SELECT COUNT(*) FROM spp WHERE status='belum' AND sekolah_id=$1`, sid).Scan(&sppBelum)
	rows, err := db.Query(`SELECT kategori, COALESCE(SUM(jumlah),0) FROM pengeluaran WHERE sekolah_id=$1 GROUP BY kategori ORDER BY 2 DESC`, sid)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	perKategori := []map[string]interface{}{}
	for rows.Next() {
		var kat string
		var jml float64
		rows.Scan(&kat, &jml)
		perKategori = append(perKategori, map[string]interface{}{"kategori": kat, "jumlah": jml})
	}
	respond(w, 200, map[string]interface{}{
		"total_pemasukan": pemasukan, "total_pengeluaran": pengeluaran,
		"saldo": pemasukan - pengeluaran, "spp_belum_lunas": sppBelum, "pengeluaran_per_kategori": perKategori,
	})
}

func handleListPengeluaran(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, tanggal::text, uraian, COALESCE(kategori,''), jumlah FROM pengeluaran WHERE sekolah_id=$1 ORDER BY tanggal DESC, id DESC LIMIT 200`, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var tgl, uraian, kat string
		var jml float64
		rows.Scan(&id, &tgl, &uraian, &kat, &jml)
		list = append(list, map[string]interface{}{"id": id, "tanggal": tgl, "uraian": uraian, "kategori": kat, "jumlah": jml})
	}
	respond(w, 200, list)
}

func handleCreatePengeluaran(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		Uraian   string  `json:"uraian"`
		Kategori string  `json:"kategori"`
		Jumlah   float64 `json:"jumlah"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Uraian == "" || in.Jumlah <= 0 {
		jsonErr(w, 400, "Uraian dan jumlah wajib")
		return
	}
	_, err := db.Exec(`INSERT INTO pengeluaran (uraian,kategori,jumlah,pencatat_id,sekolah_id) VALUES ($1,NULLIF($2,''),$3,$4,$5)`, in.Uraian, in.Kategori, in.Jumlah, c.UserID, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 201, map[string]string{"ok": "1"})
}

// ---------- PENGUMUMAN ----------

func handleListPengumuman(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT p.id, p.judul, p.isi, p.target, p.created_at::text, u.nama FROM pengumuman p JOIN users u ON u.id=p.pembuat_id WHERE p.sekolah_id=$1 ORDER BY p.created_at DESC LIMIT 100`, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var judul, isi, target, tgl, pembuat string
		rows.Scan(&id, &judul, &isi, &target, &tgl, &pembuat)
		list = append(list, map[string]interface{}{"id": id, "judul": judul, "isi": isi, "target": target, "created_at": tgl, "pembuat": pembuat})
	}
	respond(w, 200, list)
}

func handleCreatePengumuman(w http.ResponseWriter, r *http.Request, c *Claims) {
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
	_, err := db.Exec(`INSERT INTO pengumuman (judul,isi,target,pembuat_id,sekolah_id) VALUES ($1,$2,$3,$4,$5)`, in.Judul, in.Isi, in.Target, c.UserID, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 201, map[string]string{"ok": "1"})
}

func handleDeletePengumuman(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	db.Exec(`DELETE FROM pengumuman WHERE id=$1 AND sekolah_id=$2`, id, c.SekolahID)
	respond(w, 200, map[string]string{"ok": "1"})
}

// ---------- AGENDA ----------

func handleListAgenda(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT a.id, a.judul, COALESCE(a.deskripsi,''), a.tanggal::text, COALESCE(a.waktu::text,''), COALESCE(a.lokasi,''), u.nama
		FROM agenda a JOIN users u ON u.id=a.pembuat_id WHERE a.sekolah_id=$1 ORDER BY a.tanggal, a.waktu LIMIT 100`, userFromContext(r).SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var judul, desk, tgl, waktu, lokasi, pembuat string
		rows.Scan(&id, &judul, &desk, &tgl, &waktu, &lokasi, &pembuat)
		list = append(list, map[string]interface{}{"id": id, "judul": judul, "deskripsi": desk, "tanggal": tgl, "waktu": waktu, "lokasi": lokasi, "pembuat": pembuat})
	}
	respond(w, 200, list)
}

func handleCreateAgenda(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		Judul, Deskripsi, Tanggal, Waktu, Lokasi string
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Judul == "" || in.Tanggal == "" {
		jsonErr(w, 400, "Judul dan tanggal wajib")
		return
	}
	_, err := db.Exec(`INSERT INTO agenda (judul,deskripsi,tanggal,waktu,lokasi,pembuat_id,sekolah_id) VALUES ($1,NULLIF($2,''),$3,NULLIF($4,'')::TIME,NULLIF($5,''),$6,$7)`,
		in.Judul, in.Deskripsi, in.Tanggal, in.Waktu, in.Lokasi, c.UserID, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 201, map[string]string{"ok": "1"})
}

// ---------- JADWAL ----------

func handleListJadwal(w http.ResponseWriter, r *http.Request) {
	sid := userFromContext(r).SekolahID
	q := `SELECT j.id, j.kelas, j.hari, j.jam_mulai::text, j.jam_selesai::text, j.mapel, COALESCE(u.nama,'-')
		FROM jadwal j LEFT JOIN users u ON u.id=j.guru_id WHERE j.sekolah_id=$1`
	args := []interface{}{sid}
	if v := r.URL.Query().Get("kelas"); v != "" {
		q += ` AND j.kelas=$2`
		args = append(args, v)
	}
	q += ` ORDER BY ARRAY_POSITION(ARRAY['Senin','Selasa','Rabu','Kamis','Jumat','Sabtu'], j.hari), j.jam_mulai`
	rows, err := db.Query(q, args...)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var kelas, hari, mulai, selesai, mapel, guru string
		rows.Scan(&id, &kelas, &hari, &mulai, &selesai, &mapel, &guru)
		list = append(list, map[string]interface{}{"id": id, "kelas": kelas, "hari": hari, "jam_mulai": mulai, "jam_selesai": selesai, "mapel": mapel, "guru": guru})
	}
	respond(w, 200, list)
}

func handleCreateJadwal(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct {
		Kelas      string `json:"kelas"`
		Hari       string `json:"hari"`
		JamMulai   string `json:"jam_mulai"`
		JamSelesai string `json:"jam_selesai"`
		Mapel      string `json:"mapel"`
		GuruID     int    `json:"guru_id"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Kelas == "" || in.Hari == "" || in.Mapel == "" {
		jsonErr(w, 400, "Kelas, hari, dan mapel wajib")
		return
	}
	if in.JamMulai == "" || in.JamSelesai == "" {
		jsonErr(w, 400, "Jam mulai dan jam selesai wajib (format HH:MM)")
		return
	}
	_, err := db.Exec(`INSERT INTO jadwal (kelas,hari,jam_mulai,jam_selesai,mapel,guru_id,sekolah_id) VALUES ($1,$2,$3,$4,$5,NULLIF($6,0),$7)`,
		in.Kelas, in.Hari, in.JamMulai, in.JamSelesai, in.Mapel, in.GuruID, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	respond(w, 201, map[string]string{"ok": "1"})
}

// ---------- USERS ----------

func handleListUsers(w http.ResponseWriter, r *http.Request, c *Claims) {
	// platform admin lihat semua; sekolah lain hanya user-nya sendiri
	q := `SELECT id,nama,email,role,COALESCE(nip,''), COALESCE(sekolah_id,0) FROM users`
	args := []interface{}{}
	if c.Role != "platform" {
		q += ` WHERE sekolah_id=$1`
		args = append(args, c.SekolahID)
	}
	q += ` ORDER BY role, nama`
	rows, err := db.Query(q, args...)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id, sid int
		var nama, email, role, nip string
		rows.Scan(&id, &nama, &email, &role, &nip, &sid)
		list = append(list, map[string]interface{}{"id": id, "nama": nama, "email": email, "role": role, "nip": nip})
	}
	respond(w, 200, list)
}

func handleCreateUser(w http.ResponseWriter, r *http.Request, c *Claims) {
	if msg := cekKuota(c.SekolahID, "user"); msg != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(402)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}
	var in struct {
		Nama, Email, Password, Role, NIP string
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.Nama == "" || in.Email == "" || in.Password == "" {
		jsonErr(w, 400, "Nama, email, password wajib")
		return
	}
	if in.Role != "kepala_sekolah" && in.Role != "guru" && in.Role != "tu" {
		jsonErr(w, 400, "Role tidak valid")
		return
	}
	// Hanya kepala sekolah boleh membuat akun kepala sekolah lain
	if in.Role == "kepala_sekolah" && c.Role != "kepala_sekolah" {
		jsonErr(w, 403, "Hanya kepala sekolah yang dapat membuat akun kepala sekolah")
		return
	}
	_, err := db.Exec(`INSERT INTO users (nama,email,password,role,nip,sekolah_id) VALUES ($1,$2,$3,$4,NULLIF($5,'-'),$6)`,
		in.Nama, strings.ToLower(in.Email), hashPass(in.Password), in.Role, in.NIP, c.SekolahID)
	if err != nil {
		jsonErr(w, 409, userErr(err))
		return
	}
	respond(w, 201, map[string]string{"ok": "1"})
}

// ---------- DASHBOARD ----------

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	sid := c.SekolahID
	var totalSiswa, totalGuru, hadirHariIni, sppBelum int
	var saldo float64
	db.QueryRow(`SELECT COUNT(*) FROM siswa WHERE status='aktif' AND sekolah_id=$1`, sid).Scan(&totalSiswa)
	db.QueryRow(`SELECT COUNT(*) FROM users WHERE sekolah_id=$1`, sid).Scan(&totalGuru)
	db.QueryRow(`SELECT COUNT(*) FROM absensi_siswa WHERE tanggal=CURRENT_DATE AND status='hadir' AND sekolah_id=$1`, sid).Scan(&hadirHariIni)
	db.QueryRow(`SELECT COUNT(*) FROM spp WHERE status='belum' AND sekolah_id=$1`, sid).Scan(&sppBelum)
	db.QueryRow(`SELECT COALESCE(SUM(jumlah),0) FROM spp WHERE status='lunas' AND sekolah_id=$1`, sid).Scan(&saldo)

	var hafalanLancar, jenisSekolah int
	db.QueryRow(`SELECT COUNT(*), CASE WHEN COALESCE((SELECT jenis FROM sekolah WHERE id=$1),'umum')='islam' THEN 1 ELSE 0 END
		FROM hafalan WHERE status='lancar' AND tanggal >= CURRENT_DATE - 7 AND sekolah_id=$1`, sid).Scan(&hafalanLancar, &jenisSekolah)

	// Pengingat input harian (ronde 2 persona Ust. Soleh: "anak buah malas input").
	// Target GURU/admin yang input, bukan pengasuh: pesan singkat di dashboard.
	var absenHariIni, hafalHariIni int
	db.QueryRow(`SELECT COUNT(*) FROM absensi_siswa WHERE tanggal=CURRENT_DATE AND sekolah_id=$1`, sid).Scan(&absenHariIni)
	db.QueryRow(`SELECT COUNT(*) FROM hafalan WHERE tanggal=CURRENT_DATE AND sekolah_id=$1`, sid).Scan(&hafalHariIni)
	pengingat := ""
	if totalSiswa > 0 {
		hari := map[time.Weekday]string{time.Sunday:"Minggu", time.Monday:"Senin", time.Tuesday:"Selasa", time.Wednesday:"Rabu", time.Thursday:"Kamis", time.Friday:"Jumat", time.Saturday:"Sabtu"}[time.Now().Weekday()]
		if absenHariIni == 0 && (jenisSekolah == 0 || hafalHariIni == 0) {
			pengingat = "Belum ada input hari ini (" + hari + "). Catat absensi siswa dulu, biar laporan orang tua tetap lengkap. 💪"
		} else if absenHariIni == 0 {
			pengingat = "Input lain sudah dicatat ✅ — absensi hari ini (" + hari + ") belum. Jangan lupa!"
		} else if hafalHariIni == 0 && jenisSekolah == 1 && c.Role == "guru" {
			pengingat = "Absensi sudah dicatat ✅ — setoran hafalan hari ini (" + hari + ") belum ada."
		}
	}

	perKelas := []map[string]interface{}{}
	rows, err := db.Query(`SELECT kelas, COUNT(*) FROM siswa WHERE status='aktif' AND sekolah_id=$1 GROUP BY kelas ORDER BY kelas`, sid)
	if err == nil {
		for rows.Next() {
			var kelas string
			var n int
			rows.Scan(&kelas, &n)
			perKelas = append(perKelas, map[string]interface{}{"kelas": kelas, "jumlah": n})
		}
		rows.Close()
	}

	out := map[string]interface{}{
		"role": c.Role, "nama": c.Nama,
		"total_siswa": totalSiswa, "total_guru": totalGuru,
		"hadir_hari_ini": hadirHariIni, "spp_belum_lunas": sppBelum,
		"saldo": saldo, "per_kelas": perKelas,
		"pengingat": pengingat,
	}
	if jenisSekolah == 1 {
		out["hafalan_lancar_minggu_ini"] = hafalanLancar
	}
	respond(w, 200, out)
}

// GET /api/mapel — daftar mapel sekolah (dari tabel mapel; fallback: daftar statis)
func handleListMapel(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	rows, err := db.Query(`SELECT nama, COALESCE(jenjang,'') FROM mapel WHERE sekolah_id=$1 ORDER BY id`, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]string{}
	for rows.Next() {
		var nama, j string
		rows.Scan(&nama, &j)
		list = append(list, map[string]string{"nama": nama, "jenjang": j})
	}
	respond(w, 200, list)
}
