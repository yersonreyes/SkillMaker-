# SkillMaker

Plataforma interna de formacion en video — tipo Udemy / LMS corporativo. Cualquier trabajador puede crear cursos compuestos por videos, material de apoyo y evaluaciones, y participar como alumno en los cursos publicados por otros. Los administradores aprueban las publicaciones y supervisan el avance y los puntajes de los participantes.

## Stack

- **Frontend**: Angular 21 (Standalone + Zoneless) + PrimeNG + TailwindCSS
- **Backend**: Go 1.23 + Gin + GORM, monolito modular con 7 modulos de dominio
- **Base de datos**: PostgreSQL 16 + migraciones con golang-migrate
- **Auth**: Google OAuth 2.0 + JWT propio + refresh tokens con rotacion
- **Object Storage**: MinIO (dev) / S3-GCS (prod) para material adjunto
- **API Docs**: OpenAPI/Swagger generado desde anotaciones Go
- **Containers**: Docker + Compose

## Quick start

```bash
make install      # instala deps del backend (Go) y frontend (npm)
make db-up        # levanta PostgreSQL + MinIO
make db-migrate   # aplica migraciones
make db-seed      # crea los 4 roles iniciales
make dev          # arranca backend (Air) + frontend (ng serve) en paralelo
```

Para el setup completo (incluyendo configuracion de Google OAuth), ver [`SETUP.md`](SETUP.md).

## Guias

| Documento | Alcance |
|-----------|---------|
| [`SETUP.md`](SETUP.md) | Pre-requisitos, instalacion, configuracion de Google OAuth, primera ejecucion |
| [`GUIA-MONOREPO.md`](GUIA-MONOREPO.md) | Estructura del monorepo, Makefile, Docker, deployment, comunicacion entre servicios |
| [`frontend/GUIA-FRONT.md`](frontend/GUIA-FRONT.md) | Patrones internos del frontend Angular |
| [`backend/GUIA-BACK.md`](backend/GUIA-BACK.md) | Patrones internos del backend Go + Gin (monolito modular) |
| [`bases/documentacion/`](bases/documentacion/) | Fuente de verdad: requerimientos funcionales, tecnicos y modelo de datos |

## URLs en desarrollo

| Servicio | URL |
|----------|-----|
| Frontend | http://localhost:4200 |
| Backend | http://localhost:3000 |
| Swagger UI | http://localhost:3000/api/docs/index.html |
| MinIO Console | http://localhost:9001 (user: `minioadmin` / pass: `minioadmin`) |

## Roles del dominio

Los 4 roles fijos del sistema (un usuario puede tener varios simultaneamente):

- **alumno**: se inscribe, mira cursos, rinde evaluaciones, obtiene certificados e insignias
- **creador**: crea, edita y envia cursos a aprobacion (cualquier trabajador puede serlo)
- **supervisor**: visualiza avance y puntajes de los empleados a su cargo
- **administrador**: aprueba/rechaza cursos, gestiona usuarios y roles, accede a reportes globales
