package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		if os.Getenv("ALLOW_INSECURE_DEFAULT_DSN") == "1" {
			dsn = "postgres://simaka:simaka123@localhost:5432/simaka?sslmode=disable"
			log.Println("PERINGATAN: DATABASE_URL tidak di-set; memakai DSN default lokal yang TIDAK aman. Jangan dipakai di produksi.")
		} else {
			log.Fatal("DATABASE_URL wajib di-set, mis. postgres://user:pass@host:5432/namadb?sslmode=require (dev lokal: set ALLOW_INSECURE_DEFAULT_DSN=1)")
		}
	}
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(20)
	if err := db.Ping(); err != nil {
		log.Fatal("DB not reachable: ", err)
	}
	sec, err := loadOrCreateSecret()
	initEkskulSchema()
	if err != nil {
		log.Fatal("jwt secret: ", err)
	}
	jwtSecret = sec
	// rotasi log tiap 6 jam
	go func() {
		t := time.NewTicker(6 * time.Hour)
		defer t.Stop()
		for range t.C {
			rotateLog()
		}
	}()
	// laporan mingguan WA pengasuh: cek tiap jam, kirim sekali sehari setelah jam 20.00 WIB (13.00 UTC)
	go func() {
		sudahKirim := ""
		for {
			now := time.Now().UTC().Add(8 * time.Hour) // WIB
			hari := now.Format("2006-01-02")
			if now.Hour() >= 20 && sudahKirim != hari {
				sudahKirim = hari
				kirimLaporanSemua()
			}
			time.Sleep(30 * time.Minute)
		}
	}()
	if err := migrate(); err != nil {
		log.Fatal("migrate: ", err)
	}
	if err := migrateTenants(); err != nil {
		log.Fatal("migrate tenants: ", err)
	}
	if err := migrateShares(); err != nil {
		log.Fatal("migrate shares: ", err)
	}
	if err := seed(); err != nil {
		log.Fatal("seed: ", err)
	}

	frontDir = frontendDir()
	mux := http.NewServeMux()
	registerRoutes(mux)
	mux.HandleFunc("GET /assets/", serveStatic)
	mux.HandleFunc("GET /icons/", serveStatic)
	mux.HandleFunc("GET /fonts/", serveStatic)
	// Landing page promosi di /, aplikasi di /app/
	mux.Handle("GET /{$}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(frontDir, "landing.html"))
	}))
	mux.Handle("GET /app", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(frontDir, "index.html"))
	}))
	mux.Handle("GET /pitch", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(frontDir, "pitch.html"))
	}))
	mux.Handle("GET /app/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(frontDir, "index.html"))
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:           ":" + port,
		Handler:        cors(limitBody(mux)),
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   5 * time.Minute, // ekspor ZIP bisa lama untuk data besar
		IdleTimeout:    2 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Println("SchoolVia API running on :" + port)
	log.Fatal(srv.ListenAndServe())
}

var frontDir string

// frontendDir: lokasi frontend — dari env FRONTEND_DIR, lalu relatif terhadap executable
// (deployment umum: binary di backend/, frontend di ../frontend), terakhir CWD.
func frontendDir() string {
	if d := os.Getenv("FRONTEND_DIR"); d != "" {
		return d
	}
	if exe, err := os.Executable(); err == nil {
		d := filepath.Join(filepath.Dir(exe), "..", "frontend")
		if _, err := os.Stat(filepath.Join(d, "index.html")); err == nil {
			return d
		}
	}
	return "../frontend"
}

// serveStatic: sajikan file statis dari frontDir dengan penjagaan path traversal.
func serveStatic(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	if strings.Contains(p, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, filepath.Join(frontDir, p))
}

// limitBody: batasi ukuran body request (1 MB, 6 MB untuk import) agar tidak habis memori.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		max := int64(1 << 20)
		if strings.HasPrefix(r.URL.Path, "/api/import/") {
			max = 6 << 20
		}
		r.Body = http.MaxBytesReader(w, r.Body, max)
		next.ServeHTTP(w, r)
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data:; script-src 'self' 'unsafe-inline' 'unsafe-eval'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")
		// CORS hanya untuk origin yang diizinkan (same-origin tidak butuh CORS)
		origin := r.Header.Get("Origin")
		allowed := []string{"https://schoolvia.id", "https://www.schoolvia.id", "https://app.schoolvia.id", "https://app.sistemflow.com", "https://sistemflow.com"}
		for _, a := range allowed {
			if origin == a {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Vary", "Origin")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				break
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
