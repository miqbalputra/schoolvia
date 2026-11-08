<div align="center">

# 🎓 SchoolVia

**SchoolVia — Sistem Admin Sekolah All-in-One (SD, SMP, SMA/sederajat & lembaga Islam) · Multi-Sekolah SaaS**

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Vue 3](https://img.shields.io/badge/Vue-3-4FC08D?style=flat-square&logo=vue.js&logoColor=white)](https://vuejs.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql&logoColor=white)](https://postgresql.org)
[![License](https://img.shields.io/badge/License-PolyForm%20Shield%201.0.0-blue?style=flat-square)](LICENSE)
[![Release](https://img.shields.io/github/v/release/miqbalputra/free-edu?style=flat-square)](https://github.com/miqbalputra/free-edu/releases)
[![PWA](https://img.shields.io/badge/PWA-Installable-5A0FC8?style=flat-square&logo=pwa&logoColor=white)](#-pwa)

*Absensi · Hafalan · Nilai & Rapor · SPP & Keuangan · Laporan Orang Tua — semuanya dari HP, tanpa install.*

[🚀 Coba Demo](https://sistemflow.com/app?demo=1) · [📖 Dokumentasi](#-memulai) · [💬 WhatsApp](https://wa.me/6281234567890) · [📝 Pitch Deck](frontend/pitch.html)

</div>

---

## 📌 Tentang SchoolVia

SchoolVia adalah aplikasi admin untuk **lembaga tahfidz & TPQ di Indonesia** — dibangun dari keluhan nyata: hafalan santri tercatat di buku tulis yang lama-lama hilang, tagihan SPP bocor karena diperiksa manual, dan orang tua harus bertanya satu-satu soal perkembangan anaknya.

> **Untuk siapa**: lembaga 20–300 santri · Kepala Sekolah, Guru, Tata Usaha
> **Filosofi**: menu sederhana untuk pengguna non-teknis, fitur lengkap di baliknya — semuanya berbahasa Indonesia.

### Mengapa SchoolVia berbeda

| | SchoolVia | Aplikasi sekolah umum | Excel + WhatsApp |
|---|---|---|---|
| Pencatatan hafalan (surah, ayat, murojaah) | ✅ | ❌ | Manual |
| Laporan ke orang tua via WA — **tanpa install app** | ✅ Link aman per-wali | ❌ | Rapi susah |
| SPP sekali klik + kwitansi berkop sekolah | ✅ | Sebagian | Manual |
| Data milik Anda — unduh ZIP kapan saja, apa pun paket | ✅ Termasuk gratis | Sering jadi sandera | ✅ |
| Bahasa & alur khas tahfidz (juz, murojaah, setoran) | ✅ | ❌ | ✅ |

---

## ✨ Fitur

### 🎓 Operasional Harian
- **Absensi santri** — satu layar per kelas: Hadir, Sakit, Izin, Alpa (2 klik dari HP); absensi guru masuk/pulang dengan jam otomatis
- **Pencatatan hafalan** — setoran surah & ayat, status lancar/murojaah/baru, riwayat lengkap per santri, tombol ⚡ Hafal cepat per siswa
- **Nilai & rapor** — input nilai massal satu kelas, rapor resmi berkop sekolah siap cetak/PDF, catatan wali kelas
- **Jadwal, agenda & pengumuman** — dengan wali kelas per kelas

### 💰 Keuangan
- **SPP sekali klik** — generate tagihan bulanan sekaligus untuk semua santri
- **Kwitansi digital** — berkop sekolah, nomor kwitansi, dua kolom tanda tangan, siap cetak
- **Buku kas** — pemasukan/pengeluaran, saldo otomatis, ekspor Excel
- **Pengingat & kwitansi via WhatsApp** — pesan siap kirim ke orang tua (nama, bulan, nominal sudah tertulis)

### 📊 Laporan & Orang Tua
- **Laporan rekap bulanan** — SPP, kas, absensi, nilai, hafalan dalam satu halaman siap cetak
- **Laporan wali murid** — link read-only khusus per orang tua (kadaluarsa 90 hari, bisa di-reset): hafalan, kehadiran, tagihan — dalam bahasa awam, tanpa install apa pun
- **Laporan mingguan otomatis** — pengasuh menerima ringkasan di WhatsApp tiap malam tanpa perlu buka aplikasi

### 🔐 Milik Anda Selamanya (Anti-Sandera)
- **Unduh Semua Data** — satu klik → ZIP berisi seluruh data sekolah dalam format Excel-friendly
- Tersedia di **semua paket termasuk gratis** — berhenti langganan pun data tetap bisa diunduh (mode baca, bukan dikunci)
- Ekspor per-modul: siswa, nilai, hafalan, absensi, SPP, kas, rekap

### 🏗️ Platform SaaS
- **Multi-tenant** — sekolah mendaftar mandiri, data terisolasi penuh antar sekolah (terverifikasi uji lintas-tenant)
- **Paket berjenjang** — Gratis (≤30 santri) / Sekolah Rp45rb-bln (≤150) / Yayasan (multi-cabang, segera)
- **Program rujukan** — kode unik per sekolah; perujuk dapat 1 bulan gratis
- **Panel admin platform** — statistik MRR, kelola sekolah & langganan
- **PWA** — dipasang dari browser tanpa Play Store/App Store, ada dark mode
- **Import massal** — daftar siswa dari Excel/CSV: preview, pemetaan kolom otomatis, perbaikan kelas otomatis, anti-duplikat

---

## 👥 Fitur per Peran

| Modul | Kepala Sekolah | Guru | Tata Usaha |
|---|:---:|:---:|:---:|
| Dashboard statistik | ✅ | ✅ | ✅ |
| Data siswa (CRUD + import) | ✅ | lihat | ✅ |
| Absensi santri | ✅ | ✅ | ✅ |
| Nilai & hafalan | ✅ | ✅ input | lihat |
| SPP & keuangan | ✅ | lihat | ✅ |
| Laporan rekap & wali murid | ✅ | — | ✅ |
| Manajemen pengguna & paket | ✅ | — | — |
| Profil sekolah & unduh data | ✅ | — | ✅ |

Tema warna berbeda per peran — setiap orang langsung tahu dia sedang "di mana".

---

## 🚀 Memulai

### Prasyarat
- Go 1.22+
- PostgreSQL 14+
- (opsional) Node tidak diperlukan — frontend disajikan langsung oleh Go

### 1️⃣ Siapkan database

```bash
sudo -u postgres psql -c "CREATE USER simaka WITH PASSWORD 'simaka123';"
sudo -u postgres psql -c "CREATE DATABASE simaka OWNER simaka;"
```

### 2️⃣ Jalankan

```bash
git clone https://github.com/miqbalputra/free-edu.git
cd free-edu/backend
go build -o simaka .
export DATABASE_URL='postgres://simaka:simaka123@localhost:5432/simaka?sslmode=disable'  # dev lokal saja; produksi pakai sslmode=require
# dev lokal tanpa DATABASE_URL: set ALLOW_INSECURE_DEFAULT_DSN=1
./simaka
```

✅ Migrasi skema & data demo otomatis saat pertama jalan.
🌐 Aplikasi: `http://localhost:8080` · Landing: `http://localhost:8080/`

**Konfigurasi** (opsional, environment variable):

| Variabel | Default | Keterangan |
|---|---|---|
| `DATABASE_URL` | **wajib di-set** | Koneksi PostgreSQL, mis. `postgres://user:pass@host:5432/namadb?sslmode=require`. Dev lokal tanpa env ini: set `ALLOW_INSECURE_DEFAULT_DSN=1` |
| `PORT` | `8080` | Port HTTP |
| `SchoolVia_PUBLIC_URL` | `https://sistemflow.com` | URL publik untuk link WA |
| `SEED_DEMO` | `0` | Bila `1`: seed akun demo (`kepala@simaka.id`/`kepala123`) & aktifkan `?demo=1` |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | — | Wajib bila `SEED_DEMO` tidak di-set: membuat akun kepala sekolah pertama pada instalasi baru |
| `TRUSTED_PROXY` | `0` | Bila `1` (di belakang reverse proxy), rate-limit memakai IP terakhir dari `X-Forwarded-For` |
| `COOKIE_SECURE` | `0` | Bila `1` (wajib di produksi HTTPS), cookie sesi `sv_session` diberi flag `Secure` |
| `FRONTEND_DIR` | otomatis (relatif ke executable, lalu `../frontend`) | Lokasi direktori frontend yang disajikan server |
| `ALLOW_INSECURE_DEFAULT_DSN` | `0` | Bila `1`, izinkan DSN default lokal tanpa password aman — **jangan dipakai di produksi** |

| `TEST_DATABASE_URL` | — | Hanya untuk test: bila di-set, `go test ./...` di `backend/` menjalankan test integrasi lintas-tenant dengan PostgreSQL asli (CI memakainya via service container). Tanpa env ini, test integrasi di-skip otomatis. |

> ⚠️ Sebelum go-live: ganti nomor WhatsApp placeholder `6281234567890` (README & `WA_SUPPORT_PLACEHOLDER` di `frontend/index.html`) dengan nomor resmi Anda.

### 3️⃣ Masuk

| Peran | Email | Password |
|---|---|---|
| Kepala Sekolah | `kepala@simaka.id` | (lihat data demo) |
| Guru | `guru@simaka.id` | (lihat data demo) |
| Tata Usaha | `tu@simaka.id` | (lihat data demo) |

> Data demo berisi sekolah contoh dengan santri, nilai, hafalan, dan tagihan — pas untuk mencoba semua alur sebelum data asli dimasukkan. **Akun demo hanya ada bila server dijalankan dengan `SEED_DEMO=1`.** Pada instalasi baru tanpa `SEED_DEMO`, buat akun kepala sekolah pertama via `ADMIN_EMAIL` + `ADMIN_PASSWORD`.

---

## 🏛️ Arsitektur

```
┌─────────────────────────────────────────────────┐
│                   Browser / PWA                  │
│      Vue 3 (single-page, dark mode, i18n ID)     │
└──────────────────────┬──────────────────────────┘
                       │ HTTPS (JWT Bearer)
┌──────────────────────▼──────────────────────────┐
│                Go Backend (single binary)        │
│  ┌───────────┐ ┌───────────┐ ┌───────────────┐  │
│  │ Auth      │ │ Tenant    │ │ Plan/Billing  │  │
│  │ JWT+bcrypt│ │ middleware│ │ kuota+gating  │  │
│  └───────────┘ └───────────┘ └───────────────┘  │
│  ┌───────────┐ ┌───────────┐ ┌───────────────┐  │
│  │ Modules   │ │ Reports   │ │ WA Outbound   │  │
│  │ 15+ CRUD  │ │ rekap/wali│ │ wa.me links   │  │
│  └───────────┘ └───────────┘ └───────────────┘  │
│  Rate-limit (DB) · Security headers · Audit log  │
└──────────────────────┬──────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────┐
│              PostgreSQL (28 index, 18 FK)         │
│   Multi-tenant via sekolah_id di semua tabel     │
└─────────────────────────────────────────────────┘
```

**Prinsip keamanan** (hasil audit penuh, 21 temuan → diperbaiki semua):
- Isolasi tenant di **setiap** endpoint — termasuk ekspor, rapor cetak, kwitansi (terverifikasi uji lintas-sekolah)
- bcrypt + JWT auto-generated secret · rate-limit login persisten (5 gagal → lock 15 menit)
- Security headers: CSP, X-Frame-Options, nosniff, Referrer-Policy
- Hapus siswa = **arsip** (status `keluar`) — nilai, tagihan, hafalan tetap tersimpan
- Transaksi DB untuk operasi multi-tabel (hapus siswa, generate SPP)

Struktur proyek:

```
free-edu/
├── backend/           # Go — semua handler & migrasi
│   ├── routes.go      #   routing & middleware
│   ├── exports.go     #   ekspor CSV per modul
│   ├── rekap.go       #   laporan rekap & kwitansi
│   ├── wali_token.go  #   token laporan orang tua
│   └── ...
├── frontend/          # Vue 3 (index.html), landing, pitch, ikon PWA
└── LICENSE            # PolyForm Strict 1.0.0
```

---

## 🗺️ Roadmap

- [x] Multi-tenant SaaS + trial mandiri
- [x] Keamanan penuh (audit 21 temuan)
- [x] Laporan wali murid via WA (token per-wali, kadaluarsa, reset)
- [x] Unduh Semua Data (anti-sandera) di semua paket
- [x] Import massal Excel/CSV
- [x] Laporan mingguan otomatis ke WhatsApp pengasuh
- [ ] **Multi-cabang** (paket Yayasan) — *sedang dibangun, waitlist terbuka*
- [ ] Pembayaran otomatis (QRIS/Midtrans)
- [ ] Notifikasi tagihan terjadwal

---

## 🤝 Kontribusi

Kontribusi dipersilakan! Karena lisensi PolyForm Strict, kontribusi berupa **pull request ke repo ini** (bukan fork modifikasi). Baca dulu [Lisensi](#-lisensi) di bawah.

Ide kontribusi yang sedang dicari: terjemahan bahasa daerah, laporan baru, perbaikan UX untuk pengguna gaptek, dokumentasi.

---

> ℹ️ Sebelumnya dikenal sebagai **SIMAKA** — berganti nama menjadi SchoolVia (Okt 2026). Kode & data tidak berubah.

## 📄 Lisensi

Proyek ini dilisensikan under [**PolyForm Shield License 1.0.0**](LICENSE).

**Singkatnya — untuk lembaga sekolah:** pakai, sesuaikan, dan jalankan untuk lembaga Anda sepuasnya, termasuk untuk kegiatan berbayar seperti SPP. Tidak ada batasan untuk Anda.

| ✅ Boleh untuk semua orang | ❌ Tidak boleh |
|---|---|
| Memakai SchoolVia untuk lembaga sekolah/tahfidz/TPQ apa pun | Menjadikan SchoolVia (atau turunannya) sebagai **produk yang bersaing** dengan SchoolVia |
| Memodifikasi kode untuk kebutuhan lembaga sendiri (logo, istilah, alur) | Menjual ulang atau menawarkan versi hosted yang bersaing |
| Membagikan salinan asli atau yang dimodifikasi | Menghapus notice lisensi |

> **Kenapa lisensi ini?** Kami ingin lembaga pendidikan memakai SchoolVia tanpa was-was — termasuk memodifikasi sesuai kebutuhan. Tapi kami juga membangun SchoolVia sebagai pekerjaan yang berkelanjutan, jadi satu-satunya larangan adalah menjadikannya produk pesaing. Teks lengkap: [LICENSE](LICENSE).

**Buat kontributor:** PR ke repo ini sangat dipersilakan. Dengan mengirim PR, Anda menyetujui kode Anda dilisensikan di bawah lisensi yang sama (dan haknya diberikan ke pemilik proyek).

---

<div align="center">

**Dibuat oleh [Iqbal Putra](https://github.com/miqbalputra) · [SistemFlow.com](https://sistemflow.com)**

[🚀 Coba Demo](https://sistemflow.com/app?demo=1) · [💬 Chat WhatsApp](https://wa.me/6281234567890) · [⭐ Beri bintang jika bermanfaat](https://github.com/miqbalputra/free-edu/stargazers)

</div>
