package main

import (
	"fmt"
	"os"
	"strings"
)

func migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			nama TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			role TEXT NOT NULL CHECK (role IN ('kepala_sekolah','guru','tu','platform')),
			nip TEXT,
			created_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS siswa (
			id SERIAL PRIMARY KEY,
			nis TEXT,
			nisn TEXT,
			nama TEXT NOT NULL,
			jk TEXT CHECK (jk IN ('L','P')),
			kelas TEXT NOT NULL,
			tgl_lahir DATE,
			alamat TEXT,
			nama_ortu TEXT,
			hp_ortu TEXT,
			status TEXT DEFAULT 'aktif' CHECK (status IN ('aktif','lulus','pindah','keluar')),
			created_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS guru_mapel (
			id SERIAL PRIMARY KEY,
			guru_id INT REFERENCES users(id),
			mapel TEXT NOT NULL,
			kelas TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS absensi_siswa (
			id SERIAL PRIMARY KEY,
			siswa_id INT REFERENCES siswa(id),
			tanggal DATE NOT NULL DEFAULT CURRENT_DATE,
			status TEXT NOT NULL CHECK (status IN ('hadir','sakit','izin','alpa')),
			keterangan TEXT,
			pencatat INT REFERENCES users(id),
			UNIQUE(siswa_id, tanggal)
		)`,
		`CREATE TABLE IF NOT EXISTS absensi_guru (
			id SERIAL PRIMARY KEY,
			guru_id INT REFERENCES users(id),
			tanggal DATE NOT NULL DEFAULT CURRENT_DATE,
			jam_masuk TIME,
			jam_pulang TIME,
			status TEXT NOT NULL CHECK (status IN ('hadir','sakit','izin','dinas_luar','alpa')),
			keterangan TEXT,
			UNIQUE(guru_id, tanggal)
		)`,
		`CREATE TABLE IF NOT EXISTS nilai (
			id SERIAL PRIMARY KEY,
			siswa_id INT REFERENCES siswa(id),
			mapel TEXT NOT NULL,
			kelas TEXT NOT NULL,
			jenis TEXT NOT NULL CHECK (jenis IN ('tugas','uh','pts','pas')),
			semester TEXT NOT NULL CHECK (semester IN ('ganjil','genap')),
			tahun_ajaran TEXT NOT NULL,
			nilai NUMERIC(5,2) NOT NULL,
			guru_id INT REFERENCES users(id),
			created_at TIMESTAMPTZ DEFAULT now(),
			UNIQUE(siswa_id, mapel, jenis, semester, tahun_ajaran)
		)`,
		`CREATE TABLE IF NOT EXISTS hafalan (
			id SERIAL PRIMARY KEY,
			siswa_id INT REFERENCES siswa(id),
			surah TEXT NOT NULL,
			ayat_dari INT,
			ayat_sampai INT,
			status TEXT NOT NULL CHECK (status IN ('baru','proses','lancar','murojaah')),
			penyetor_guru_id INT REFERENCES users(id),
			tanggal DATE NOT NULL DEFAULT CURRENT_DATE,
			catatan TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS spp (
			id SERIAL PRIMARY KEY,
			siswa_id INT REFERENCES siswa(id),
			bulan TEXT NOT NULL,
			tahun TEXT NOT NULL,
			jumlah NUMERIC(12,2) NOT NULL,
			jatuh_tempo DATE,
			status TEXT DEFAULT 'belum' CHECK (status IN ('belum','lunas','sebagian')),
			tgl_bayar DATE,
			metode TEXT,
			petugas_id INT REFERENCES users(id),
			UNIQUE(siswa_id, bulan, tahun)
		)`,
		`CREATE TABLE IF NOT EXISTS pengeluaran (
			id SERIAL PRIMARY KEY,
			tanggal DATE NOT NULL DEFAULT CURRENT_DATE,
			uraian TEXT NOT NULL,
			kategori TEXT,
			jumlah NUMERIC(12,2) NOT NULL,
			pencatat_id INT REFERENCES users(id),
			created_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS pengumuman (
			id SERIAL PRIMARY KEY,
			judul TEXT NOT NULL,
			isi TEXT NOT NULL,
			target TEXT DEFAULT 'semua' CHECK (target IN ('semua','guru','siswa','tu')),
			pembuat_id INT REFERENCES users(id),
			created_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS agenda (
			id SERIAL PRIMARY KEY,
			judul TEXT NOT NULL,
			deskripsi TEXT,
			tanggal DATE NOT NULL,
			waktu TIME,
			lokasi TEXT,
			pembuat_id INT REFERENCES users(id),
			created_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS jadwal (
			id SERIAL PRIMARY KEY,
			kelas TEXT NOT NULL,
			hari TEXT NOT NULL CHECK (hari IN ('Senin','Selasa','Rabu','Kamis','Jumat','Sabtu')),
			jam_mulai TIME NOT NULL,
			jam_selesai TIME NOT NULL,
			mapel TEXT NOT NULL,
			guru_id INT REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS spp_setting (
			id SERIAL PRIMARY KEY CHECK (id = 1),
			jumlah_default NUMERIC(12,2) DEFAULT 45000
		)`,
		`INSERT INTO spp_setting (id) VALUES (1) ON CONFLICT DO NOTHING`,
		// SPP default kini per-sekolah: pengaturan.spp_default (migrasi 2026-11)
		`ALTER TABLE pengaturan ADD COLUMN IF NOT EXISTS spp_default NUMERIC(12,2) NOT NULL DEFAULT 45000`,
		`CREATE TABLE IF NOT EXISTS pengaturan (
			id SERIAL PRIMARY KEY,
			nama_sekolah TEXT DEFAULT 'Griya Qur''an Tunas Ilmu',
			alamat TEXT DEFAULT '',
			npsn TEXT DEFAULT '',
			kepala_sekolah TEXT DEFAULT '',
			tahun_ajaran_aktif TEXT DEFAULT '2026/2027',
			semester_aktif TEXT DEFAULT 'ganjil'
		)`,
		`INSERT INTO pengaturan (id, nama_sekolah) VALUES (1, 'Sekolah Saya') ON CONFLICT DO NOTHING`,
		// pengaturan per sekolah (id = sekolah.id) dibuat otomatis di handleDaftar
		`DROP TABLE IF EXISTS login_fails`,
		`CREATE TABLE IF NOT EXISTS login_fails (
			key TEXT PRIMARY KEY,
			count INT NOT NULL DEFAULT 0,
			locked_until TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01'
		)`,
		`CREATE TABLE IF NOT EXISTS activity_log (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES users(id),
			aksi TEXT NOT NULL,
			detail TEXT,
			waktu TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS rapor_catatan (
			id SERIAL PRIMARY KEY,
			siswa_id INT REFERENCES siswa(id),
			tahun_ajaran TEXT NOT NULL,
			catatan_wali TEXT DEFAULT '',
			ekstrakurikuler TEXT DEFAULT '',
			prestasi TEXT DEFAULT '',
			wali_id INT REFERENCES users(id),
			UNIQUE(siswa_id, tahun_ajaran)
		)`,
		`CREATE TABLE IF NOT EXISTS mapel (
			id SERIAL PRIMARY KEY,
			sekolah_id INT NOT NULL REFERENCES sekolah(id) ON DELETE CASCADE,
			nama TEXT NOT NULL,
			jenjang TEXT,
			UNIQUE(sekolah_id, nama)
		)`,
		`CREATE TABLE IF NOT EXISTS kelas_tingkat (
			id SERIAL PRIMARY KEY,
			sekolah_id INT NOT NULL REFERENCES sekolah(id) ON DELETE CASCADE,
			tingkat TEXT NOT NULL,
			UNIQUE(sekolah_id, tingkat)
		)`,
		`CREATE TABLE IF NOT EXISTS wali_token (
			id SERIAL PRIMARY KEY,
			siswa_id INT NOT NULL REFERENCES siswa(id),
			sekolah_id INT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			kadaluarsa TIMESTAMPTZ NOT NULL,
			dibuat TIMESTAMPTZ DEFAULT now()
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("%q: %w", s[:60], err)
		}
	}
	return nil
}

func seed() error {
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	if n > 0 {
		return nil
	}
	if os.Getenv("SEED_DEMO") == "1" {
		return seedDemo()
	}
	email := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	pass := os.Getenv("ADMIN_PASSWORD")
	if email == "" || pass == "" {
		return fmt.Errorf("tabel users masih kosong: set SEED_DEMO=1 untuk menanam data demo, atau set ADMIN_EMAIL dan ADMIN_PASSWORD untuk membuat akun kepala sekolah pertama")
	}
	if len(pass) < 8 {
		return fmt.Errorf("ADMIN_PASSWORD minimal 8 karakter")
	}
	_, err := db.Exec(`INSERT INTO users (nama,email,password,role,sekolah_id) VALUES ('Kepala Sekolah', $1, $2, 'kepala_sekolah', 1)`,
		strings.ToLower(email), hashPass(pass))
	return err
}

func seedDemo() error {
	users := []struct {
		nama, email, pass, role, nip string
	}{
		{"Budi Santoso, M.Pd", "kepala@simaka.id", hashPass("kepala123"), "kepala_sekolah", "197001012000121001"},
		{"Siti Aminah, S.Pd", "guru@simaka.id", hashPass("guru123"), "guru", "198505152010012002"},
		{"Rina Dewi", "tu@simaka.id", hashPass("tu123"), "tu", "-"},
	}
	for _, u := range users {
		if _, err := db.Exec(`INSERT INTO users (nama,email,password,role,nip) VALUES ($1,$2,$3,$4,$5)`,
			u.nama, u.email, u.pass, u.role, u.nip); err != nil {
			return err
		}
	}
	siswa := []struct {
		nis, nisn, nama, jk, kelas, ortu, hp string
	}{
		{"2401", "0051234501", "Ahmad Fauzan", "L", "1A", "Bapak Hasan", "081200000001"},
		{"2402", "0051234502", "Aisyah Putri", "P", "1A", "Bapak Ridwan", "081200000002"},
		{"2403", "0051234503", "Muhammad Rizki", "L", "2A", "Ibu Fatimah", "081200000003"},
		{"2404", "0051234504", "Khadijah Salma", "P", "2A", "Bapak Ali", "081200000004"},
		{"2405", "0051234505", "Umar Abdillah", "L", "3A", "Bapak Yusuf", "081200000005"},
	}
	for _, s := range siswa {
		if _, err := db.Exec(`INSERT INTO siswa (nis,nisn,nama,jk,kelas,nama_ortu,hp_ortu) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			s.nis, s.nisn, s.nama, s.jk, s.kelas, s.ortu, s.hp); err != nil {
			return err
		}
	}
	db.Exec(`INSERT INTO jadwal (kelas,hari,jam_mulai,jam_selesai,mapel,guru_id) VALUES
		('1A','Senin','07:00','08:30','Matematika',2),
		('1A','Senin','08:30','10:00','Bahasa Indonesia',2),
		('2A','Senin','07:00','08:30','IPA',2)`)
	db.Exec(`INSERT INTO pengumuman (judul,isi,target,pembuat_id) VALUES
		('Selamat Datang di SchoolVia','Sistem Admin Sekolah siap digunakan. Silakan eksplorasi menu sesuai peran Anda.','semua',1)`)
	return nil
}
