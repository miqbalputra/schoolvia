package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

// TestIntegrasiLintasTenant butuh PostgreSQL asli: set TEST_DATABASE_URL.
// CI menyediakannya via service container; tanpa env, test ini di-skip (unit test tetap jalan).
func TestIntegrasiLintasTenant(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL tidak di-set — lewati test integrasi")
	}
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { db.Close(); db = nil }()
	if err := db.Ping(); err != nil {
		t.Fatal("DB tidak bisa dihubungi: ", err)
	}
	jwtSecret = []byte("integration-test-secret-32-byte-panjang")

	if err := migrate(); err != nil {
		t.Fatal("migrate: ", err)
	}
	if err := migrateTenants(); err != nil {
		t.Fatal("migrateTenants: ", err)
	}
	if err := migrateShares(); err != nil {
		t.Fatal("migrateShares: ", err)
	}
	initEkskulSchema()

	mustExec := func(q string, args ...interface{}) {
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("fixture exec: %v\nquery: %s", err, q)
		}
	}
	qrow := func(q string, args ...interface{}) int {
		var id int
		if err := db.QueryRow(q, args...).Scan(&id); err != nil {
			t.Fatalf("fixture row: %v\nquery: %s", err, q)
		}
		return id
	}

	uidA := qrow(`INSERT INTO users (nama,email,password,role,sekolah_id) VALUES ('Kepala A','a@test.local',$1,'kepala_sekolah',1) RETURNING id`, hashPass("password123"))
	qrow(`INSERT INTO users (nama,email,password,role,sekolah_id) VALUES ('Guru G','g@test.local',$1,'guru',1) RETURNING id`, hashPass("password123"))
	sidB := qrow(`INSERT INTO sekolah (subdomain,nama,status) VALUES ('b','Sekolah B','trial') RETURNING id`)
	qrow(`INSERT INTO users (nama,email,password,role,sekolah_id) VALUES ('Kepala B','b@test.local',$1,'kepala_sekolah',$2) RETURNING id`, hashPass("password123"), sidB)
	mustExec(`INSERT INTO pengaturan (id, nama_sekolah) VALUES ($1,'Sekolah B')`, sidB)
	siswaA := qrow(`INSERT INTO siswa (nis,nama,kelas,sekolah_id) VALUES ('A1','Siswa A','1A',1) RETURNING id`)
	siswaB := qrow(`INSERT INTO siswa (nis,nama,kelas,sekolah_id) VALUES ('B1','Siswa B','1A',$1) RETURNING id`, sidB)
	mustExec(`INSERT INTO absensi_siswa (siswa_id,tanggal,status,sekolah_id) VALUES ($1,'2026-11-08','hadir',1)`, siswaA)
	sppA := qrow(`INSERT INTO spp (siswa_id,bulan,tahun,jumlah,status,sekolah_id) VALUES ($1,'November','2026',45000,'lunas',1) RETURNING id`, siswaA)
	logActivity(uidA, "login", "test A")
	logActivitySekolah(sidB, "login", "test B")

	mux := http.NewServeMux()
	registerRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	doReq := func(c *http.Client, method, path string, body interface{}) (int, string, http.Header) {
		var rdr *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		} else {
			rdr = bytes.NewReader(nil)
		}
		req, err := http.NewRequest(method, srv.URL+path, rdr)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		return resp.StatusCode, buf.String(), resp.Header
	}
	login := func(email string) *http.Client {
		jar, _ := cookiejar.New(nil)
		c := &http.Client{Jar: jar}
		code, body, hdr := doReq(c, "POST", "/api/login", map[string]string{"email": email, "password": "password123"})
		if code != 200 {
			t.Fatalf("login %s gagal: %d %s", email, code, body)
		}
		var lr map[string]interface{}
		json.Unmarshal([]byte(body), &lr)
		if _, ok := lr["token"]; ok {
			t.Error("response login tidak boleh mengandung field token (harus via cookie)")
		}
		sc := hdr.Get("Set-Cookie")
		if !strings.Contains(sc, "sv_session=") || !strings.Contains(sc, "HttpOnly") {
			t.Errorf("cookie sv_session harus ada dan HttpOnly, dapat: %s", sc)
		}
		return c
	}
	anon := &http.Client{}

	cliA := login("a@test.local")
	cliG := login("g@test.local")
	cliB := login("b@test.local")

	// --- A sebagai kepala sekolah 1 ---
	code, body, _ := doReq(cliA, "GET", "/api/siswa", nil)
	if code != 200 || !strings.Contains(body, "Siswa A") || strings.Contains(body, "Siswa B") {
		t.Errorf("GET /api/siswa (A): code=%d, harus memuat Siswa A saja. body=%s", code, body)
	}
	code, body, _ = doReq(cliA, "GET", "/api/absensi-siswa?tanggal=2026-11-08", nil)
	if code != 200 || !strings.Contains(body, "Siswa A") || !strings.Contains(body, "hadir") || strings.Contains(body, "Siswa B") {
		t.Errorf("GET /api/absensi-siswa (A): code=%d, harus memuat Siswa A saja. body=%s", code, body)
	}
	code, body, _ = doReq(cliA, "GET", "/api/activity-log", nil)
	if code != 200 || !strings.Contains(body, "test A") || strings.Contains(body, "test B") {
		t.Errorf("GET /api/activity-log (A): code=%d, harus memuat log sekolah 1 saja. body=%s", code, body)
	}
	// tulis lintas-tenant: hafalan untuk siswa milik sekolah B harus ditolak
	code, _, _ = doReq(cliA, "POST", "/api/hafalan", map[string]interface{}{"siswa_id": siswaB, "surah": "Al-Fatihah", "status": "baru"})
	if code != 404 {
		t.Errorf("POST /api/hafalan lintas-tenant harus 404, dapat %d", code)
	}
	// rapor-cetak lintas-tenant
	code, _, _ = doReq(cliA, "GET", "/api/rapor-cetak/"+itoa(siswaB), nil)
	if code != 404 {
		t.Errorf("GET /api/rapor-cetak siswa B oleh A harus 404, dapat %d", code)
	}
	code, _, _ = doReq(cliA, "GET", "/api/rapor-cetak/"+itoa(siswaA), nil)
	if code != 200 {
		t.Errorf("GET /api/rapor-cetak siswa A oleh A harus 200, dapat %d", code)
	}
	// kwitansi: A bisa, B tidak; share token bisa tanpa cookie; share invalid ditolak
	code, _, _ = doReq(cliA, "GET", "/api/kwitansi/"+itoa(sppA), nil)
	if code != 200 {
		t.Errorf("GET /api/kwitansi (A): harus 200, dapat %d", code)
	}
	code, _, _ = doReq(cliB, "GET", "/api/kwitansi/"+itoa(sppA), nil)
	if code != 404 {
		t.Errorf("GET /api/kwitansi spp A oleh B harus 404, dapat %d", code)
	}
	code, body, _ = doReq(cliA, "POST", "/api/kwitansi/"+itoa(sppA)+"/share", nil)
	if code != 200 {
		t.Fatalf("POST share kwitansi harus 200, dapat %d %s", code, body)
	}
	var sr map[string]string
	json.Unmarshal([]byte(body), &sr)
	share := sr["token"]
	if share == "" {
		t.Fatal("share token kosong")
	}
	code, _, _ = doReq(anon, "GET", "/api/kwitansi/"+itoa(sppA)+"?share="+share, nil)
	if code != 200 {
		t.Errorf("kwitansi via share token harus 200, dapat %d", code)
	}
	code, _, _ = doReq(anon, "GET", "/api/kwitansi/"+itoa(sppA)+"?share=salah", nil)
	if code != 401 {
		t.Errorf("kwitansi via share invalid harus 401, dapat %d", code)
	}
	code, _, _ = doReq(anon, "GET", "/api/kwitansi/"+itoa(sppA), nil)
	if code != 401 {
		t.Errorf("kwitansi tanpa sesi harus 401, dapat %d", code)
	}
	// token via query string harus ditolak
	code, _, _ = doReq(anon, "GET", "/api/siswa?token=apapun", nil)
	if code != 401 {
		t.Errorf("?token= di query harus ditolak (401), dapat %d", code)
	}
	// ekspor: kepala boleh, guru ditolak
	code, body, _ = doReq(cliA, "GET", "/api/export/siswa", nil)
	if code != 200 || !strings.Contains(body, "Siswa A") || strings.Contains(body, "Siswa B") {
		t.Errorf("GET /api/export/siswa (A): code=%d harus 200 & hanya Siswa A. body=%s", code, body)
	}
	code, _, _ = doReq(cliA, "GET", "/api/export/all", nil)
	if code != 200 {
		t.Errorf("GET /api/export/all (A): harus 200, dapat %d", code)
	}
	code, _, _ = doReq(cliG, "GET", "/api/export/siswa", nil)
	if code != 403 {
		t.Errorf("guru harus ditolak (403) di export/siswa, dapat %d", code)
	}
	code, _, _ = doReq(cliG, "GET", "/api/export/all", nil)
	if code != 403 {
		t.Errorf("guru harus ditolak (403) di export/all, dapat %d", code)
	}
	code, _, _ = doReq(cliG, "GET", "/api/activity-log", nil)
	if code != 403 {
		t.Errorf("guru harus ditolak (403) di activity-log, dapat %d", code)
	}
	// laporan wali: token per siswa, halaman publik
	code, body, _ = doReq(cliA, "GET", "/api/wali-token/"+itoa(siswaA), nil)
	if code != 200 {
		t.Fatalf("GET /api/wali-token harus 200, dapat %d", code)
	}
	var wt map[string]string
	json.Unmarshal([]byte(body), &wt)
	waliTok := wt["token"]
	code, body, _ = doReq(anon, "GET", "/api/laporan-wali/"+itoa(siswaA)+"?token="+waliTok, nil)
	if code != 200 || !strings.Contains(body, "Siswa A") || strings.Contains(body, "Siswa B") {
		t.Errorf("laporan-wali publik harus 200 & hanya Siswa A, dapat %d", code)
	}
	code, _, _ = doReq(anon, "GET", "/api/laporan-wali/"+itoa(siswaB)+"?token="+waliTok, nil)
	if code != 403 {
		t.Errorf("token wali A dipakai untuk siswa B harus 403, dapat %d", code)
	}
	code, _, _ = doReq(anon, "GET", "/api/laporan-wali/"+itoa(siswaA)+"?token=salah", nil)
	if code != 401 {
		t.Errorf("laporan-wali token invalid harus 401, dapat %d", code)
	}

	// --- B sebagai kepala sekolah 2: tidak boleh melihat apa pun milik sekolah 1 ---
	code, body, _ = doReq(cliB, "GET", "/api/siswa", nil)
	if code != 200 || !strings.Contains(body, "Siswa B") || strings.Contains(body, "Siswa A") {
		t.Errorf("GET /api/siswa (B): code=%d, harus memuat Siswa B saja. body=%s", code, body)
	}
	code, body, _ = doReq(cliB, "GET", "/api/absensi-siswa?tanggal=2026-11-08", nil)
	if code != 200 || strings.Contains(body, "Siswa A") || !strings.Contains(body, "Siswa B") {
		t.Errorf("GET /api/absensi-siswa (B): tidak boleh memuat Siswa A. code=%d body=%s", code, body)
	}
	code, body, _ = doReq(cliB, "GET", "/api/activity-log", nil)
	if code != 200 || strings.Contains(body, "test A") {
		t.Errorf("GET /api/activity-log (B): tidak boleh memuat log sekolah 1. body=%s", body)
	}
	code, _, _ = doReq(cliB, "GET", "/api/rapor-cetak/"+itoa(siswaA), nil)
	if code != 404 {
		t.Errorf("GET /api/rapor-cetak siswa A oleh B harus 404, dapat %d", code)
	}
}
