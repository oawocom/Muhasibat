# OAWO Mühasibat

**1C-dən daha güclü, tam professional mühasibatlıq sistemi — ayrıca OAWO məhsulu.**

Bu məhsul mövcud OAWO platformasından **tamamilə müstəqildir**. Öz backend-i,
öz verilənlər bazası və öz interfeysi var. Platformanın koduna heç bir asılılığı
yoxdur — tək başına işə düşür və müştərilərə ayrıca lisenziya kimi satıla bilər.

## Nə üçün 1C-dən güclü?

- **Real ikitərəfli (double-entry) mühərrik** — hər əməliyyat balanslı yazılış
  yaradır, debet = kredit qaydası məcburidir. Balanssız yazılış kitablaşdırıla bilmir.
- **Azərbaycan Hesablar Planı** hazır seed olunur (Aktiv / Öhdəlik / Kapital / Gəlir / Xərc).
- **ƏDV (18%) avtomatik** — satış və alış fakturalarında giriş/çıxış ƏDV ayrıca hesablanır.
- **Sənəd → yazılış avtomatizasiyası** — faktura, mədaxil, məxaric təsdiqləndikdə
  mühasibat yazılışı özü yaranır (sistem hesabları vasitəsilə).
- **Anbar uçotu** — mal hərəkəti, qalıqlar və maya dəyəri (COGS) izlənir.
- **Çoxvalyutalılıq** — AZN baza, USD/EUR/RUB/TRY məzənnələri.
- **Peşəkar hesabatlar** — Dövriyyə balansı, Balans, Mənfəət/Zərər, Baş kitab,
  Debitor/Kreditor, Anbar qalıqları.
- **Müasir veb interfeys** — quraşdırma tələb etmir, brauzerdən işləyir.

## Çox şirkətlilik (3 səviyyəli Multi-tenant)

```
Superadmin (OAWO)  →  Tenant (müqavilə / abunə)  →  Şirkət (mühasibat vahidi)
```

- **Superadmin** yalnız yaratdığı **tenantları** (müqavilələri) və onların hesabatlarını görür.
  Hər tenant üçün: bir əsas admin yaradır, abunə olduğu **modulları** və **abunə qiymətini** təyin edir.
  **Ödəniş/abunə məbləği yalnız superadmin-ə görünür.**
- **Tenant admin** ona verilən tək hesabla tenant daxilində **hər şeyi** idarə edir: çox şirkət yaradır,
  istifadəçilər yaradır, rollar təyin edir — amma yalnız abunə olduğu modullar çərçivəsində.
- **Şirkət** başına **ayrı verilənlər bazası** (`oawo_company_<id>`) — tam izolyasiya.
- **Rollar:** Sahib, Admin, Mühasib, Anbardar, Baxış (yalnız oxu). İcazələr backend-də məcburidir;
  başqa tenant/şirkətə giriş bloklanır.
- **Modullar abunəni müəyyən edir** — tenant hansı modulları işlədirsə, qiymət ona görə formalaşır.

## Frontend (React + Tailwind)

İnterfeys **React + Vite + Tailwind CSS** ilə qurulub (`frontend/`). Docker build zamanı statik
fayllar hazırlanıb Go binary-yə hopdurulur (`go:embed`) — yenə **tək konteyner**.

Lokal frontend inkişafı:
```bash
cd frontend && npm install && npm run dev   # http://localhost:5173 (API-ni :8080-ə proxy edir)
```
Docker olmadan Go-nu tək işlətmək üçün əvvəlcə `cd frontend && npm run build` (nəticə `backend/web`-ə düşür).

## Sürətli başlanğıc (Docker)

```bash
cd oawo-muhasibat   # və ya klonladığınız qovluq
docker compose up -d --build
```

Brauzerdə aç: **http://localhost:8090**
Giriş: `admin@oawo.com` / `admin123`

## Lokal işə salmaq (Docker olmadan)

PostgreSQL lazımdır. Sonra:

```bash
cd oawo-muhasibat/backend
export DATABASE_URL="host=localhost port=5432 user=oawo password=oawo dbname=oawo_muhasibat sslmode=disable"
go run .
```

Backend `:8080` portunda həm API-ni, həm də embedded interfeysi verir.

## Arxitektura

```
oawo-muhasibat/
├── docker-compose.yml        # PostgreSQL + tətbiq
├── frontend/                # React + Tailwind (Vite) — interfeys
└── backend/
    ├── main.go               # server + embedded frontend
    ├── web/                  # React build çıxışı (Docker-də yaranır, go:embed)
    └── internal/
        ├── models/           # verilənlər bazası modelləri
        ├── engine/           # ikitərəfli mühərrik + hesabatlar
        ├── seed/             # AZ hesablar planı, valyuta, ƏDV
        ├── store/            # DB bağlantısı + miqrasiya
        └── api/              # REST API + autentifikasiya
```

- **Backend:** Go 1.24 (Gin, GORM) · **Frontend:** React + Vite + Tailwind CSS
- **Baza:** PostgreSQL 15
- **Tək binary:** frontend Go binary-ə hopdurulur (`go:embed`), ayrıca build lazım deyil.

## API (nümunə)

| Metod | Endpoint | Təsvir |
|-------|----------|--------|
| POST | `/api/auth/login` | Giriş |
| GET | `/api/accounts` | Hesablar planı |
| POST | `/api/documents?post=1` | Faktura yarat və təsdiqlə |
| POST | `/api/journal/:id/post` | Yazılışı kitablaşdır |
| GET | `/api/reports/trial-balance` | Dövriyyə balansı |
| GET | `/api/reports/balance-sheet` | Balans |
| GET | `/api/reports/profit-loss` | Mənfəət / Zərər |
| GET | `/api/dashboard` | İdarə paneli göstəriciləri |

## Konfiqurasiya (ENV)

| Dəyişən | Default | Təsvir |
|---------|---------|--------|
| `DATABASE_URL` | — | Tam DSN (varsa DB_* əvəz olunur) |
| `DB_HOST/PORT/USER/PASSWORD/NAME` | localhost/5432/oawo/oawo/oawo_muhasibat | DB parametrləri |
| `PORT` | 8080 | Server portu |
| `JWT_SECRET` | dev-secret | Token imzası (produksiyada dəyişin) |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | admin@oawo.com / admin123 | İlk admin |
| `COMPANY_NAME` | OAWO MMC | Şirkət adı |

> **Qeyd:** Produksiyada mütləq `JWT_SECRET` və admin şifrəsini dəyişin.
