package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ---------- PLAN LIMITS (Free / Sekolah / Yayasan) ----------

// planOf: paket aktif sekolah (trial dihitung = semua fitur terbuka selama trial)
func planOf(sekolahID int) string {
	var status, paket string
	if err := db.QueryRow(`SELECT s.status, l.paket FROM sekolah s
		LEFT JOIN langganan l ON l.sekolah_id=s.id AND l.status='aktif'
		WHERE s.id=$1`, sekolahID).Scan(&status, &paket); err != nil {
		return "gratis"
	}
	if status == "trial" {
		return "sekolah" // trial = full fitur
	}
	if paket == "" {
		paket = "gratis"
	}
	return paket
}

type planLimits struct {
	MaxSiswa, MaxUser int
	Export, Rekap     bool
}

func limitsFor(paket string) planLimits {
	switch paket {
	case "yayasan":
		return planLimits{MaxSiswa: 0, MaxUser: 0, Export: true, Rekap: true} // 0 = unlimited
	case "sekolah":
		return planLimits{MaxSiswa: 150, MaxUser: 20, Export: true, Rekap: true}
	default:
		return planLimits{MaxSiswa: 30, MaxUser: 3, Export: false, Rekap: false}
	}
}

// planMW: middleware fitur — tolak 402 kalau plan tidak termasuk fitur
func planMW(fitur string, h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := userFromContext(r)
		if c == nil {
			jsonErr(w, 401, "Unauthorized")
			return
		}
		lim := limitsFor(planOf(c.SekolahID))
		allowed := false
		switch fitur {
		case "export":
			allowed = lim.Export
		case "rekap":
			allowed = lim.Rekap
		}
		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(402)
			json.NewEncoder(w).Encode(map[string]string{
				"error":  "Fitur ini tersedia di paket Sekolah atau Yayasan. Silakan upgrade.",
				"upgrade": "https://sistemflow.com",
			})
			return
		}
		h(w, r)
	})
}

// planMWClaims: seperti planMW tapi untuk handler dengan Claims
func planMWClaims(fitur string, h func(http.ResponseWriter, *http.Request, *Claims)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := userFromContext(r)
		if c == nil {
			jsonErr(w, 401, "Unauthorized")
			return
		}
		lim := limitsFor(planOf(c.SekolahID))
		allowed := false
		switch fitur {
		case "export":
			allowed = lim.Export
		case "rekap":
			allowed = lim.Rekap
		}
		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(402)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Fitur ini tersedia di paket Sekolah atau Yayasan. Silakan upgrade.",
				"upgrade": "https://sistemflow.com",
			})
			return
		}
		h(w, r, c)
	})
}

// cekKuota: return pesan error atau "" — untuk POST siswa/users
func cekKuota(sekolahID int, jenis string) string {
	lim := limitsFor(planOf(sekolahID))
	if jenis == "siswa" && lim.MaxSiswa > 0 {
		var n int
		db.QueryRow(`SELECT COUNT(*) FROM siswa WHERE sekolah_id=$1 AND status='aktif'`, sekolahID).Scan(&n)
		if n >= lim.MaxSiswa {
			return "Paket Gratis dibatasi " + itoa(lim.MaxSiswa) + " siswa aktif. Upgrade ke paket Sekolah untuk lebih banyak."
		}
	}
	if jenis == "user" && lim.MaxUser > 0 {
		var n int
		db.QueryRow(`SELECT COUNT(*) FROM users WHERE sekolah_id=$1`, sekolahID).Scan(&n)
		if n >= lim.MaxUser {
			return "Paket Gratis dibatasi " + itoa(lim.MaxUser) + " pengguna. Upgrade ke paket Sekolah untuk lebih banyak."
		}
	}
	return ""
}

// cekLangganan: sekolah harus aktif/trial yang belum habis.
func cekLangganan(sekolahID int) (bool, string) {
	var status string
	var trialSampai time.Time
	err := db.QueryRow(`SELECT status, trial_sampai FROM sekolah WHERE id=$1`, sekolahID).Scan(&status, &trialSampai)
	if err != nil {
		return false, "Sekolah tidak ditemukan"
	}
	if status == "suspend" {
		return false, "Langganan sekolah Anda ditangguhkan. Hubungi admin."
	}
	if status == "trial" && time.Now().After(trialSampai) {
		return false, "Masa trial 14 hari sudah berakhir. Upgrade ke paket Sekolah untuk melanjutkan."
	}
	// langganan habis (bukan suspend): TIDAK mengunci data — mode baca + unduh tetap bebas (anti-sandera)
	return true, ""
}

// readMode: true jika langganan habis (akses baca + unduh saja, input ditolak dengan pesan lembut)
func readMode(sekolahID int) bool {
	var status string
	var trialSampai time.Time
	err := db.QueryRow(`SELECT status, trial_sampai FROM sekolah WHERE id=$1`, sekolahID).Scan(&status, &trialSampai)
	if err != nil {
		return false
	}
	if status == "suspend" {
		return true
	}
	if status == "trial" && time.Now().After(trialSampai) {
		return true
	}
	if status == "kadarluarsa" {
		return true
	}
	return false
}

// ---------- ONBOARDING: daftar sekolah baru ----------

func handleDaftar(w http.ResponseWriter, r *http.Request) {
	var in struct {
		NamaSekolah string `json:"nama_sekolah"`
		Subdomain   string `json:"subdomain"`
		NamaAdmin   string `json:"nama_admin"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		IsiDemo     *bool  `json:"isi_demo"`
		Jenis       string `json:"jenis"`     // 'umum' | 'islam' (default umum)
		Jenjang     string `json:"jenjang"`   // SD | SMP | SMA (sekolah umum)
		KodeRef     string `json:"kode_ref"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if !daftarRateLimit(clientIP(r)) {
		jsonErr(w, 429, "Terlalu banyak pendaftaran dari alamat ini. Coba lagi dalam satu jam.")
		return
	}
	// default: tanpa data demo; aktif bila server jalan dengan SEED_DEMO=1
	isiDemo := in.IsiDemo != nil && *in.IsiDemo
	if in.IsiDemo == nil && os.Getenv("SEED_DEMO") == "1" {
		isiDemo = true
	}
	in.Subdomain = strings.ToLower(strings.TrimSpace(in.Subdomain))
	if in.NamaSekolah == "" || in.Subdomain == "" || in.NamaAdmin == "" || in.Email == "" || len(in.Password) < 8 {
		jsonErr(w, 400, "Semua field wajib (password min 8 karakter)")
		return
	}
	for _, ch := range in.Subdomain {
		if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-') {
			jsonErr(w, 400, "Subdomain hanya boleh huruf kecil, angka, dan tanda minus")
			return
		}
	}
	var exists int
	db.QueryRow(`SELECT COUNT(*) FROM sekolah WHERE subdomain=$1`, in.Subdomain).Scan(&exists)
	if exists > 0 {
		jsonErr(w, 400, "Subdomain sudah dipakai, pilih yang lain")
		return
	}
	db.QueryRow(`SELECT COUNT(*) FROM users WHERE email=$1`, strings.ToLower(in.Email)).Scan(&exists)
	if exists > 0 {
		jsonErr(w, 400, "Email sudah terdaftar")
		return
	}
	jenis := in.Jenis
	if jenis != "islam" {
		jenis = "umum"
	}
	var sid int
	refTok, _ := waliTokenNew()
	kodeRef := "REF" + strings.ToUpper(refTok[:12])
	jenjang := strings.ToUpper(strings.TrimSpace(in.Jenjang))
	if jenjang != "SD" && jenjang != "SMP" && jenjang != "SMA" {
		jenjang = ""
	}
	err := db.QueryRow(`INSERT INTO sekolah (subdomain, nama, status, jenis, jenjang, kode_ref, dirujuk_oleh) VALUES ($1,$2,'trial',$3,$4,$5,NULLIF($6,'')) RETURNING id`,
		in.Subdomain, in.NamaSekolah, jenis, jenjang, kodeRef, strings.TrimSpace(in.KodeRef)).Scan(&sid)
	if err != nil {
		// fallback tanpa kolom rujukan (instalasi lama)
		err = db.QueryRow(`INSERT INTO sekolah (subdomain, nama, status) VALUES ($1,$2,'trial') RETURNING id`, in.Subdomain, in.NamaSekolah).Scan(&sid)
	}
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	// Program rujukan: pendaftar via kode → perujuk dapat 1 bulan gratis (plan Sekolah)
	if ref := strings.TrimSpace(in.KodeRef); ref != "" {
		var refID int
		if db.QueryRow(`SELECT id FROM sekolah WHERE kode_ref=$1 AND id<>$2`, ref, sid).Scan(&refID) == nil {
			var paket, status string
			if db.QueryRow(`SELECT paket, status FROM langganan WHERE sekolah_id=$1 ORDER BY id DESC LIMIT 1`, refID).Scan(&paket, &status) == nil && paket != "yayasan" {
				// perpanjang/masukkan 1 bulan gratis
				var sampai time.Time
				e := db.QueryRow(`SELECT COALESCE(sampai, now()) FROM langganan WHERE sekolah_id=$1 ORDER BY id DESC LIMIT 1`, refID).Scan(&sampai)
				base := time.Now()
				if e == nil && sampai.After(base) {
					base = sampai
				}
				db.Exec(`INSERT INTO langganan (sekolah_id, paket, status, sampai, catatan) VALUES ($1,'sekolah','aktif',$2,'Bonus rujukan 1 bulan')`, refID, base.AddDate(0, 1, 0))
				logActivitySekolah(refID, "rujukan", "Sekolah ID "+itoa(refID)+" dapat bonus 1 bulan dari rujukan "+ref)
			}
		}
	}
	db.Exec(`INSERT INTO langganan (sekolah_id, paket, status, catatan) VALUES ($1,'gratis','aktif','Trial 14 hari')`, sid)
	// pengaturan per sekolah (kop kwitansi, tahun ajaran, dsb.)
	db.Exec(`INSERT INTO pengaturan (id, nama_sekolah, tahun_ajaran_aktif) VALUES ($1,$2,'2026/2027') ON CONFLICT DO NOTHING`, sid, in.NamaSekolah)
	// P2 pivot: sekolah umum → mapel default Kurikulum Merdeka sesuai jenjang
	if jenis == "umum" && jenjang != "" {
		seedKurikulumUmum(sid, jenjang)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	var uid int
	db.QueryRow(`INSERT INTO users (nama,email,password,role,sekolah_id) VALUES ($1,$2,$3,'kepala_sekolah',$4) RETURNING id`,
		in.NamaAdmin, strings.ToLower(in.Email), string(hash), sid).Scan(&uid)
	logActivity(uid, "daftar_sekolah", "Sekolah baru: "+in.NamaSekolah+" (subdomain: "+in.Subdomain+")")
	if isiDemo {
		seedDataDemo(sid, uid)
	}
	respond(w, 201, map[string]interface{}{
		"ok": "1", "sekolah_id": sid,
		"pesan": "Pendaftaran berhasil! Trial 14 hari aktif. Login di halaman masuk.",
	})
}

// ---------- STATUS LANGGANAN (untuk banner di UI) ----------

func handleLangganan(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	var status string
	var trialSampai time.Time
	db.QueryRow(`SELECT status, trial_sampai FROM sekolah WHERE id=$1`, c.SekolahID).Scan(&status, &trialSampai)
	sisa := int(time.Until(trialSampai).Hours() / 24)
	if sisa < 0 {
		sisa = 0
	}
	paket := planOf(c.SekolahID)
	lim := limitsFor(paket)
	// kuota terpakai
	var nSiswa, nUser int
	db.QueryRow(`SELECT COUNT(*) FROM siswa WHERE sekolah_id=$1 AND status='aktif'`, c.SekolahID).Scan(&nSiswa)
	db.QueryRow(`SELECT COUNT(*) FROM users WHERE sekolah_id=$1`, c.SekolahID).Scan(&nUser)
	respond(w, 200, map[string]interface{}{
		"status": status, "trial_sisa_hari": sisa,
		"plan": paket,
		"limit_siswa": lim.MaxSiswa, "limit_user": lim.MaxUser,
		"pakai_siswa": nSiswa, "pakai_user": nUser,
		"export": lim.Export, "rekap": lim.Rekap,
	})
}

// ---------- NOTIFIKASI WA (link wa.me siap kirim) ----------

func handleNotifSPP(w http.ResponseWriter, r *http.Request) {
	c := userFromContext(r)
	bulan := r.URL.Query().Get("bulan")
	tahun := r.URL.Query().Get("tahun")
	if bulan == "" || tahun == "" {
		jsonErr(w, 400, "Bulan dan tahun wajib")
		return
	}
	var namaSekolah string
	db.QueryRow(`SELECT nama FROM sekolah WHERE id=$1`, c.SekolahID).Scan(&namaSekolah)
	rows, err := db.Query(`SELECT s.nama, COALESCE(s.hp_ortu,''), sp.jumlah
		FROM spp sp JOIN siswa s ON s.id=sp.siswa_id
		WHERE sp.bulan=$1 AND sp.tahun=$2 AND sp.status='belum' AND s.sekolah_id=$3 AND s.hp_ortu <> ''`,
		bulan, tahun, c.SekolahID)
	if err != nil {
		jsonErr(w, 500, userErr(err))
		return
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var nama, hp string
		var jumlah float64
		rows.Scan(&nama, &hp, &jumlah)
		hp = "628" + strings.TrimPrefix(strings.TrimPrefix(hp, "08"), "62")
		hp = strings.ReplaceAll(hp, "-", "")
		msg := "Assalamualaikum Bapak/Ibu, kami informasikan bahwa SPP ananda " + nama +
			" bulan " + bulan + " " + tahun + " sebesar Rp" + itoa(int(jumlah)) +
			" di " + namaSekolah + " belum dibayarkan. Mohon pembayarannya. Terima kasih. 🙏"
		list = append(list, map[string]interface{}{"nama": nama, "hp": hp, "link_wa": "https://wa.me/" + hp + "?text=" + waEncode(msg)})
	}
	respond(w, 200, map[string]interface{}{"jumlah": len(list), "pesan": list})
}

func waEncode(s string) string {
	replacer := strings.NewReplacer(" ", "%20", "\n", "%0A", ",", "%2C", "'", "%27")
	return replacer.Replace(s)
}

// ---------- PWA MANIFEST ----------

func handleManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{
  "name": "SchoolVia — Sistem Admin Sekolah",
  "short_name": "SchoolVia",
  "start_url": "/app",
  "display": "standalone",
  "background_color": "#f4f6fb",
  "theme_color": "#4f46e5",
  "icons": [
  {"src": "/icons/icon-192.png", "sizes": "192x192", "type": "image/png"},
  {"src": "/icons/icon-512.png", "sizes": "512x512", "type": "image/png"}
  ]
  }`))
}
