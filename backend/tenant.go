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
		// Kolom sekolah untuk pivot P1/P2 (sebelumnya tidak ada → INSERT handleDaftar selalu gagal ke fallback)
		`ALTER TABLE sekolah ADD COLUMN IF NOT EXISTS jenis TEXT NOT NULL DEFAULT 'islam' CHECK (jenis IN ('umum','islam'))`,
		`ALTER TABLE sekolah ADD COLUMN IF NOT EXISTS jenjang TEXT DEFAULT ''`,
		`ALTER TABLE sekolah ADD COLUMN IF NOT EXISTS kode_ref TEXT UNIQUE`,
		`ALTER TABLE sekolah ADD COLUMN IF NOT EXISTS dirujuk_oleh TEXT`,
		// Pengaturan per-sekolah: cabut CHECK(id=1) agar id=sekolah.id bisa dipakai
		`ALTER TABLE pengaturan DROP CONSTRAINT IF EXISTS pengaturan_id_check`,
		`ALTER TABLE pengaturan ADD COLUMN IF NOT EXISTS wa_pengasuh TEXT DEFAULT ''`,
		// wali_kelas sempat tidak punya DDL → handleListWali/handleSetWali error 500
		`CREATE TABLE IF NOT EXISTS wali_kelas (
			id SERIAL PRIMARY KEY,
			sekolah_id INT NOT NULL REFERENCES sekolah(id) ON DELETE CASCADE,
			kelas TEXT NOT NULL,
			user_id INT NOT NULL REFERENCES users(id),
			UNIQUE(sekolah_id, kelas)
		)`,
		// wali_token: satu token per siswa — dedupe dulu lalu unique index (sebelumnya menumpuk & bisa multiple rows)
		`DELETE FROM wali_token a USING wali_token b WHERE a.siswa_id=b.siswa_id AND a.sekolah_id=b.sekolah_id AND a.id < b.id`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_wali_token_siswa ON wali_token (siswa_id, sekolah_id)`,
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
		// setelah semua ALTER ADD COLUMN sekolah_id: NIS unik per sekolah
		`ALTER TABLE siswa DROP CONSTRAINT IF EXISTS siswa_nis_key`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_siswa_nis_per_sekolah ON siswa (sekolah_id, nis) WHERE nis IS NOT NULL`,
		// Registrasi instalasi utama sebagai sekolah 1, lalu majukan sequence
		`INSERT INTO sekolah (id, subdomain, nama, status)
		 VALUES (1, 'utama', 'Griya Quran Tunas Ilmu', 'aktif') ON CONFLICT (id) DO NOTHING`,
		// tabel yang punya FK ke sekolah(id) — dibuat setelah sekolah ada (dulu di migrate() sehingga gagal di DB fresh)
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
