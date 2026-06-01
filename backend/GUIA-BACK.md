# Guia de Patrones — Backend Go + Gin (SkillMaker)

> **Proposito**: Documenta los patrones, librerias, configuraciones y convenciones del backend de **SkillMaker** — plataforma interna de formacion en video (tipo Udemy / LMS corporativo). Tambien sirve como plantilla reutilizable para construir un monolito modular en Go + Gin + GORM con auth Google OAuth + JWT y object storage S3/MinIO.

> **Para entender el contexto del proyecto:**
> - Estructura del monorepo, Docker, deployment: [`../GUIA-MONOREPO.md`](../GUIA-MONOREPO.md)
> - Frontend Angular: [`../frontend/GUIA-FRONT.md`](../frontend/GUIA-FRONT.md)
> - Requerimientos (fuente de verdad): [`../bases/documentacion/`](../bases/documentacion/)
>   - `Requerimientos_Plataforma_Formacion.docx` — RFs y RNFs
>   - `Requerimientos_Tecnicos_Plataforma_Formacion.docx` — RTs y arquitectura C4
>   - `Modelo_Datos_Plataforma_Formacion.docx` — diccionario de datos y DDL

---

## Tabla de Contenidos

1. [Stack Tecnologico y Librerias](#1-stack-tecnologico-y-librerias)
2. [Bootstrap y Composition Root (cmd/api/main.go)](#2-bootstrap-y-composition-root-cmdapimain-go)
3. [Configuracion de Swagger / OpenAPI (swaggo/swag)](#3-configuracion-de-swagger--openapi-swaggoswag)
4. [Estructura del Monolito Modular](#4-estructura-del-monolito-modular)
5. [Patron de DTOs (Request / Response)](#5-patron-de-dtos-request--response)
6. [Patron de Handlers (Gin)](#6-patron-de-handlers-gin)
7. [Patron de Services (Logica de Dominio)](#7-patron-de-services-logica-de-dominio)
8. [Patron de Repositories (GORM)](#8-patron-de-repositories-gorm)
9. [Sistema de Autenticacion (Google OAuth + JWT)](#9-sistema-de-autenticacion-google-oauth--jwt)
10. [Sistema de Roles y Autorizacion (RBAC)](#10-sistema-de-roles-y-autorizacion-rbac)
11. [GORM: Modelos y Convenciones](#11-gorm-modelos-y-convenciones)
12. [Migraciones con golang-migrate](#12-migraciones-con-golang-migrate)
13. [Object Storage (S3 / MinIO) y URLs Prefirmadas](#13-object-storage-s3--minio-y-urls-prefirmadas)
14. [Comunicacion entre Modulos (Interfaces)](#14-comunicacion-entre-modulos-interfaces)
15. [Manejo de Errores Consistente](#15-manejo-de-errores-consistente)
16. [Testing (go test, testify, testcontainers)](#16-testing-go-test-testify-testcontainers)
17. [Logging Estructurado y Observabilidad](#17-logging-estructurado-y-observabilidad)
18. [DevOps y Calidad de Codigo](#18-devops-y-calidad-de-codigo)
19. [Convenciones de Nomenclatura](#19-convenciones-de-nomenclatura)
20. [Checklist para Crear un Nuevo Modulo](#20-checklist-para-crear-un-nuevo-modulo)

---

## 1. Stack Tecnologico y Librerias

### Dependencias de Produccion

| Libreria | Version recomendada | Proposito |
|----------|---------------------|-----------|
| `github.com/gin-gonic/gin` | v1.10+ | Framework HTTP (RT-06) |
| `gorm.io/gorm` | v1.25+ | ORM para PostgreSQL (RT-17) |
| `gorm.io/driver/postgres` | v1.5+ | Driver Postgres para GORM |
| `github.com/golang-migrate/migrate/v4` | v4.18+ | Migraciones reproducibles (RT-18) |
| `github.com/golang-jwt/jwt/v5` | v5.2+ | Emision/verificacion de JWT (RT-14, RT-15) |
| `google.golang.org/api/idtoken` | latest | Validacion de ID tokens de Google OAuth (RT-12, RT-13) |
| `github.com/minio/minio-go/v7` | v7.0+ | Cliente S3-compatible (RT-21) |
| `github.com/google/uuid` | v1.6+ | Generacion de UUIDs (modelo de datos seccion 1) |
| `github.com/caarlos0/env/v11` | v11+ | Carga tipada de variables de entorno (RT-11) |
| `github.com/swaggo/swag` + `swaggo/gin-swagger` | v1.16+ | OpenAPI desde anotaciones (RT-09) |
| `github.com/gin-contrib/cors` | v1.7+ | Middleware CORS |
| `github.com/gin-contrib/requestid` | v1+ | Request-id en cada peticion (para trazabilidad) |
| `go.uber.org/zap` o `log/slog` (estandar) | latest | Logging estructurado (RT-28) |
| `github.com/go-playground/validator/v10` | v10+ | Validacion de DTOs |

### Dependencias de Desarrollo

| Libreria | Version | Proposito |
|----------|---------|-----------|
| `github.com/air-verse/air` | v1.52+ | Hot reload local |
| `github.com/swaggo/swag/cmd/swag` | v1.16+ | CLI para regenerar `docs/` |
| `github.com/golang-migrate/migrate/v4/cmd/migrate` | v4.18+ | CLI de migraciones |
| `github.com/golangci/golangci-lint` | v1.62+ | Linter unificado |
| `github.com/stretchr/testify` | v1.10+ | Aserciones y mocks en tests |
| `github.com/testcontainers/testcontainers-go` | v0.34+ | Postgres efimero en tests de integracion |
| `go.uber.org/mock` o `github.com/stretchr/testify/mock` | latest | Generacion / declaracion de mocks |

### Version de Go

`go 1.23` (segun la directiva `toolchain` del `go.mod`). Esta version es compatible con `slog`, `range over int`, y mejoras de performance del runtime.

---

## 2. Bootstrap y Composition Root (cmd/api/main.go)

`cmd/api/main.go` es el **composition root**: el unico lugar donde se construyen implementaciones concretas y se inyectan en otros modulos via interfaces. **Toda la logica vive en `internal/`**, este archivo solo cabletea.

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"

    "github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/users"
    "github.com/yersonreyes/SkillMaker-/backend/internal/platform/config"
    "github.com/yersonreyes/SkillMaker-/backend/internal/platform/database"
    "github.com/yersonreyes/SkillMaker-/backend/internal/platform/httpserver"
    "github.com/yersonreyes/SkillMaker-/backend/internal/platform/logger"
    "github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"

    _ "github.com/yersonreyes/SkillMaker-/backend/docs" // generado por swag
)

// @title           SkillMaker API
// @version         1.0
// @description     Plataforma interna de formacion en video.
// @host            localhost:3000
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
    // 1. Configuracion + logger
    cfg := config.MustLoad()
    log := logger.New(cfg.LogLevel, cfg.AppEnv)
    slog.SetDefault(log)

    // 2. Infraestructura (platform)
    db, err := database.Open(cfg.DatabaseURL, cfg.DBMaxOpenConns, cfg.DBMaxIdleConns)
    if err != nil {
        log.Error("no se pudo abrir la base de datos", "err", err)
        os.Exit(1)
    }
    defer database.Close(db)

    storageClient, err := storage.New(cfg.Storage)
    if err != nil {
        log.Error("no se pudo inicializar storage", "err", err)
        os.Exit(1)
    }

    // 3. Modulos de dominio — construir de "menos dependientes" a "mas dependientes"
    usersRepo := users.NewRepository(db)
    usersSvc := users.NewService(usersRepo)

    authRepo := auth.NewRepository(db)                        // store de refresh tokens
    authSvc := auth.NewService(cfg.Auth, usersSvc, authRepo)  // depende de users.Service

    certsRepo := certificates.NewRepository(db)
    certsSvc := certificates.NewService(certsRepo, storageClient)

    evalsRepo := evaluations.NewRepository(db)
    evalsSvc := evaluations.NewService(evalsRepo, certsSvc) // emite certificado al aprobar

    coursesRepo := courses.NewRepository(db)
    coursesSvc := courses.NewService(coursesRepo, storageClient, evalsSvc)

    approvalsRepo := approvals.NewRepository(db)
    approvalsSvc := approvals.NewService(approvalsRepo, coursesSvc)

    reportingSvc := reporting.NewService(db) // lectura agregada (vista read-only)

    // 4. HTTP server: registrar rutas
    if cfg.AppEnv == "production" {
        gin.SetMode(gin.ReleaseMode)
    }
    router := httpserver.NewRouter(cfg)
    api := router.Group("/api")

    auth.RegisterRoutes(api, authSvc)
    apiAuth := api.Group("", middleware.JWT(cfg.Auth.JWTSecret))
    {
        users.RegisterRoutes(apiAuth, usersSvc)
        courses.RegisterRoutes(apiAuth, coursesSvc)
        evaluations.RegisterRoutes(apiAuth, evalsSvc)
        approvals.RegisterRoutes(apiAuth, approvalsSvc)
        certificates.RegisterRoutes(apiAuth, certsSvc)
        reporting.RegisterRoutes(apiAuth, reportingSvc)
    }

    srv := httpserver.NewServer(cfg.Port, router)

    // 5. Arranque con graceful shutdown
    go func() {
        log.Info("servidor escuchando", "port", cfg.Port)
        if err := srv.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
            log.Error("server error", "err", err)
            os.Exit(1)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Info("apagando servidor...")

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    _ = srv.Shutdown(ctx)
    log.Info("servidor apagado")
}
```

**Principios de este archivo:**

- **Cero logica de dominio**: solo wiring. Si quieres agregar una regla de negocio aqui, **detente y movela al service del modulo correspondiente**.
- **Construccion explicita**: ver el grafo de dependencias en codigo (no hay magia DI). Esto hace trivial reemplazar implementaciones (fakes en tests, clientes HTTP para microservicios futuros).
- **Orden de construccion**: primero infra (`config`, `logger`, `db`, `storage`), luego modulos de dominio en orden de dependencia, finalmente el HTTP server.
- **Graceful shutdown**: el server se cierra limpiamente con timeout de 15s ante `SIGINT`/`SIGTERM` (RT-25 → buen comportamiento en Docker).
- **Trust proxy**: en `httpserver.NewServer` se configura el `TrustedProxies` de Gin para que `c.ClientIP()` devuelva la IP real detras de nginx.

### Carga de Configuracion (internal/platform/config)

Patron tipado con `caarlos0/env`:

```go
package config

import (
    "log"
    "time"

    "github.com/caarlos0/env/v11"
)

type Config struct {
    AppEnv          string        `env:"APP_ENV" envDefault:"development"`
    Port            int           `env:"PORT" envDefault:"3000"`
    LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
    AllowedOrigins  []string      `env:"ALLOWED_ORIGINS" envSeparator:"," envDefault:""`
    DatabaseURL     string        `env:"DATABASE_URL,required"`
    DBMaxOpenConns  int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
    DBMaxIdleConns  int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
    Auth            AuthConfig    `envPrefix:""`
    Storage         StorageConfig `envPrefix:""`
}

type AuthConfig struct {
    JWTSecret             string        `env:"JWT_SECRET,required"`
    JWTExpiresIn          time.Duration `env:"JWT_EXPIRES_IN" envDefault:"1h"`
    RefreshTokenExpiresIn time.Duration `env:"REFRESH_TOKEN_EXPIRES_IN" envDefault:"168h"` // 7 dias
    GoogleClientID        string        `env:"GOOGLE_CLIENT_ID,required"`
    GoogleClientSecret    string        `env:"GOOGLE_CLIENT_SECRET"`
    GoogleHostedDomain    string        `env:"GOOGLE_HOSTED_DOMAIN,required"`
    GoogleRedirectURI     string        `env:"GOOGLE_REDIRECT_URI"`
}

type StorageConfig struct {
    Endpoint        string        `env:"STORAGE_ENDPOINT,required"`
    Region          string        `env:"STORAGE_REGION" envDefault:"us-east-1"`
    Bucket          string        `env:"STORAGE_BUCKET,required"`
    AccessKey       string        `env:"STORAGE_ACCESS_KEY,required"`
    SecretKey       string        `env:"STORAGE_SECRET_KEY,required"`
    UseSSL          bool          `env:"STORAGE_USE_SSL" envDefault:"true"`
    PresignTTL      time.Duration `env:"STORAGE_PRESIGN_TTL" envDefault:"15m"`
    MaxUploadBytes  int64         `env:"MAX_UPLOAD_BYTES" envDefault:"52428800"` // 50MB (RT-24b)
}

func MustLoad() Config {
    var cfg Config
    if err := env.Parse(&cfg); err != nil {
        log.Fatalf("config: %v", err)
    }
    return cfg
}
```

**Reglas:**
- `required` se valida en arranque. **El proceso no levanta sin sus secrets**. Esto previene errores en runtime por configuracion incompleta (RNFT-03).
- **Nunca leer `os.Getenv` directamente** desde los modulos de dominio. Si un modulo necesita una variable, agrega un campo a `Config` y pasala como parametro.

---

## 3. Configuracion de Swagger / OpenAPI (swaggo/swag)

El contrato de la API se genera desde anotaciones en los handlers Go usando `swaggo/swag` (RT-09).

### Setup en main.go

Las anotaciones del header del paquete `main` (ya vistas en seccion 2) definen el spec global. Tras agregar/cambiar endpoints:

```bash
make swagger        # equivale a: swag init -g cmd/api/main.go -o docs
```

Esto regenera `backend/docs/swagger.json`, `swagger.yaml` y `docs.go`. El servidor expone la UI en `/api/docs/index.html`:

```go
// internal/platform/httpserver/router.go
import (
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(cfg config.Config) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery(), requestid.New(), middleware.Logger(), middleware.CORS(cfg.AllowedOrigins))
    r.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
    r.GET("/api/health", healthHandler)
    return r
}
```

### Decoradores Swagger en Handlers

Cada endpoint debe anotarse con un bloque `// godoc`. Ejemplo de un handler de cursos:

```go
// Catalog godoc
// @Summary      Catalogo de cursos aprobados
// @Description  Lista paginada con busqueda por titulo o creador (RF-18, RF-18b)
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        page  query     int     false  "Pagina (default 1)"
// @Param        size  query     int     false  "Tamano de pagina (default 12, max 100)"
// @Param        q     query     string  false  "Texto de busqueda"
// @Success      200   {object}  dto.PaginatedResponse[dto.CourseResponse]
// @Failure      401   {object}  dto.ErrorResponse
// @Router       /courses [get]
// @Security     BearerAuth
func (h *Handler) Catalog(c *gin.Context) { ... }
```

**Reglas:**
- **Todo endpoint protegido lleva `@Security BearerAuth`**. Si lo omitis, el UI no envia el header `Authorization`.
- **Documentar todos los responses no-200 relevantes** (`400`, `401`, `403`, `404`, `409`, `500`). El equipo de frontend confia en esto.
- **Tags**: uno por modulo (`auth`, `users`, `courses`, `evaluations`, `approvals`, `certificates`, `reporting`).

---

## 4. Estructura del Monolito Modular

El backend es un **monolito modular** (RT-07): un solo binario dividido en modulos de dominio con frontera estricta. Cada modulo puede extraerse a microservicio sin reescribir su logica (RNFT-02).

### Anatomia de un Modulo

```
backend/internal/modules/<modulo>/
├── <modulo>.go             # API publica: interface Service + factories (NewService, NewRepository, NewHandler, RegisterRoutes)
├── handler/
│   └── handler.go          # Capa HTTP (Gin)
├── service/
│   └── service.go          # Logica de dominio
├── repository/
│   └── repository.go       # Acceso a datos (GORM)
├── domain/
│   └── models.go           # Entidades GORM + value objects
├── dto/
│   ├── request.go          # DTOs de entrada (con tags `binding`)
│   └── response.go         # DTOs de salida
└── service_test.go         # Tests unitarios al lado de la logica
```

### Los 7 Modulos de Dominio

Definidos en la documentacion tecnica seccion 4.3 (Componentes del backend, Nivel 3 C4):

| Modulo | Responsabilidad | Tablas (modelo de datos) |
|--------|------------------|---------------------------|
| `auth` | Validar ID token de Google, emitir JWT propio | — (no tiene tablas propias; usa `users`) |
| `users` | Usuarios, roles, supervision | `user`, `role`, `user_role`, `supervision` |
| `courses` | Cursos, secciones, videos, material, inscripcion | `course`, `section`, `video`, `material`, `enrollment` |
| `evaluations` | Evaluaciones, preguntas, intentos, respuestas | `evaluation`, `question`, `option`, `attempt`, `answer` |
| `approvals` | Revision y aprobacion de cursos | `approval` |
| `certificates` | Certificados, badges, ranking | `certificate`, `badge`, `user_badge` |
| `reporting` | Reportes agregados (admin + supervisor) | — (lectura cruzada read-only) |

**Frontera de propiedad (RT-19):**
- Cada modulo es **dueno** de sus tablas. Solo su `repository` las consulta.
- Si `evaluations` necesita un usuario, **llama** `users.GetByID(ctx, id)`, **no** hace `JOIN` contra `user`.
- `reporting` es la unica excepcion: lee vistas/queries agregadas read-only. No escribe nada.

### API Publica del Modulo (modulo.go)

```go
// internal/modules/courses/courses.go
package courses

import (
    "context"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/dto"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/handler"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations"
    "github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
)

// Service es la API publica del modulo courses.
// Otros modulos dependen de esta interface, nunca de la struct concreta.
type Service interface {
    Catalog(ctx context.Context, q dto.CatalogQuery) (dto.PaginatedResponse[dto.CourseResponse], error)
    GetByID(ctx context.Context, id string) (dto.CourseDetailResponse, error)
    Create(ctx context.Context, creatorID string, req dto.CreateCourseRequest) (dto.CourseResponse, error)
    SubmitForReview(ctx context.Context, creatorID, courseID string) error
    Publish(ctx context.Context, courseID string) error // llamado por approvals al aprobar
    // ...
}

// Repository, NewRepository, NewService, RegisterRoutes son re-exportados
// para uso desde cmd/api/main.go. El consumidor externo NUNCA importa los sub-paquetes.

func NewRepository(db *gorm.DB) repository.Repository              { return repository.New(db) }
func NewService(r repository.Repository, s storage.Client, e evaluations.Service) Service {
    return service.New(r, s, e)
}
func RegisterRoutes(rg *gin.RouterGroup, s Service)                { handler.Register(rg, s) }
```

**Por que este patron:**
- El consumidor (`cmd/api/main.go`) ve **un solo paquete** por modulo (`courses`), no sus internals.
- Los sub-paquetes (`handler`, `service`, `repository`) son detalles de implementacion.
- Si manana `courses` se vuelve microservicio, el sub-paquete `handler` puede reemplazarse por un cliente HTTP sin que el resto del sistema se entere.

---

## 5. Patron de DTOs (Request / Response)

Los DTOs son **structs planos** que representan el contrato HTTP. **Nunca usar los modelos GORM como DTOs** — separa la representacion de almacenamiento del contrato API.

### Request DTOs

```go
// internal/modules/courses/dto/request.go
package dto

type CreateCourseRequest struct {
    Titulo      string  `json:"titulo" binding:"required,min=3,max=200"`
    Descripcion *string `json:"descripcion" binding:"omitempty,max=2000"`
}

type UpdateCourseRequest struct {
    Titulo      *string `json:"titulo" binding:"omitempty,min=3,max=200"`
    Descripcion *string `json:"descripcion" binding:"omitempty,max=2000"`
}

type AddVideoRequest struct {
    SectionID    string `json:"sectionId" binding:"required,uuid"`
    Titulo       string `json:"titulo" binding:"required"`
    URL          string `json:"url" binding:"required,url"`
    Proveedor    string `json:"proveedor" binding:"required,oneof=youtube vimeo"` // RT-24
    DuracionSeg  *int   `json:"duracionSeg" binding:"omitempty,min=0"`
    Orden        int    `json:"orden" binding:"required,min=1"`
}

type CatalogQuery struct {
    Page int    `form:"page,default=1" binding:"min=1"`
    Size int    `form:"size,default=12" binding:"min=1,max=100"`
    Q    string `form:"q" binding:"max=100"`
}
```

**Reglas:**
- **Usar punteros para campos opcionales** (`*string`, `*int`). Asi se distingue "no enviado" de "string vacio". Critico en `UpdateXxxRequest` para no sobreescribir con zero values.
- **Validaciones declarativas** con tags `binding` (de go-playground/validator integrado en Gin).
- **`oneof`** para enums (estados, proveedores, tipos de pregunta). Refleja los `CHECK` del modelo de datos.
- **`uuid`** para validar IDs (las entidades de dominio usan UUID v4, modelo de datos seccion 1).

### Response DTOs

```go
// internal/modules/courses/dto/response.go
package dto

import "time"

type CourseResponse struct {
    ID           string    `json:"id"`
    Titulo       string    `json:"titulo"`
    Descripcion  *string   `json:"descripcion,omitempty"`
    Estado       string    `json:"estado"`        // borrador|en_revision|aprobado|rechazado
    CreadorID    string    `json:"creadorId"`
    PublicadoEn  *time.Time `json:"publicadoEn,omitempty"`
    CreatedAt    time.Time `json:"createdAt"`
}

type CourseDetailResponse struct {
    CourseResponse
    Secciones []SectionResponse `json:"secciones"`
    Materiales []MaterialResponse `json:"materiales"`
    Evaluacion *EvaluationSummary `json:"evaluacion,omitempty"`
}

type SectionResponse struct {
    ID     string         `json:"id"`
    Titulo string         `json:"titulo"`
    Orden  int            `json:"orden"`
    Videos []VideoResponse `json:"videos"`
}

// Estructura generica de paginacion (RT-10b)
type PaginatedResponse[T any] struct {
    Items      []T   `json:"items"`
    Page       int   `json:"page"`
    Size       int   `json:"size"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"totalPages"`
}

type ErrorResponse struct {
    Code    string            `json:"code"`              // ej: "VALIDATION_ERROR", "NOT_FOUND"
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"` // por-campo en errores de validacion
}
```

### Reglas Generales de DTOs

- **Casing JSON: camelCase** (alineado con el frontend Angular/TS).
- **Campos `created_at`/`updated_at` en DB se exponen como `createdAt`/`updatedAt`**.
- **Nunca exponer campos sensibles** (ej: `google_sub` interno) — son detalles de almacenamiento.
- **DTOs viven en el modulo**, no en un paquete global. Si un response tiene que cruzar modulos, exponelo desde la interface publica de ese modulo (ver seccion 14).

---

## 6. Patron de Handlers (Gin)

El handler es **delgado**: parsea el request, llama al service, mapea el resultado a respuesta HTTP. **Nada de logica de dominio aqui**.

### Estructura de un Handler

```go
// internal/modules/courses/handler/handler.go
package handler

import (
    "errors"
    "net/http"

    "github.com/gin-gonic/gin"

    "github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/dto"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
    "github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
)

type Handler struct {
    svc service.Service
}

func Register(rg *gin.RouterGroup, svc service.Service) {
    h := &Handler{svc: svc}
    g := rg.Group("/courses")
    {
        g.GET("",              h.Catalog)                              // todos
        g.GET("/:id",          h.GetByID)                              // todos
        g.POST("",             middleware.RequireRole("creador"), h.Create)
        g.PATCH("/:id",        middleware.RequireRole("creador"), h.Update)
        g.POST("/:id/submit",  middleware.RequireRole("creador"), h.SubmitForReview)
    }
}

// Catalog godoc
// @Summary  Catalogo de cursos aprobados
// @Tags     courses
// @Param    page query int    false "Pagina"
// @Param    size query int    false "Tamano de pagina"
// @Param    q    query string false "Busqueda"
// @Success  200 {object} dto.PaginatedResponse[dto.CourseResponse]
// @Router   /courses [get]
// @Security BearerAuth
func (h *Handler) Catalog(c *gin.Context) {
    var q dto.CatalogQuery
    if err := c.ShouldBindQuery(&q); err != nil {
        httperr.Render(c, httperr.BadRequest("parametros invalidos", err))
        return
    }
    res, err := h.svc.Catalog(c.Request.Context(), q)
    if err != nil {
        httperr.Render(c, err)
        return
    }
    c.JSON(http.StatusOK, res)
}

// Create godoc
// @Summary  Crea un curso en borrador (RF-05)
// @Tags     courses
// @Accept   json
// @Param    body body dto.CreateCourseRequest true "Datos del curso"
// @Success  201 {object} dto.CourseResponse
// @Failure  400 {object} dto.ErrorResponse
// @Router   /courses [post]
// @Security BearerAuth
func (h *Handler) Create(c *gin.Context) {
    var req dto.CreateCourseRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httperr.Render(c, httperr.BadRequest("body invalido", err))
        return
    }
    userID := middleware.UserIDFrom(c)
    res, err := h.svc.Create(c.Request.Context(), userID, req)
    if err != nil {
        httperr.Render(c, err)
        return
    }
    c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetByID(c *gin.Context) {
    res, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
    if err != nil {
        if errors.Is(err, service.ErrNotFound) {
            httperr.Render(c, httperr.NotFound("curso no encontrado"))
            return
        }
        httperr.Render(c, err)
        return
    }
    c.JSON(http.StatusOK, res)
}
```

### Convenciones de Rutas HTTP

| Verbo + Ruta | Operacion | Status exitoso |
|--------------|-----------|----------------|
| `GET /resource` | Listar (paginado) | `200 OK` |
| `GET /resource/:id` | Obtener uno | `200 OK` |
| `POST /resource` | Crear | `201 Created` |
| `PATCH /resource/:id` | Actualizar parcial | `200 OK` |
| `PUT /resource/:id` | Reemplazar completo (raro) | `200 OK` |
| `DELETE /resource/:id` | Eliminar (logico) | `204 No Content` |
| `POST /resource/:id/action` | Accion no-CRUD (ej: `/courses/:id/submit`) | `200 OK` o `202 Accepted` |

### Anti-patrones a Evitar en Handlers

- ❌ **Acceso directo a la BD**: `db.Where(...)` no aparece en `handler/`. Va en `repository/`.
- ❌ **Logica de dominio**: validar reglas de negocio (ej: "solo el creador puede editar"), calcular puntajes, decidir transiciones de estado — todo eso vive en el service.
- ❌ **Manejo de errores con `c.JSON(500, gin.H{"error": ...})`**: usar `httperr.Render` (ver seccion 15) para tener respuestas consistentes.
- ❌ **Logica condicional compleja**: si un handler tiene `if`/`switch` que decide el flujo, es senal de que el service deberia exponer un metodo mas alto-nivel.

---

## 7. Patron de Services (Logica de Dominio)

El service **es** el dominio. Aplica reglas de negocio, orquesta repositories y otros modulos, **devuelve errores de dominio tipados**.

### Estructura de un Service

```go
// internal/modules/courses/service/service.go
package service

import (
    "context"
    "errors"
    "time"

    "github.com/google/uuid"

    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/domain"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/dto"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations"
    "github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
)

// Errores de dominio — el handler los traduce a HTTP status (seccion 15).
var (
    ErrNotFound          = errors.New("curso no encontrado")
    ErrNotOwner          = errors.New("no eres el creador del curso")
    ErrInvalidTransition = errors.New("transicion de estado no permitida")
)

type Service interface {
    Catalog(ctx context.Context, q dto.CatalogQuery) (dto.PaginatedResponse[dto.CourseResponse], error)
    GetByID(ctx context.Context, id string) (dto.CourseDetailResponse, error)
    Create(ctx context.Context, creatorID string, req dto.CreateCourseRequest) (dto.CourseResponse, error)
    SubmitForReview(ctx context.Context, creatorID, courseID string) error
    Publish(ctx context.Context, courseID string) error
}

type service struct {
    repo    repository.Repository
    storage storage.Client
    evals   evaluations.Service
}

func New(r repository.Repository, s storage.Client, e evaluations.Service) Service {
    return &service{repo: r, storage: s, evals: e}
}

func (s *service) Catalog(ctx context.Context, q dto.CatalogQuery) (dto.PaginatedResponse[dto.CourseResponse], error) {
    // Catalogo solo muestra cursos APROBADOS (RF-16)
    items, total, err := s.repo.SearchPublished(ctx, q.Q, q.Page, q.Size)
    if err != nil {
        return dto.PaginatedResponse[dto.CourseResponse]{}, err
    }
    return dto.PaginatedResponse[dto.CourseResponse]{
        Items:      toCourseResponses(items),
        Page:       q.Page,
        Size:       q.Size,
        Total:      total,
        TotalPages: int((total + int64(q.Size) - 1) / int64(q.Size)),
    }, nil
}

func (s *service) Create(ctx context.Context, creatorID string, req dto.CreateCourseRequest) (dto.CourseResponse, error) {
    c := domain.Course{
        ID:          uuid.NewString(),
        CreadorID:   creatorID,
        Titulo:      req.Titulo,
        Descripcion: req.Descripcion,
        Estado:      domain.EstadoBorrador,
        CreatedAt:   time.Now().UTC(),
    }
    if err := s.repo.Create(ctx, &c); err != nil {
        return dto.CourseResponse{}, err
    }
    return toCourseResponse(c), nil
}

func (s *service) SubmitForReview(ctx context.Context, creatorID, courseID string) error {
    c, err := s.repo.GetByID(ctx, courseID)
    if err != nil {
        return err
    }
    if c.CreadorID != creatorID {
        return ErrNotOwner
    }
    if c.Estado != domain.EstadoBorrador && c.Estado != domain.EstadoRechazado {
        return ErrInvalidTransition
    }
    // Validacion de dominio: un curso enviado a revision debe tener al menos un video y una evaluacion (RF-08, RF-10)
    if !s.repo.HasContent(ctx, courseID) {
        return errors.New("el curso debe tener al menos un video antes de enviarse a revision")
    }
    if _, err := s.evals.GetByCourseID(ctx, courseID); err != nil {
        return errors.New("el curso debe tener evaluacion definida")
    }
    return s.repo.UpdateEstado(ctx, courseID, domain.EstadoEnRevision)
}

// Publish es llamado por el modulo approvals cuando se aprueba el curso (RF-16).
func (s *service) Publish(ctx context.Context, courseID string) error {
    now := time.Now().UTC()
    return s.repo.MarkAprobado(ctx, courseID, now)
}
```

### Reglas del Service

- **Recibe `context.Context` en cada metodo** y lo propaga a repos/storage/otros modulos. Imprescindible para cancelacion y deadlines (timeouts en handlers).
- **Errores de dominio como sentinel `var Err...`**. El handler los traduce a HTTP status. Nunca devuelvas `fmt.Errorf("not found")` — no se puede comparar con `errors.Is`.
- **Recibe IDs/DTOs, devuelve DTOs**. **No exponer entidades GORM** fuera del modulo: pueden cambiar (rename de columna, agregar campos sensibles) sin romper la API.
- **No accede a `*gin.Context`**. Si necesitas request-scoped data (ej: el `userID` autenticado), recibilo como parametro. Esto hace al service trivialmente testeable sin levantar Gin.

### Funciones de Mapeo (Domain ↔ DTO)

Convencion: archivo `mapper.go` o helpers `toXxxResponse(...)` al pie del service.

```go
func toCourseResponse(c domain.Course) dto.CourseResponse {
    return dto.CourseResponse{
        ID:          c.ID,
        Titulo:      c.Titulo,
        Descripcion: c.Descripcion,
        Estado:      string(c.Estado),
        CreadorID:   c.CreadorID,
        PublicadoEn: c.PublicadoEn,
        CreatedAt:   c.CreatedAt,
    }
}

func toCourseResponses(items []domain.Course) []dto.CourseResponse {
    out := make([]dto.CourseResponse, len(items))
    for i, c := range items {
        out[i] = toCourseResponse(c)
    }
    return out
}
```

---

## 8. Patron de Repositories (GORM)

El repository encapsula GORM. **Es el unico lugar donde aparece `db.Where(...)`, `db.Find(...)`, etc.** Si manana cambias GORM por `sqlc` o `pgx`, solo afectas esta capa.

### Estructura de un Repository

```go
// internal/modules/courses/repository/repository.go
package repository

import (
    "context"
    "errors"
    "strings"
    "time"

    "gorm.io/gorm"

    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/domain"
    coursesvc "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
)

type Repository interface {
    Create(ctx context.Context, c *domain.Course) error
    GetByID(ctx context.Context, id string) (domain.Course, error)
    SearchPublished(ctx context.Context, q string, page, size int) ([]domain.Course, int64, error)
    UpdateEstado(ctx context.Context, id string, estado domain.Estado) error
    MarkAprobado(ctx context.Context, id string, at time.Time) error
    HasContent(ctx context.Context, id string) bool
}

type repo struct {
    db *gorm.DB
}

func New(db *gorm.DB) Repository {
    return &repo{db: db}
}

func (r *repo) Create(ctx context.Context, c *domain.Course) error {
    return r.db.WithContext(ctx).Create(c).Error
}

func (r *repo) GetByID(ctx context.Context, id string) (domain.Course, error) {
    var c domain.Course
    err := r.db.WithContext(ctx).
        Preload("Secciones.Videos").
        Preload("Materiales").
        First(&c, "id = ?", id).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return domain.Course{}, coursesvc.ErrNotFound
    }
    return c, err
}

func (r *repo) SearchPublished(ctx context.Context, q string, page, size int) ([]domain.Course, int64, error) {
    var (
        items []domain.Course
        total int64
    )
    qb := r.db.WithContext(ctx).Model(&domain.Course{}).Where("estado = ?", domain.EstadoAprobado)
    if strings.TrimSpace(q) != "" {
        qb = qb.Where("titulo ILIKE ?", "%"+q+"%")
    }
    if err := qb.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    if err := qb.
        Order("publicado_en DESC NULLS LAST").
        Limit(size).
        Offset((page - 1) * size).
        Find(&items).Error; err != nil {
        return nil, 0, err
    }
    return items, total, nil
}

func (r *repo) UpdateEstado(ctx context.Context, id string, estado domain.Estado) error {
    return r.db.WithContext(ctx).
        Model(&domain.Course{}).
        Where("id = ?", id).
        Update("estado", estado).Error
}
```

### Reglas del Repository

- **Siempre `WithContext(ctx)`**: propaga cancelacion y deadlines. Una query sin contexto en una request HTTP es un bug.
- **Devuelve errores de dominio**: traduce `gorm.ErrRecordNotFound` a `service.ErrNotFound` para que el service no dependa de GORM.
- **No exponer queries GORM al service**: si el service necesita `Order` o `Limit`, expone metodos especificos (`SearchPublished`, `ListByCreator`) en vez de devolver un `*gorm.DB`.
- **Paginacion siempre con `Count` + `Limit/Offset`**. Para grandes volumenes, considerar cursor pagination en el futuro.
- **Solo el repo del modulo dueno toca sus tablas**. `courses/repository` jamas hace `JOIN` contra `user` ni `attempt`.

### Transacciones

Cuando una operacion abarca multiples tablas **del mismo modulo**, usar transaccion:

```go
func (r *repo) CreateWithSections(ctx context.Context, c *domain.Course, sections []domain.Section) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Create(c).Error; err != nil {
            return err
        }
        for i := range sections {
            sections[i].CourseID = c.ID
        }
        return tx.Create(&sections).Error
    })
}
```

**No usar transacciones cross-modulo**. Si una operacion necesita coordinar varios modulos, modelarla como **secuencia con compensacion** (saga light) o como **evento interno** asincrono — esto preserva la extractibilidad a microservicios (recomendacion 5.6 del doc tecnico).

---

## 9. Sistema de Autenticacion (Google OAuth + JWT)

### Flujo Completo

```
Frontend                       Backend (auth module)              Google
   │                              │                                  │
   ├── google.accounts.id.prompt()─────────────────────────────────>│
   │                              │                                  │
   │<─── credential (ID token de Google, firmado por Google) ───────│
   │                              │                                  │
   ├── POST /api/auth/google ───>│                                  │
   │     { idToken }              │                                  │
   │                              ├── idtoken.Validate(ctx,...)─────>│
   │                              │<─── claims (sub, email, hd) ────│
   │                              │                                  │
   │                              ├── Verifica hd == GOOGLE_HOSTED_DOMAIN (RT-13)
   │                              ├── upsert user (google_sub UNIQUE)
   │                              ├── carga roles del usuario
   │                              ├── emite access JWT (1h) + refresh token opaco (7d)
   │                              ├── persiste hash(refresh_token) en tabla refresh_token
   │                              │
   │<─── { access_token, refresh_token, expires_at, user } ──┤
   │                              │
   ├── peticiones con `Authorization: Bearer <access_token>`────>│
                                  ├── middleware JWT valida firma + exp
                                  ├── inyecta userID y roles en context
                                  └── handler/service procesan

   Cuando access expira (o esta por hacerlo):
   ├── POST /api/auth/refresh { refreshToken } ──>│
                                  ├── hashea, busca en refresh_token
                                  ├── valida activo + no usado (deteccion de replay)
                                  ├── revoca el viejo, emite uno nuevo (rotacion)
   │<─── { access_token, refresh_token, expires_at, user } ──┤

   Logout:
   ├── POST /api/auth/logout { refreshToken } ──>│
                                  └── revoca el refresh; idempotente (204 No Content)
```

### Endpoint POST /api/auth/google

```go
// internal/modules/auth/service/service.go
package service

import (
    "context"
    "errors"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "google.golang.org/api/idtoken"

    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/users"
)

var (
    ErrInvalidGoogleToken  = errors.New("token de google invalido")
    ErrUnauthorizedDomain  = errors.New("cuenta no pertenece al dominio corporativo")
    ErrInvalidRefreshToken = errors.New("refresh token invalido o expirado")
    ErrRefreshTokenReused  = errors.New("refresh token reutilizado: posible ataque, sesiones revocadas")
)

type Service interface {
    LoginWithGoogle(ctx context.Context, idTokenStr string) (LoginResponse, error)
    Refresh(ctx context.Context, refreshToken string) (LoginResponse, error)
    Logout(ctx context.Context, refreshToken string) error
}

type Config struct {
    JWTSecret             string
    JWTExpiresIn          time.Duration
    RefreshTokenExpiresIn time.Duration
    GoogleClientID        string
    GoogleHostedDomain    string
}

type LoginResponse struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
    User         UserDTO   `json:"user"`
}

type UserDTO struct {
    ID     string   `json:"id"`
    Email  string   `json:"email"`
    Nombre string   `json:"nombre"`
    Roles  []string `json:"roles"`
}

type service struct {
    cfg   Config
    users users.Service
    repo  Repository // store de refresh tokens
}

func New(cfg Config, u users.Service, r Repository) Service {
    return &service{cfg: cfg, users: u, repo: r}
}

func (s *service) LoginWithGoogle(ctx context.Context, idTokenStr string) (LoginResponse, error) {
    payload, err := idtoken.Validate(ctx, idTokenStr, s.cfg.GoogleClientID)
    if err != nil {
        return LoginResponse{}, ErrInvalidGoogleToken
    }
    // RT-13: el dominio del email debe coincidir con el dominio corporativo
    hd, _ := payload.Claims["hd"].(string)
    if hd != s.cfg.GoogleHostedDomain {
        return LoginResponse{}, ErrUnauthorizedDomain
    }

    sub, _ := payload.Claims["sub"].(string)
    email, _ := payload.Claims["email"].(string)
    name, _ := payload.Claims["name"].(string)

    // Upsert por google_sub (estable, no cambia si el email cambia)
    u, err := s.users.UpsertFromGoogle(ctx, users.GoogleProfile{
        GoogleSub: sub,
        Email:     email,
        Nombre:    name,
    })
    if err != nil {
        return LoginResponse{}, err
    }

    accessToken, exp, err := s.issueJWT(u)
    if err != nil {
        return LoginResponse{}, err
    }
    refreshToken, err := s.issueRefreshToken(ctx, u.ID, "")
    if err != nil {
        return LoginResponse{}, err
    }

    return LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresAt:    exp,
        User: UserDTO{
            ID:     u.ID,
            Email:  u.Email,
            Nombre: u.Nombre,
            Roles:  u.Roles,
        },
    }, nil
}

func (s *service) issueJWT(u users.User) (string, time.Time, error) {
    exp := time.Now().UTC().Add(s.cfg.JWTExpiresIn)
    claims := jwt.MapClaims{
        "sub":    u.ID,
        "email":  u.Email,
        "nombre": u.Nombre,
        "roles":  u.Roles, // []string, ej: ["alumno","creador"]
        "exp":    exp.Unix(),
        "iat":    time.Now().Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
    return signed, exp, err
}
```

### Refresh Tokens (POST /api/auth/refresh)

El access token (JWT) tiene vida corta (1h). El refresh token tiene vida larga (7 dias por defecto) y se usa exclusivamente para obtener un nuevo access sin volver a pasar por el flujo de Google. Es **opaco** (no es un JWT — es un string aleatorio de 32 bytes en base64) y se almacena hasheado en la BD junto con metadata de la sesion.

**Por que necesitamos refresh tokens en este LMS:**
- Un alumno mirando un video de 60-90 minutos no debe ser deslogueado a mitad por expirar el JWT.
- Mantener JWTs cortos limita el blast radius si uno se filtra (logs, XSS, etc.).
- Permite revocar sesiones individualmente (logout-this-device) o todas (logout-all) sin esperar a que expire el JWT.

#### Tabla `refresh_token` (modulo auth)

```sql
CREATE TABLE refresh_token (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    token_hash   text NOT NULL UNIQUE,           -- SHA-256 del token plano
    parent_id    uuid REFERENCES refresh_token(id) ON DELETE SET NULL, -- chain de rotacion
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz,                    -- NULL = activo
    used_at      timestamptz,                    -- NULL = no consumido aun
    user_agent   text,
    ip           inet,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_token_user_id ON refresh_token(user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_refresh_token_hash ON refresh_token(token_hash);
```

**Por que esta tabla pertenece al modulo `auth` (y no a `users`):** auth es dueno del ciclo de vida de la sesion; users no debe conocer detalles de tokens. Si manana auth se vuelve microservicio, esta tabla se va con el.

**Por que `token_hash` y no el token plano:** si la BD se filtra, el atacante no puede usar los refresh tokens sin invertir SHA-256. Es la misma logica que hashear passwords (aunque aqui no haya un humano eligiendo el secret).

**Por que `parent_id`:** permite detectar **reuse de refresh tokens rotados** — el ataque tipico cuando un atacante roba un refresh token. Ver "Deteccion de reuse" mas abajo.

#### Repository del modulo auth

```go
// internal/modules/auth/repository/repository.go
package repository

import (
    "context"
    "time"

    "gorm.io/gorm"
)

type RefreshToken struct {
    ID         string     `gorm:"type:uuid;primaryKey"`
    UserID     string     `gorm:"type:uuid;not null;index"`
    TokenHash  string     `gorm:"type:text;not null;uniqueIndex"`
    ParentID   *string    `gorm:"type:uuid"`
    ExpiresAt  time.Time  `gorm:"type:timestamptz;not null"`
    RevokedAt  *time.Time `gorm:"type:timestamptz"`
    UsedAt     *time.Time `gorm:"type:timestamptz"`
    UserAgent  string     `gorm:"type:text"`
    IP         string     `gorm:"type:inet"`
    CreatedAt  time.Time  `gorm:"type:timestamptz;default:now()"`
}

func (RefreshToken) TableName() string { return "refresh_token" }

type Repository interface {
    Insert(ctx context.Context, rt *RefreshToken) error
    FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
    MarkUsed(ctx context.Context, id string, at time.Time) error
    Revoke(ctx context.Context, id string, at time.Time) error
    RevokeChain(ctx context.Context, rootID string) error   // revoca toda la cadena desde un nodo
    RevokeAllForUser(ctx context.Context, userID string) error
}
```

#### Emision del refresh token

```go
import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"

    "github.com/google/uuid"

    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/repository"
)

func (s *service) issueRefreshToken(ctx context.Context, userID string, parentID string) (string, error) {
    // 1. Generar 32 bytes aleatorios → ~43 chars base64url
    raw := make([]byte, 32)
    if _, err := rand.Read(raw); err != nil {
        return "", err
    }
    tokenPlain := base64.RawURLEncoding.EncodeToString(raw)

    // 2. Hashear con SHA-256 antes de persistir
    sum := sha256.Sum256([]byte(tokenPlain))
    tokenHash := hex.EncodeToString(sum[:])

    var parent *string
    if parentID != "" {
        parent = &parentID
    }
    rt := &repository.RefreshToken{
        ID:        uuid.NewString(),
        UserID:    userID,
        TokenHash: tokenHash,
        ParentID:  parent,
        ExpiresAt: time.Now().UTC().Add(s.cfg.RefreshTokenExpiresIn),
    }
    if err := s.repo.Insert(ctx, rt); err != nil {
        return "", err
    }
    // El backend NUNCA mas vera el token plano — solo el cliente lo guarda
    return tokenPlain, nil
}
```

#### Refresh (rotacion obligatoria + deteccion de reuse)

```go
func (s *service) Refresh(ctx context.Context, refreshTokenPlain string) (LoginResponse, error) {
    sum := sha256.Sum256([]byte(refreshTokenPlain))
    hash := hex.EncodeToString(sum[:])

    rt, err := s.repo.FindByHash(ctx, hash)
    if err != nil || rt == nil {
        return LoginResponse{}, ErrInvalidRefreshToken
    }
    if rt.RevokedAt != nil || time.Now().UTC().After(rt.ExpiresAt) {
        return LoginResponse{}, ErrInvalidRefreshToken
    }

    // ─── Deteccion de reuse (OWASP) ────────────────────────────
    // Si este token YA fue usado antes (UsedAt != nil), es un replay:
    // alguien tiene una copia robada y la esta usando despues que
    // el cliente legitimo ya roto a uno nuevo. Revocar toda la cadena.
    if rt.UsedAt != nil {
        _ = s.repo.RevokeAllForUser(ctx, rt.UserID)
        return LoginResponse{}, ErrRefreshTokenReused
    }

    // ─── Rotacion ─────────────────────────────────────────────
    now := time.Now().UTC()
    if err := s.repo.MarkUsed(ctx, rt.ID, now); err != nil {
        return LoginResponse{}, err
    }
    if err := s.repo.Revoke(ctx, rt.ID, now); err != nil {
        return LoginResponse{}, err
    }

    u, err := s.users.GetByID(ctx, rt.UserID)
    if err != nil {
        return LoginResponse{}, err
    }

    accessToken, exp, err := s.issueJWT(u)
    if err != nil {
        return LoginResponse{}, err
    }
    newRefresh, err := s.issueRefreshToken(ctx, u.ID, rt.ID) // parent_id = rt.ID
    if err != nil {
        return LoginResponse{}, err
    }

    return LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: newRefresh,
        ExpiresAt:    exp,
        User:         UserDTO{ID: u.ID, Email: u.Email, Nombre: u.Nombre, Roles: u.Roles},
    }, nil
}
```

**Reglas no negociables:**
- **Rotacion siempre**: cada `/refresh` exitoso revoca el viejo y emite uno nuevo. NO se permite reuso.
- **Deteccion de replay**: si un token ya marcado como `used` se presenta de nuevo, se asume robo y se revocan TODOS los refresh tokens del usuario (vuelve a loguearse via Google).
- **Constante en validacion**: comparar hashes con tiempo constante si fuera el caso de uso (aqui no es critico porque hex de SHA-256 ya es uniforme).

#### Logout (POST /api/auth/logout)

```go
func (s *service) Logout(ctx context.Context, refreshTokenPlain string) error {
    if refreshTokenPlain == "" {
        return nil // logout silencioso si no hay token
    }
    sum := sha256.Sum256([]byte(refreshTokenPlain))
    hash := hex.EncodeToString(sum[:])

    rt, err := s.repo.FindByHash(ctx, hash)
    if err != nil || rt == nil {
        return nil // idempotente: ya estaba invalido
    }
    return s.repo.Revoke(ctx, rt.ID, time.Now().UTC())
}
```

**Notas:**
- Logout es **idempotente**: invocarlo con un token ya revocado retorna OK.
- El access token (JWT) no se "revoca" — simplemente expira en `JWT_EXPIRES_IN` (1h). Si necesitas revocacion inmediata del JWT (caso raro), implementar un blacklist en Redis con el `jti` del JWT.

#### Endpoint handlers

```go
// internal/modules/auth/handler/handler.go

// Refresh godoc
// @Summary  Renueva el access token usando el refresh token
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body dto.RefreshRequest true "Refresh token"
// @Success  200 {object} dto.LoginResponse
// @Failure  401 {object} dto.ErrorResponse
// @Router   /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
    var req dto.RefreshRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httperr.Render(c, httperr.BadRequest("body invalido", err))
        return
    }
    res, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
    if err != nil {
        switch {
        case errors.Is(err, service.ErrInvalidRefreshToken),
             errors.Is(err, service.ErrRefreshTokenReused):
            httperr.Render(c, &httperr.Error{
                Status: http.StatusUnauthorized, Code: "INVALID_REFRESH", Message: err.Error(),
            })
        default:
            httperr.Render(c, httperr.Internal(err))
        }
        return
    }
    c.JSON(http.StatusOK, res)
}

// Logout godoc
// @Summary  Revoca el refresh token de la sesion actual
// @Tags     auth
// @Accept   json
// @Param    body body dto.RefreshRequest true "Refresh token a revocar"
// @Success  204
// @Router   /auth/logout [post]
// @Security BearerAuth
func (h *Handler) Logout(c *gin.Context) {
    var req dto.RefreshRequest
    _ = c.ShouldBindJSON(&req) // ignora errores: logout silencioso
    _ = h.svc.Logout(c.Request.Context(), req.RefreshToken)
    c.Status(http.StatusNoContent)
}
```

DTO:

```go
// internal/modules/auth/dto/request.go
type RefreshRequest struct {
    RefreshToken string `json:"refreshToken" binding:"required"`
}
```

#### Limpieza de refresh tokens expirados

Un job periodico (cron o ticker en `cmd/api/main.go`) elimina rows con `expires_at < now() - 30d` para que la tabla no crezca indefinidamente. **No borrar antes** de los 30 dias: los tokens revocados sirven como evidencia forense ante incidentes.

```sql
DELETE FROM refresh_token WHERE expires_at < now() - interval '30 days';
```

Implementacion sugerida: una goroutine en `main.go` con `time.NewTicker(24 * time.Hour)` que ejecuta la query y loggea cuantas rows borro.

### Middleware JWT

```go
// internal/middleware/jwt.go
package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
    ctxUserID ctxKey = "userID"
    ctxRoles  ctxKey = "roles"
)

func JWT(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        h := c.GetHeader("Authorization")
        if !strings.HasPrefix(h, "Bearer ") {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "token requerido"})
            return
        }
        tokenStr := strings.TrimPrefix(h, "Bearer ")

        token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
            if t.Method != jwt.SigningMethodHS256 {
                return nil, jwt.ErrSignatureInvalid
            }
            return []byte(secret), nil
        })
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "token invalido"})
            return
        }
        claims, _ := token.Claims.(jwt.MapClaims)

        userID, _ := claims["sub"].(string)
        roles := toStringSlice(claims["roles"])

        c.Set(string(ctxUserID), userID)
        c.Set(string(ctxRoles), roles)
        c.Next()
    }
}

// Helpers para acceder desde handlers.
func UserIDFrom(c *gin.Context) string {
    v, _ := c.Get(string(ctxUserID))
    s, _ := v.(string)
    return s
}

func RolesFrom(c *gin.Context) []string {
    v, _ := c.Get(string(ctxRoles))
    r, _ := v.([]string)
    return r
}

func toStringSlice(v interface{}) []string {
    arr, ok := v.([]interface{})
    if !ok {
        return nil
    }
    out := make([]string, 0, len(arr))
    for _, x := range arr {
        if s, ok := x.(string); ok {
            out = append(out, s)
        }
    }
    return out
}
```

### Variables de Entorno

| Variable | Default | Notas |
|----------|---------|-------|
| `JWT_SECRET` | (requerido) | HS256 secret del access token, minimo 32 caracteres |
| `JWT_EXPIRES_IN` | `1h` | Duracion del access token (Go duration string) |
| `REFRESH_TOKEN_EXPIRES_IN` | `168h` | Duracion del refresh token (default 7 dias) |
| `GOOGLE_CLIENT_ID` | (requerido) | OAuth Client ID publico |
| `GOOGLE_CLIENT_SECRET` | (opcional) | Solo si se usa code flow server-side |
| `GOOGLE_HOSTED_DOMAIN` | (requerido) | Dominio corporativo (RT-13) |
| `GOOGLE_REDIRECT_URI` | (opcional) | Solo si se usa code flow |

---

## 10. Sistema de Roles y Autorizacion (RBAC)

Los **4 roles del dominio** son fijos (RF-02, tabla `role` del modelo de datos): `alumno`, `creador`, `supervisor`, `administrador`. Un usuario puede tener varios simultaneos (RF-03 → tabla `user_role`).

**No existe un sistema de permisos granulares**: el modelo de datos define unicamente la tabla `role` (no hay `permission` ni `role_permission`), y los requerimientos funcionales solo mencionan control por rol (RF-02, RF-04, RNF-04). Toda decision de autorizacion se toma evaluando el conjunto de roles del usuario; las reglas mas finas (ownership) viven en los services.

### Middleware RequireRole

```go
// internal/middleware/rbac.go
package middleware

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

func RequireRole(allowed ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRoles := RolesFrom(c)
        for _, ur := range userRoles {
            for _, ar := range allowed {
                if ur == ar {
                    c.Next()
                    return
                }
            }
        }
        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
            "code":    "FORBIDDEN",
            "message": "rol insuficiente",
        })
    }
}
```

### Uso en Handlers

```go
g.POST("",            middleware.RequireRole("creador"), h.Create)
g.POST("/:id/approve", middleware.RequireRole("administrador"), h.Approve) // RF-15
g.GET("/team",         middleware.RequireRole("supervisor"), h.Team)         // RF-24
```

### Reglas Mas Finas (Ownership)

El middleware solo valida **roles**. Reglas como "solo el creador puede editar su curso" viven en el **service**:

```go
if c.CreadorID != userID && !contains(userRoles, "administrador") {
    return ErrNotOwner
}
```

### Roles iniciales (seed)

`backend/cmd/seed/main.go` debe poblar la tabla `role` con los 4 roles fijos al primer arranque:

```go
for _, name := range []string{"alumno", "creador", "supervisor", "administrador"} {
    db.FirstOrCreate(&domain.Role{Nombre: name}, "nombre = ?", name)
}
```

Por defecto, todo usuario nuevo creado via Google OAuth recibe el rol `alumno` automaticamente (RF-18 implica que cualquier trabajador puede inscribirse). El administrador asigna roles adicionales segun RF-04.

---

## 11. GORM: Modelos y Convenciones

### Convenciones Generales (del documento de datos seccion 1)

- **Identificadores**: `uuid` para entidades de dominio (`gen_random_uuid()` o `uuid.NewString()` en aplicacion). `bigserial` solo para catalogos pequenos (`role`).
- **Timestamps**: `created_at` y `updated_at` como `timestamptz` con default `now()`.
- **Estados**: `text` + `CHECK` constraint, no enums Postgres (portabilidad, evita migraciones de tipo).
- **Borrado**: logico (campo `activo` o `estado`). Nunca `DELETE` fisico en `user`, `course`, `evaluation`.
- **Propiedad por modulo**: cada tabla pertenece a un solo modulo (RT-19).

### Modelo Estandar (Course)

```go
// internal/modules/courses/domain/models.go
package domain

import "time"

type Estado string

const (
    EstadoBorrador   Estado = "borrador"
    EstadoEnRevision Estado = "en_revision"
    EstadoAprobado   Estado = "aprobado"
    EstadoRechazado  Estado = "rechazado"
)

type Course struct {
    ID           string     `gorm:"type:uuid;primaryKey"`
    CreadorID    string     `gorm:"type:uuid;not null;index"`
    Titulo       string     `gorm:"type:text;not null"`
    Descripcion  *string    `gorm:"type:text"`
    Estado       Estado     `gorm:"type:text;not null;default:'borrador'"`
    PublicadoEn  *time.Time `gorm:"type:timestamptz"`
    CreatedAt    time.Time  `gorm:"type:timestamptz;default:now()"`
    UpdatedAt    time.Time  `gorm:"type:timestamptz;default:now()"`

    Secciones  []Section  `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE"`
    Materiales []Material `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE"`
}

func (Course) TableName() string { return "course" }
```

### Convenciones GORM

- **`TableName()`** explicito por modelo, para mantener nombres consistentes con el DDL del documento (singular: `user`, `course`, `evaluation`, no `users`/`courses`).
- **Foreign keys con `constraint`**: declara `OnDelete:CASCADE`/`OnDelete:RESTRICT` segun el documento de datos.
- **Indices**: declara `index` en columnas usadas en `WHERE` (FKs, estados, fechas de ordenamiento).
- **Punteros para nullable**: `*string`, `*time.Time` para columnas `NULL`. GORM trata cero-valor como NULL solo si el tipo es punter.
- **Hooks GORM (`BeforeCreate`, `AfterUpdate`)**: usarlos con MUCHA precaucion. Esconden side-effects. **Preferir orquestacion explicita en el service**.

### Many-to-Many

```go
// internal/modules/users/domain/models.go
type User struct {
    ID         string    `gorm:"type:uuid;primaryKey"`
    GoogleSub  string    `gorm:"type:text;not null;uniqueIndex"`
    Email      string    `gorm:"type:text;not null;uniqueIndex"`
    Nombre     string    `gorm:"type:text;not null"`
    Activo     bool      `gorm:"not null;default:true"`
    CreatedAt  time.Time `gorm:"type:timestamptz;default:now()"`
    UpdatedAt  time.Time `gorm:"type:timestamptz;default:now()"`

    Roles []Role `gorm:"many2many:user_role;joinForeignKey:UserID;joinReferences:RoleID"`
}

type Role struct {
    ID     int64  `gorm:"primaryKey"`
    Nombre string `gorm:"type:text;not null;uniqueIndex"` // alumno|creador|supervisor|administrador
}
```

### Soft Delete (Borrado Logico)

GORM tiene `gorm.DeletedAt`, pero el modelo de datos prefiere borrado logico via **estado** o flag `activo`. Ejemplo para usuarios:

```go
// users.Service.Inactivate (no DELETE fisico, preserva historial — RT-20b)
func (s *service) Inactivate(ctx context.Context, id string) error {
    return s.repo.SetActivo(ctx, id, false)
}
```

Asi se conserva FK integrity: los intentos, puntajes y certificados del usuario siguen referenciandolo.

### GORM AutoMigrate: ¿Usarlo?

**No en produccion**. Solo en tests con SQLite efimero o en `make db-migrate-new` cuando explorando localmente. Las migraciones reproducibles van via `golang-migrate` (seccion 12). Esta postura cumple RT-18.

---

## 12. Migraciones con golang-migrate

### Estructura

```
backend/migrations/
├── 0001_init.up.sql
├── 0001_init.down.sql
├── 0002_add_courses_evaluations.up.sql
├── 0002_add_courses_evaluations.down.sql
└── ...
```

**Reglas:**
- Numeracion secuencial con padding (`0001`, `0002`, ...). El CLI lo genera automaticamente con `make db-migrate-new name=...`.
- Cada `.up.sql` debe tener su `.down.sql` correspondiente que **revierte exactamente** lo que el `up` hizo.
- **Nunca editar una migracion ya aplicada en main**. Crear una nueva que arregle lo anterior.

### Comandos (desde la raiz del monorepo)

```bash
make db-migrate-new name=add_certificates_table   # crea NNNN_add_certificates_table.{up,down}.sql
make db-migrate                                    # aplica todas las pendientes
make db-migrate-down                               # revierte la ultima
```

### Ejemplo: migracion inicial

```sql
-- 0001_init.up.sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ===== users =====
CREATE TABLE "user" (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub  text NOT NULL UNIQUE,
    email       text NOT NULL UNIQUE,
    nombre      text NOT NULL,
    activo      boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE role (
    id     bigserial PRIMARY KEY,
    nombre text NOT NULL UNIQUE CHECK (nombre IN ('alumno','creador','supervisor','administrador'))
);

CREATE TABLE user_role (
    user_id      uuid REFERENCES "user"(id) ON DELETE CASCADE,
    role_id      bigint REFERENCES role(id) ON DELETE RESTRICT,
    asignado_en  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE supervision (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    supervisor_id uuid NOT NULL REFERENCES "user"(id),
    empleado_id   uuid NOT NULL REFERENCES "user"(id),
    UNIQUE (supervisor_id, empleado_id),
    CHECK (supervisor_id <> empleado_id)
);
-- ... resto de tablas segun el modelo de datos
```

```sql
-- 0001_init.down.sql
DROP TABLE IF EXISTS supervision;
DROP TABLE IF EXISTS user_role;
DROP TABLE IF EXISTS role;
DROP TABLE IF EXISTS "user";
DROP EXTENSION IF EXISTS pgcrypto;
```

### Migraciones en Produccion

Ya cubierto en `GUIA-MONOREPO.md` seccion 7.4: job `migrate` separado en `docker-compose.yml` que corre `migrate up` y termina, con `backend` dependiendo de `service_completed_successfully`. Esto:

- Evita race conditions si escalas el backend horizontalmente.
- Si una migracion falla, el backend no arranca (visible en `docker ps`).
- Mantiene el binario backend chico (no embebe migrate).

---

## 13. Object Storage (S3 / MinIO) y URLs Prefirmadas

El material adjunto de los cursos se guarda en object storage; la BD solo guarda la referencia (RT-21, RT-22).

### Cliente Storage (internal/platform/storage)

```go
// internal/platform/storage/storage.go
package storage

import (
    "context"
    "errors"
    "fmt"
    "net/url"
    "time"

    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

type Client interface {
    PresignPutURL(ctx context.Context, key, contentType string, maxBytes int64) (string, error)
    PresignGetURL(ctx context.Context, key string) (string, error)
    Delete(ctx context.Context, key string) error
}

type Config struct {
    Endpoint   string
    Region     string
    Bucket     string
    AccessKey  string
    SecretKey  string
    UseSSL     bool
    PresignTTL time.Duration
}

type client struct {
    mc     *minio.Client
    bucket string
    ttl    time.Duration
}

func New(cfg Config) (Client, error) {
    endpoint, err := url.Parse(cfg.Endpoint)
    if err != nil {
        return nil, fmt.Errorf("storage endpoint invalido: %w", err)
    }
    host := endpoint.Host
    if host == "" {
        host = endpoint.Path
    }
    mc, err := minio.New(host, &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
        Secure: cfg.UseSSL,
        Region: cfg.Region,
    })
    if err != nil {
        return nil, err
    }
    return &client{mc: mc, bucket: cfg.Bucket, ttl: cfg.PresignTTL}, nil
}

func (c *client) PresignPutURL(ctx context.Context, key, contentType string, maxBytes int64) (string, error) {
    if maxBytes <= 0 {
        return "", errors.New("maxBytes invalido")
    }
    // Politica con limite de tamano enforced en la URL prefirmada (RT-24b)
    // Para limites estrictos, considerar PostPolicy en su lugar.
    reqURL, err := c.mc.PresignedPutObject(ctx, c.bucket, key, c.ttl)
    if err != nil {
        return "", err
    }
    return reqURL.String(), nil
}

func (c *client) PresignGetURL(ctx context.Context, key string) (string, error) {
    reqURL, err := c.mc.PresignedGetObject(ctx, c.bucket, key, c.ttl, url.Values{})
    if err != nil {
        return "", err
    }
    return reqURL.String(), nil
}

func (c *client) Delete(ctx context.Context, key string) error {
    return c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}
```

### Flujo de Upload de Material

```
Frontend                            Backend                            MinIO/S3
   │                                   │                                  │
   ├── POST /api/courses/:id/materials/presign ─>│
   │   { nombre, contentType, tamanoBytes }      │
   │                                   ├── valida courseId + ownership   │
   │                                   ├── valida tamano <= MAX_UPLOAD_BYTES (RT-24b)
   │                                   ├── valida contentType permitido
   │                                   ├── genera key: courses/<id>/materials/<uuid>-<nombre>
   │                                   ├── PresignPutURL(key, ...)──────>│
   │                                   │<──────── presigned URL ─────────┤
   │<── { uploadUrl, key, expiresIn } ─┤
   │                                   │
   ├── PUT uploadUrl (archivo binario, header Content-Type)──────────────>│
   │                                                                      │
   ├── POST /api/courses/:id/materials   (confirma upload)               │
   │   { key, nombre, contentType, tamanoBytes }                          │
   │                                   ├── persiste material en DB
   │<── 201 { material }              ─┤
```

**Para descarga:**
- En cursos publicos (RF-19) el material puede tener URL prefirmada GET corta vida.
- Para acceso restringido por rol, generar la URL en el momento de la peticion y enviarla al cliente con TTL bajo (15min default).

### Validaciones en el Handler de Presign

```go
func (h *Handler) PresignMaterial(c *gin.Context) {
    var req dto.PresignMaterialRequest
    if err := c.ShouldBindJSON(&req); err != nil { ... }

    if req.TamanoBytes > h.maxUploadBytes {
        httperr.Render(c, httperr.BadRequest("archivo excede el limite", nil))
        return
    }
    if !allowedMimeTypes[req.ContentType] {
        httperr.Render(c, httperr.BadRequest("tipo de archivo no permitido", nil))
        return
    }
    // ... llamada al service que retorna {uploadUrl, key, expiresAt}
}

var allowedMimeTypes = map[string]bool{
    "application/pdf":            true,
    "application/zip":            true,
    "application/msword":         true,
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
    "image/png":                  true,
    "image/jpeg":                 true,
}
```

---

## 14. Comunicacion entre Modulos (Interfaces)

Este es **el punto mas importante de la arquitectura** (RT-08, RT-19, RNFT-02).

### Regla de Oro

> Un modulo SOLO depende de la **interface publica** de otro modulo. NUNCA de sus structs, repositories, tablas, o paquetes internos.

### Ejemplo: courses depende de evaluations

```go
// courses/service usa la interface public de evaluations
import "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations"

func New(r repository.Repository, s storage.Client, e evaluations.Service) Service {
    return &service{repo: r, storage: s, evals: e}
}

func (s *service) SubmitForReview(ctx context.Context, ...) error {
    // ...
    if _, err := s.evals.GetByCourseID(ctx, courseID); err != nil {
        return errors.New("el curso debe tener evaluacion definida")
    }
    // ...
}
```

`evaluations.Service` es una interface declarada en `internal/modules/evaluations/evaluations.go`. Cuando manana `evaluations` se vuelva microservicio:

```go
// Hoy
evalsSvc := evaluations.NewService(...)

// Manana
evalsSvc := evaluationsclient.NewHTTPClient("http://evaluations-svc:3000")
```

`courses` no se entera. Misma interface, otra implementacion.

### Eventos Internos (Para Flujos Asincronos)

Algunos flujos son naturalmente asincronos (recomendacion 5.6 del doc tecnico):

- "Curso aprobado" → generar certificado al primer estudiante que lo termine
- "Intento aprobado" → emitir certificado + insignia + actualizar ranking

Implementacion incremental:

1. **Hoy (monolito)**: el service emisor llama directo a `s.certs.IssueOnPass(...)`.
2. **Manana (microservicios)**: el emisor publica en una cola (`courses.approved`), el consumidor procesa.

Estructurarlo desde hoy con una interface `EventBus` permite migrar sin reescribir logica:

```go
type EventBus interface {
    Publish(ctx context.Context, event string, payload any) error
}

// Hoy: implementacion in-memory que llama directo a handlers locales
// Manana: implementacion sobre RabbitMQ/Kafka/SQS
```

### Anti-patron: Transacciones Cross-Modulo

**Jamas hacer esto:**

```go
// MAL — transaccion que abarca tablas de varios modulos
db.Transaction(func(tx *gorm.DB) error {
    tx.Create(&course)
    tx.Create(&approval) // <- approvals
    tx.Create(&notification) // <- notifications
    return nil
})
```

Es imposible de mantener tras la separacion. En su lugar:

```go
// BIEN — secuencia explicita con compensacion si falla
if err := coursesSvc.Approve(ctx, id); err != nil { return err }
if err := approvalsSvc.Record(ctx, id, adminID, "aprobado"); err != nil {
    // Compensar: revertir el approve
    _ = coursesSvc.Revert(ctx, id)
    return err
}
```

Sí, es mas verboso. Sí, vale la pena: cuando `approvals` salga del monolito, el codigo no cambia.

---

## 15. Manejo de Errores Consistente

Todos los errores HTTP siguen el mismo shape (`dto.ErrorResponse` de seccion 5).

### Paquete httperr

```go
// internal/platform/httperr/httperr.go
package httperr

import (
    "errors"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/go-playground/validator/v10"
)

type Error struct {
    Status  int               `json:"-"`
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func BadRequest(msg string, cause error) *Error {
    e := &Error{Status: http.StatusBadRequest, Code: "VALIDATION_ERROR", Message: msg}
    if ve, ok := cause.(validator.ValidationErrors); ok {
        e.Details = map[string]string{}
        for _, f := range ve {
            e.Details[f.Field()] = f.Tag()
        }
    }
    return e
}

func NotFound(msg string) *Error {
    return &Error{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: msg}
}

func Forbidden(msg string) *Error {
    return &Error{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: msg}
}

func Conflict(msg string) *Error {
    return &Error{Status: http.StatusConflict, Code: "CONFLICT", Message: msg}
}

func Internal(cause error) *Error {
    return &Error{Status: http.StatusInternalServerError, Code: "INTERNAL", Message: "error interno"}
}

func Render(c *gin.Context, err error) {
    var he *Error
    if errors.As(err, &he) {
        c.AbortWithStatusJSON(he.Status, he)
        return
    }
    // Mapeo de sentinels de dominio conocidos
    // (cada modulo puede declarar los suyos y registrarlos via un switch local)
    c.AbortWithStatusJSON(http.StatusInternalServerError, &Error{
        Code: "INTERNAL", Message: "error interno",
    })
}
```

### Uso en el Handler

```go
res, err := h.svc.SubmitForReview(ctx, userID, courseID)
if err != nil {
    switch {
    case errors.Is(err, service.ErrNotFound):
        httperr.Render(c, httperr.NotFound("curso no encontrado"))
    case errors.Is(err, service.ErrNotOwner):
        httperr.Render(c, httperr.Forbidden("no eres el creador del curso"))
    case errors.Is(err, service.ErrInvalidTransition):
        httperr.Render(c, httperr.Conflict("estado no permite envio a revision"))
    default:
        httperr.Render(c, httperr.Internal(err))
    }
    return
}
```

### Reglas

- **El service devuelve `error` "puro"** (sentinels). No conoce HTTP.
- **El handler traduce** errores de dominio a `*httperr.Error`.
- **Loggear el `cause` original** en el middleware de recovery (con el request-id) — el cliente solo ve el mensaje publico.

---

## 16. Testing (go test, testify, testcontainers)

### Estrategia

| Nivel | Que valida | Herramienta | Ubicacion |
|-------|------------|-------------|-----------|
| Unit | Reglas de dominio del service con repo mock | testify | `service_test.go` al lado del codigo |
| Integration | Repository contra Postgres real | testcontainers-go | `repository_test.go` o `tests/integration/` con build tag |
| End-to-end | API completa con DB efimera | testcontainers + `httptest` | `tests/e2e/` con build tag |

### Unit Test de un Service

```go
// internal/modules/courses/service_test.go
package service_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/domain"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/dto"
    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
)

type repoMock struct{ mock.Mock }

func (m *repoMock) Create(ctx context.Context, c *domain.Course) error {
    args := m.Called(ctx, c)
    return args.Error(0)
}
// ... implementar resto de la interface

func TestCreate_OK(t *testing.T) {
    repo := new(repoMock)
    repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Course")).
        Return(nil)

    svc := service.New(repo, nil, nil)
    res, err := svc.Create(context.Background(), "user-1", dto.CreateCourseRequest{
        Titulo: "Curso de prueba",
    })

    assert.NoError(t, err)
    assert.Equal(t, "Curso de prueba", res.Titulo)
    assert.Equal(t, "borrador", res.Estado)
    repo.AssertExpectations(t)
}

func TestSubmitForReview_NotOwner(t *testing.T) {
    repo := new(repoMock)
    repo.On("GetByID", mock.Anything, "course-1").
        Return(domain.Course{ID: "course-1", CreadorID: "other-user", Estado: domain.EstadoBorrador}, nil)

    svc := service.New(repo, nil, nil)
    err := svc.SubmitForReview(context.Background(), "user-1", "course-1")
    assert.ErrorIs(t, err, service.ErrNotOwner)
}
```

### Tests con Tabla (Idiomatico Go)

```go
func TestEstadoTransitions(t *testing.T) {
    cases := []struct {
        name    string
        from    domain.Estado
        wantErr error
    }{
        {"desde_borrador", domain.EstadoBorrador, nil},
        {"desde_rechazado", domain.EstadoRechazado, nil},
        {"desde_aprobado", domain.EstadoAprobado, service.ErrInvalidTransition},
        {"desde_en_revision", domain.EstadoEnRevision, service.ErrInvalidTransition},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            // ... setup mocks con tc.from
            err := svc.SubmitForReview(ctx, "user-1", "course-1")
            assert.ErrorIs(t, err, tc.wantErr)
        })
    }
}
```

### Integration Test con testcontainers

```go
//go:build integration

package repository_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func setupPostgres(t *testing.T) *gorm.DB {
    ctx := context.Background()
    container, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
    )
    require.NoError(t, err)
    t.Cleanup(func() { _ = container.Terminate(ctx) })

    dsn, err := container.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    require.NoError(t, err)

    // Ejecutar migraciones reales (o golang-migrate.Up)
    // ...
    return db
}

func TestRepository_CreateAndGet(t *testing.T) {
    db := setupPostgres(t)
    repo := repository.New(db)

    err := repo.Create(context.Background(), &domain.Course{
        ID:        uuid.NewString(),
        Titulo:    "X",
        CreadorID: uuid.NewString(),
        Estado:    domain.EstadoBorrador,
    })
    require.NoError(t, err)
}
```

Ejecutar solo unit (rapido, default):

```bash
make backend-test
```

Ejecutar tambien integration (mas lento, requiere Docker):

```bash
make backend-test-integration
```

### Cobertura

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Objetivo (RNFT-05):** cobertura razonable en la capa de **services** de cada modulo. No perseguir 100%; priorizar tests significativos sobre cobertura cosmetica.

---

## 17. Logging Estructurado y Observabilidad

### Logger

Usar `log/slog` (estandar de la libreria estandar a partir de Go 1.21) o `zap` segun preferencia del equipo. El default recomendado es `slog` por simplicidad.

```go
// internal/platform/logger/logger.go
package logger

import (
    "log/slog"
    "os"
)

func New(level string, env string) *slog.Logger {
    var lvl slog.Level
    _ = lvl.UnmarshalText([]byte(level))

    var handler slog.Handler
    if env == "production" {
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
    } else {
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
    }
    return slog.New(handler)
}
```

### Middleware de Request Logging

```go
// internal/middleware/logger.go
func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        slog.Info("http",
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "status", c.Writer.Status(),
            "duration_ms", time.Since(start).Milliseconds(),
            "request_id", requestid.Get(c),
            "user_id", middleware.UserIDFrom(c),
            "ip", c.ClientIP(),
        )
    }
}
```

### Reglas

- **NUNCA loggear secrets**: JWT tokens, contrasenas (no hay, pero por las dudas), claves de storage.
- **NUNCA loggear PII innecesaria**: emails enteros pueden estar bien para auditoria, pero evitar nombres completos o datos sensibles si no aportan.
- **Cada log incluye `request_id`** para correlacionar todo el ciclo de vida de una request.
- **Errores con stack trace** solo en `slog.Error`, no en `slog.Info`.

### Health Endpoints (RT-27)

```go
// internal/platform/httpserver/health.go
func RegisterHealth(r *gin.Engine, db *gorm.DB, storage storage.Client) {
    r.GET("/api/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })
    r.GET("/api/health/ready", func(c *gin.Context) {
        sqlDB, err := db.DB()
        if err != nil || sqlDB.Ping() != nil {
            c.JSON(503, gin.H{"status": "down", "component": "db"})
            return
        }
        // (Opcional) ping storage
        c.JSON(200, gin.H{"status": "ready"})
    })
}
```

---

## 18. DevOps y Calidad de Codigo

### Conventional Commits

```
feat: agrega endpoint de aprobacion de cursos
fix(auth): corrige validacion de hosted_domain
chore(deps): actualiza gorm a v1.25.10
refactor(courses): extrae mapping a archivo separado
docs(backend): documenta flujo de upload de material
test(evaluations): cubre transiciones de estado
```

### Air (Hot Reload)

`.air.toml` (en la raiz de `backend/`):

```toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/api ./cmd/api"
bin = "./tmp/api"
include_ext = ["go", "tpl", "tmpl", "html"]
exclude_dir = ["tmp", "vendor", "docs", "migrations", "dist"]
delay = 200

[log]
time = true
```

Arranque: `air` (o `make backend-dev`).

### golangci-lint

`.golangci.yml`:

```yaml
run:
  timeout: 5m
  tests: true
  go: "1.23"

linters:
  enable:
    - govet
    - errcheck
    - staticcheck
    - gosec
    - gofmt
    - goimports
    - revive
    - unused
    - ineffassign
    - gocritic
    - sqlclosecheck
    - bodyclose
    - misspell

linters-settings:
  gocritic:
    enabled-tags: [diagnostic, performance, style]

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec  # tests pueden usar valores hardcoded
```

Correr: `make backend-lint`.

### Dockerfile

Ya documentado en `GUIA-MONOREPO.md` seccion 7.5 (multi-stage Go + distroless/static, imagen final ~20MB).

### Variables de Entorno Completas

Ver `GUIA-MONOREPO.md` seccion 6.3 (referencia unica). En este backend la carga es tipada via `caarlos0/env` (seccion 2 de esta guia).

---

## 19. Convenciones de Nomenclatura

### Resumen General

| Elemento | Convencion | Ejemplo |
|----------|-----------|---------|
| Paquete | `lowercase`, una palabra | `courses`, `auth`, `httperr` |
| Archivo | `lower_snake` | `course_repository.go`, `jwt_middleware.go` |
| Tipo exportado | `PascalCase` | `Service`, `CourseResponse`, `Repository` |
| Funcion exportada | `PascalCase` | `New`, `RegisterRoutes`, `UserIDFrom` |
| Tipo/funcion privada | `camelCase` | `service`, `toCourseResponse`, `repo` |
| Interface | nombre del rol + `er` cuando aplique, o sustantivo | `Reader`, `Service`, `Repository`, `Client` |
| Constante | `PascalCase` (exportadas) o `camelCase` (privadas) | `EstadoBorrador`, `defaultTTL` |
| Variable de error | prefijo `Err` | `ErrNotFound`, `ErrNotOwner` |
| Test | `TestXxx_Caso` | `TestSubmitForReview_NotOwner` |

### Tabla / Modelo / Endpoint

| Capa | Convencion | Ejemplo |
|------|-----------|---------|
| Nombre de tabla SQL | singular, lower-snake | `course`, `user_role`, `attempt` |
| Modelo Go | PascalCase singular | `Course`, `UserRole`, `Attempt` |
| Endpoint REST | plural en URL | `/courses`, `/users`, `/attempts` |
| Tag JSON | camelCase | `creadorId`, `publicadoEn`, `tamanoBytes` |

### Comentarios y godoc

- **Funciones exportadas**: comentario en formato godoc empezando con el nombre.

```go
// Catalog devuelve los cursos aprobados con paginacion y busqueda.
// Implementa RF-18 / RF-18b. Solo retorna cursos en estado "aprobado".
func (s *service) Catalog(ctx context.Context, q dto.CatalogQuery) (...) { ... }
```

- **Cuando NO comentar**: no explicar lo que el codigo ya dice. Comentar el **porque** y los **invariantes** no obvios.

### Imports

Ordenados en tres bloques separados por linea en blanco (lo aplica `goimports` automaticamente):

```go
import (
    "context"
    "errors"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/dto"
    "github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
)
```

---

## 20. Checklist para Crear un Nuevo Modulo

Cuando agregues un modulo nuevo (raro en el MVP, ya que los 7 estan cerrados), seguir este checklist:

- [ ] **Definir frontera**: ¿que tablas pertenecen a este modulo? Documentarlo en el comentario del paquete.
- [ ] **Crear estructura de directorios** segun seccion 4:
  ```
  internal/modules/<modulo>/
  ├── <modulo>.go    handler/  service/  repository/  domain/  dto/
  ```
- [ ] **Definir `Service` interface** en `<modulo>.go`. Esa es la API publica.
- [ ] **Implementar `service.New(...)`, `repository.New(...)`, `handler.Register(...)`** y re-exportarlas desde `<modulo>.go`.
- [ ] **Modelos GORM** en `domain/models.go` con `TableName()` explicito.
- [ ] **Crear migracion** con `make db-migrate-new name=add_<modulo>_tables`. Escribir `.up.sql` y `.down.sql` con el DDL.
- [ ] **DTOs** en `dto/request.go` y `dto/response.go` con tags `binding` apropiados.
- [ ] **Errores de dominio** como `ErrXxx` sentinels en `service/service.go`.
- [ ] **Registrar rutas** en `<modulo>.go` con `RegisterRoutes(rg *gin.RouterGroup, svc Service)`.
- [ ] **Anotar handlers con godoc swag** (`@Summary`, `@Tags`, `@Success`, `@Router`, `@Security BearerAuth`).
- [ ] **Aplicar middleware de roles** segun corresponda (`middleware.RequireRole(...)`).
- [ ] **Cablear en `cmd/api/main.go`** en la seccion 3 (modulos de dominio). Respetar el orden de dependencias.
- [ ] **Tests unitarios** del service con repo mock (testify).
- [ ] **Test de integracion** del repository con testcontainers (al menos los happy paths).
- [ ] **Regenerar Swagger**: `make swagger`.
- [ ] **Lint**: `make backend-lint`.
- [ ] **Commit en pequenos pasos** (conventional commits).

---

## Apendice: Mapeo Endpoint → Requerimiento Funcional

Trazabilidad rapida entre los endpoints principales y los RFs:

| Endpoint | RF | Notas |
|----------|----|----|
| `POST /api/auth/google` | RF-01 | Login con Google Workspace; emite access JWT + refresh token |
| `POST /api/auth/refresh` | RF-01 | Rota el refresh token y emite un nuevo access JWT |
| `POST /api/auth/logout` | RF-01 | Revoca el refresh token activo (idempotente) |
| `GET /api/users` | RF-04 | Gestion de usuarios (admin) |
| `PATCH /api/users/:id/roles` | RF-04 | Asignar/revocar roles |
| `POST /api/supervisions` | RF-04b | Asignar empleados a un supervisor |
| `GET /api/courses` | RF-18, RF-18b | Catalogo paginado con busqueda |
| `GET /api/courses/:id` | RF-19 | Detalle: videos + material |
| `POST /api/courses` | RF-05 | Crear curso (creador) |
| `PATCH /api/courses/:id` | RF-09 | Editar mientras esta en borrador |
| `POST /api/courses/:id/submit` | RF-10 | Enviar a revision |
| `POST /api/courses/:courseId/sections` | RF-06 | Agregar seccion a un curso |
| `POST /api/sections/:sectionId/videos` | RF-06 | Agregar video a una seccion (YouTube/Vimeo) |
| `POST /api/courses/:id/materials/presign` | RF-07, RT-23 | URL prefirmada para upload |
| `POST /api/courses/:id/materials` | RF-07 | Confirma material adjunto |
| `POST /api/courses/:id/evaluation` | RF-08 | Crea evaluacion |
| `POST /api/evaluations/:id/questions` | RF-11, RF-12 | Pregunta opcion multiple / V-F |
| `POST /api/courses/:id/enroll` | RF-18 | Inscripcion del alumno |
| `POST /api/evaluations/:id/attempts` | RF-13 | Iniciar intento |
| `POST /api/attempts/:id/submit` | RF-13, RF-14 | Enviar respuestas, calcular nota |
| `POST /api/courses/:id/approve` | RF-15, RF-16 | Aprobar (admin) |
| `POST /api/courses/:id/reject` | RF-15, RF-17, RF-17b | Rechazar con comentario |
| `GET /api/certificates/me` | RF-21 | Mis certificados |
| `GET /api/certificates/:id/download` | RF-21 | Descarga PDF (URL prefirmada) |
| `GET /api/badges/me` | RF-22 | Mis insignias |
| `GET /api/badges/ranking` | RF-22 | Ranking |
| `GET /api/reports/global` | RF-23, RF-25 | Reportes admin |
| `GET /api/reports/team` | RF-24 | Avance del equipo (supervisor) |

Esta tabla debe mantenerse sincronizada con el spec OpenAPI generado por swag.
