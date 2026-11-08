package main

import "fmt"

// migrateTenants: kolom sekolah_id di semua tabel data + tabel sekolah & langganan.
// Aman dijalankan berulang (IF NOT EXISTS / DO blocks).
func migrateTenants() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sekolah (
			id SERIAL PRIMARY KEY,
			subdomain TEXT UNIQUE NOT NULL,
			nama TEXT NOT NULL,
			alamat TEXT DEFAULT '',
			npsn TEXT DEFAULT '',
			kepala TEXT DEFAULT '',
			status TEXT DEFAULT 'trial' CHECK (status IN ('trial','aktif','suspend')),
			trial_sampai TIMESTAMPTZ DEFAULT now() + INTERVAL '14 days',
			created_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS langganan (
			id SERIAL PRIMARY KEY,
			sekolah_id INT REFERENCES sekolah(id),
			paket TEXT NOT NULL CHECK (paket IN ('gratis','sekolah','yayasan')),
			mulai TIMESTAMPTZ DEFAULT now(),
			sampai TIMESTAMPTZ,
			status TEXT DEFAULT 'aktif' CHECK (status IN ('aktif','berakhir')),
			catatan TEXT DEFAULT ''
		)`,
		// Kolom sekolah_id (default 1 = instalasi utama)
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INT NOT NULL DEFAULT 0`,
		`ALTER TABLE siswa DROP CONSTRAINT IF EXISTS siswa_nis_key`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_siswa_nis_per_sekolah ON siswa (sekolah_id, nis) WHERE nis IS NOT NULL`,
		`ALTER TABLE siswa ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE pengumuman ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE agenda ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE jadwal ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE pengeluaran ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE absensi_siswa ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE absensi_guru ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE nilai ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE hafalan ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE spp ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`ALTER TABLE activity_log ADD COLUMN IF NOT EXISTS sekolah_id INT DEFAULT 1`,
		`CREATE INDEX IF NOT EXISTS idx_siswa_sekolah ON siswa(sekolah_id)`,
		`CREATE INDEX IF NOT EXISTS idx_spp_sekolah ON spp(sekolah_id)`,
		// Registrasi instalasi utama sebagai sekolah 1, lalu majukan sequence
		`INSERT INTO sekolah (id, subdomain, nama, status)
		 VALUES (1, 'utama', 'Griya Quran Tunas Ilmu', 'aktif') ON CONFLICT (id) DO NOTHING`,
		`SELECT setval('sekolah_id_seq', (SELECT COALESCE(MAX(id),1) FROM sekolah))`,
		// Pengaturan lama dipetakan ke sekolah 1
		`UPDATE users SET sekolah_id=1 WHERE sekolah_id IS NULL`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("tenant: %q: %w", s[:50], err)
		}
	}
	return nil
}
