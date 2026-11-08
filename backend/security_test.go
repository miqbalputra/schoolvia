package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAuthMiddlewareRejectsQueryToken(t *testing.T) {
	jwtSecret = []byte("test-secret-paling-sedikit-32-byte-panjang")
	u := &User{ID: 1, Nama: "Tes", Role: "guru", SekolahID: 1}
	tok, err := makeToken(u)
	if err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := authMiddleware(next)

	req := httptest.NewRequest("GET", "/api/me?token="+tok, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("token via query string harus ditolak (401), dapat %d", rec.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/me", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("token via header harus diterima (200), dapat %d", rec2.Code)
	}

	req3 := httptest.NewRequest("GET", "/api/me", nil)
	req3.AddCookie(&http.Cookie{Name: "sv_session", Value: tok})
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("token via cookie httpOnly harus diterima (200), dapat %d", rec3.Code)
	}
}

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{
		"=1+1":     "'=1+1",
		"+cmd":     "'+cmd",
		"@SUM(A1)": "'@SUM(A1)",
		"\t=1":     "'\t=1",
		"nama":     "nama",
		"-500":     "-500", // angka negatif tidak boleh rusak
		"":         "",
	}
	for in, want := range cases {
		if got := csvSafe(in); got != want {
			t.Errorf("csvSafe(%q) = %q, ingin %q", in, got, want)
		}
	}
}

func TestIsDigits(t *testing.T) {
	for _, ok := range []string{"6", "12", "007"} {
		if !isDigits(ok) {
			t.Errorf("isDigits(%q) harus true", ok)
		}
	}
	for _, bad := range []string{"", "%", "_", "6A", "6 OR 1=1", "7\n"} {
		if isDigits(bad) {
			t.Errorf("isDigits(%q) harus false", bad)
		}
	}
}

func TestDaftarRateLimit(t *testing.T) {
	daftarMu.Lock()
	daftarHits = map[string][]time.Time{}
	daftarMu.Unlock()
	for i := 0; i < daftarMaxPerHour; i++ {
		if !daftarRateLimit("1.2.3.4") {
			t.Fatalf("hit ke-%d harus diizinkan", i+1)
		}
	}
	if daftarRateLimit("1.2.3.4") {
		t.Fatal("hit melebihi batas harus ditolak")
	}
	if !daftarRateLimit("5.6.7.8") {
		t.Fatal("IP lain tidak boleh ikut terblokir")
	}
}

func TestClientIP(t *testing.T) {
	os.Unsetenv("TRUSTED_PROXY")
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "9.9.9.9, 8.8.8.8")
	if got := clientIP(req); got != "10.0.0.1" {
		t.Errorf("tanpa TRUSTED_PROXY, XFF harus diabaikan; dapat %s", got)
	}
	os.Setenv("TRUSTED_PROXY", "true")
	defer os.Unsetenv("TRUSTED_PROXY")
	if got := clientIP(req); got != "8.8.8.8" {
		t.Errorf("dengan TRUSTED_PROXY, pakai entry XFF terakhir; dapat %s", got)
	}
}

func TestEscapeLike(t *testing.T) {
	if got := escapeLike(`100%_a\b`); got != `100\%\_a\\b` {
		t.Errorf("escapeLike salah: %q", got)
	}
	if got := escapeLike("biasa"); got != "biasa" {
		t.Errorf("escapeLike tidak boleh mengubah teks biasa: %q", got)
	}
}

func TestValidTanggal(t *testing.T) {
	for _, ok := range []string{"2026-11-08", "2000-01-01"} {
		if !validTanggal(ok) {
			t.Errorf("validTanggal(%q) harus true", ok)
		}
	}
	for _, bad := range []string{"", "2026-11-8", "08-11-2026", "2026/11/08", "2026-13-99", "abc"} {
		if validTanggal(bad) {
			t.Errorf("validTanggal(%q) harus false", bad)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	csp := rec.Header().Get("Content-Security-Policy")
	for _, want := range []string{"object-src 'none'", "base-uri 'self'", "form-action 'self'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP kekurangan %q: %s", want, csp)
		}
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options harus DENY")
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options harus nosniff")
	}
}
