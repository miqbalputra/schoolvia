package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://simaka:simaka123@localhost:5432/simaka?sslmode=disable"
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
		for range time.Tick(6 * time.Hour) {
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
	if err := seed(); err != nil {
		log.Fatal("seed: ", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux)
	mux.HandleFunc("GET /assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/"+strings.TrimPrefix(r.URL.Path, "/"))
	}))
	mux.HandleFunc("GET /icons/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/"+strings.TrimPrefix(r.URL.Path, "/"))
	}))
	mux.HandleFunc("GET /fonts/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/"+strings.TrimPrefix(r.URL.Path, "/"))
	}))
	// Landing page promosi di /, aplikasi di /app/
	mux.Handle("GET /{$}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/landing.html")
	}))
	mux.Handle("GET /app", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/index.html")
	}))
	mux.Handle("GET /pitch", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/pitch.html")
	}))
	mux.Handle("GET /app/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/index.html")
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      cors(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	fmt.Println("SchoolVia API running on :" + port)
	log.Fatal(srv.ListenAndServe())
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data:; script-src 'self' 'unsafe-inline' 'unsafe-eval'")
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
