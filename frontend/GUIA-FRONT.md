# Guia de Proyecto Base - Angular (SkillMaker Frontend)

Documentacion de la arquitectura, estructura y patrones del **frontend de SkillMaker** — plataforma interna de formacion en video (tipo Udemy / LMS corporativo). Sirve tambien como referencia para replicar esta base en futuros proyectos Angular.

> **Para entender el contexto del proyecto:**
> - Estructura del monorepo, Docker, deployment: [`../GUIA-MONOREPO.md`](../GUIA-MONOREPO.md)
> - Requerimientos funcionales y tecnicos (fuente de verdad): [`../bases/documentacion/`](../bases/documentacion/)
>   - `Requerimientos_Plataforma_Formacion.docx` — RFs y RNFs
>   - `Requerimientos_Tecnicos_Plataforma_Formacion.docx` — RTs y arquitectura C4
>   - `Modelo_Datos_Plataforma_Formacion.docx` — diccionario de datos y DDL
> - Patrones del backend Go + Gin: [`../backend/GUIA-BACK.md`](../backend/GUIA-BACK.md)

---

## 1. Stack Tecnologico

| Tecnologia | Version | Proposito |
|---|---|---|
| Angular | 21 | Framework principal (Standalone, Zoneless) |
| PrimeNG | 21 | Componentes UI |
| PrimeIcons | 7 | Iconos |
| @primeuix/themes | 2 | Motor de temas (Aura preset) |
| Tailwind CSS | 4 | Utilidades CSS |
| tailwindcss-primeui | 0.6 | Integracion Tailwind + PrimeNG |
| TypeScript | 5.9 | Lenguaje |
| Vitest | 4 | Testing unitario |
| RxJS | 7.8 | Programacion reactiva |
| SASS | - | Estilos globales y overrides |
| `@react-oauth/google` Google Identity Services (GIS) | - | SDK oficial para iniciar el flujo OAuth con Google (RT-12) |

**Decisiones de stack vinculadas a las bases:**
- **SPA Angular servida desde CDN** (RT-01): la app se compila estatica y se sirve via nginx/CDN en produccion.
- **API REST consumida via contrato OpenAPI** (RT-02): los tipos TypeScript se generan idealmente desde `backend/docs/swagger.json` con `openapi-typescript`.
- **Login con Google Workspace, sin contrasenas propias** (RT-03, RT-04, RNF-03): el frontend obtiene un ID token de Google y lo intercambia por un JWT propio en el backend.
- **JWT propio para sesiones** (RT-04, RT-14, RT-15): tras el intercambio, el frontend almacena el JWT y lo envia en cada peticion.
- **UI adapta vistas y acciones segun rol** (RT-05, RNF-04): guards de ruta + directivas estructurales para mostrar/ocultar elementos.

---

## 2. Estructura de Directorios

```
src/
├── app/
│   ├── app.ts                          # Componente raiz
│   ├── app.html                        # Template raiz (router-outlet + toast + confirm)
│   ├── app.config.ts                   # Configuracion de providers, tema, interceptors
│   ├── app.routes.ts                   # Rutas principales (auth + platform)
│   │
│   ├── api/                            # Tipos generados desde OpenAPI (make types)
│   │   ├── types.ts                    # Auto-generado — NO editar a mano
│   │   └── types.contract.ts           # Asercion de compilacion (contrato)
│   │
│   ├── core/                           # Singleton: guards, interceptors, services
│   │   ├── constants/
│   │   │   └── texts.ts                # Constantes de texto
│   │   ├── guards/
│   │   │   ├── auth.guard.ts           # Protege rutas autenticadas
│   │   │   ├── guest.guard.ts          # Protege rutas publicas (callback de OAuth)
│   │   │   └── role.guard.ts           # Protege rutas por rol (alumno/creador/supervisor/administrador)
│   │   ├── interceptors/
│   │   │   ├── auth-token.interceptor.ts    # Inyecta Bearer token
│   │   │   └── auth-refresh.interceptor.ts  # Maneja 401 y refresca token
│   │   └── services/
│   │       ├── http-promise-builder.service.ts  # Builder HTTP (Promise-based)
│   │       ├── ui-dialog.service.ts             # Dialogos y toasts
│   │       ├── authService/                     # Login Google + JWT propio
│   │       │   ├── auth.service.ts
│   │       │   ├── auth.req.dto.ts
│   │       │   └── auth.res.dto.ts
│   │       ├── common/
│   │       │   ├── error-response.dto.ts
│   │       │   └── role-check.service.ts        # Verificacion de roles (4 roles fijos del dominio)
│   │       └── [entidad]Service/                # Un folder por entidad de dominio
│   │           ├── [entidad].service.ts
│   │           ├── [entidad].req.dto.ts
│   │           └── [entidad].res.dto.ts
│   │
│   ├── pages/                          # Feature modules (lazy loaded)
│   │   ├── auth/                       # Modulo de autenticacion (solo Google OAuth)
│   │   │   ├── auth.routes.ts          # Rutas hijas de /auth
│   │   │   ├── layout/                 # Layout de auth (two-column)
│   │   │   ├── login/                  # Boton "Continuar con Google"
│   │   │   └── callback/               # Callback OAuth (recibe credential y autentica)
│   │   │
│   │   └── platform/                   # Modulo principal de la app (LMS)
│   │       ├── platform.routes.ts      # Rutas hijas de /platform
│   │       ├── layout/                 # Layout con sidebar + header
│   │       │
│   │       ├── catalog/                # RF-18: Catalogo de cursos aprobados
│   │       ├── course-detail/          # RF-19: Detalle del curso (videos, material)
│   │       ├── course-player/          # Reproductor + checkpoint de avance
│   │       ├── evaluation/             # RF-11 a RF-13: Rendir evaluacion
│   │       ├── my-courses/             # Cursos inscritos + progreso
│   │       ├── certificates/           # RF-21: Mis certificados
│   │       ├── badges/                 # RF-22: Insignias y ranking
│   │       ├── profile/                # Datos del usuario actual (read-only en este MVP)
│   │       │
│   │       ├── creator/                # Vistas exclusivas del rol "creador"
│   │       │   ├── my-content/         # Cursos creados por el usuario
│   │       │   ├── course-edit/        # RF-05 a RF-09: crear/editar curso
│   │       │   ├── evaluation-edit/    # Crear/editar evaluacion del curso
│   │       │   └── submit-review/      # RF-10: enviar curso a revision
│   │       │
│   │       ├── admin/                  # Vistas exclusivas del rol "administrador"
│   │       │   ├── approvals/          # RF-15: bandeja de cursos pendientes de aprobacion
│   │       │   ├── user-management/    # RF-04: gestion de usuarios y roles
│   │       │   ├── supervision-setup/  # RF-04b: asignar empleados a un supervisor
│   │       │   └── global-reports/     # RF-23, RF-25: reportes globales
│   │       │
│   │       └── supervisor/             # Vistas exclusivas del rol "supervisor"
│   │           └── team-progress/      # RF-24: avance y puntajes de su equipo
│   │
│   ├── shared/                         # Componentes y directivas reutilizables
│   │   ├── components/
│   │   │   ├── loading-wall/           # Overlay de carga fullscreen
│   │   │   ├── video-embed/            # Embed seguro de YouTube/Vimeo (RT-24)
│   │   │   ├── material-uploader/      # Upload a object storage via URL prefirmada
│   │   │   └── course-card/            # Tarjeta de curso reutilizable
│   │   └── directives/
│   │       └── has-role.directive.ts        # Directiva estructural por rol
│   │
│   └── sass/
│       └── _colors.sass                # Variables de color (override del tema)
│
├── environments/
│   ├── environment.ts                  # Desarrollo
│   └── environment.prod.ts             # Produccion
│
├── index.html                          # HTML principal (carga Google Identity Services)
├── main.ts                             # Bootstrap
├── styles.sass                         # Estilos globales y overrides PrimeNG
└── tailwind.css                        # Config Tailwind + plugin PrimeUI
```

### Convenciones de Nombres

- **Servicios de dominio:** `[entidad]Service/` con 3 archivos: `service.ts`, `req.dto.ts`, `res.dto.ts`
- **Componentes de pagina:** `[feature]/[feature].component.ts/html/sass`
- **Sub-componentes:** dentro del folder de su pagina padre (ej: `course-edit/section-form/`)
- **Guards e interceptors:** funcionales (no clases), exportados como `const`
- **Carpetas por rol:** las vistas exclusivas de un rol (`creator/`, `admin/`, `supervisor/`) se agrupan para que el sidebar y los guards se cableen de forma consistente.

### Path Aliases (tsconfig.json)

```json
{
  "paths": {
    "@api/*":    ["src/app/api/*"],
    "@core/*":   ["src/app/core/*"],
    "@pages/*":  ["src/app/pages/*"],
    "@shared/*": ["src/app/shared/*"],
    "@env/*":    ["src/environments/*"]
  }
}
```

> **Nota:** los paths no usan `./` como prefijo — coinciden con la configuracion real de `tsconfig.json`.

---

## 3. Configuracion Base (app.config.ts)

```typescript
import {
  ApplicationConfig,
  provideBrowserGlobalErrorListeners,
  provideZonelessChangeDetection,
} from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { ConfirmationService, MessageService } from 'primeng/api';
import { providePrimeNG } from 'primeng/config';
import { definePreset } from '@primeuix/themes';
import Aura from '@primeuix/themes/aura';

import { routes } from './app.routes';
import { authTokenInterceptor } from '@core/interceptors/auth-token.interceptor';
import { authRefreshInterceptor } from '@core/interceptors/auth-refresh.interceptor';

// Tema personalizado — el color primario se puede ajustar segun la identidad del producto.
// SkillMaker usa azul corporativo como ejemplo; cambiar la paleta para otro proyecto.
const SkillMakerPreset = definePreset(Aura, {
  semantic: {
    primary: {
      50: '{blue.50}', 100: '{blue.100}', 200: '{blue.200}',
      300: '{blue.300}', 400: '{blue.400}', 500: '{blue.500}',
      600: '{blue.600}', 700: '{blue.700}', 800: '{blue.800}',
      900: '{blue.900}', 950: '{blue.950}',
    },
  },
});

// Locale espanol para calendarios y componentes de fecha
const esLocale = {
  firstDayOfWeek: 1,
  dayNames: ['Domingo', 'Lunes', 'Martes', 'Miercoles', 'Jueves', 'Viernes', 'Sabado'],
  dayNamesShort: ['Dom', 'Lun', 'Mar', 'Mie', 'Jue', 'Vie', 'Sab'],
  dayNamesMin: ['D', 'L', 'M', 'M', 'J', 'V', 'S'],
  monthNames: [
    'Enero', 'Febrero', 'Marzo', 'Abril', 'Mayo', 'Junio',
    'Julio', 'Agosto', 'Septiembre', 'Octubre', 'Noviembre', 'Diciembre',
  ],
  monthNamesShort: [
    'Ene', 'Feb', 'Mar', 'Abr', 'May', 'Jun',
    'Jul', 'Ago', 'Sep', 'Oct', 'Nov', 'Dic',
  ],
  today: 'Hoy',
  clear: 'Limpiar',
  dateFormat: 'dd/mm/yy',
  weekHeader: 'Sem',
};

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideZonelessChangeDetection(),          // Sin Zone.js
    provideRouter(routes),
    provideHttpClient(
      withInterceptors([
        authTokenInterceptor,                  // Inyecta Bearer token
        authRefreshInterceptor,                // Maneja 401 y refresca
      ])
    ),
    provideAnimations(),
    provideAnimationsAsync(),
    ConfirmationService,                       // PrimeNG: dialogos de confirmacion
    MessageService,                            // PrimeNG: toasts
    providePrimeNG({
      theme: {
        preset: SkillMakerPreset,
        options: { darkModeSelector: 'none' }, // Sin dark mode en MVP
      },
      translation: esLocale,
    }),
  ],
};
```

**Puntos clave:**
- `provideZonelessChangeDetection()` elimina Zone.js, usa Signals para change detection.
- Interceptors se registran como funciones via `withInterceptors([])`.
- PrimeNG se configura con tema custom (Aura base + colores propios) y locale espanol.
- `ConfirmationService` y `MessageService` se proveen globalmente para dialogos/toasts.

**Carga de Google Identity Services en `index.html`:**

```html
<head>
  <!-- ... -->
  <script src="https://accounts.google.com/gsi/client" async defer></script>
</head>
```

Este script expone `window.google.accounts.id` para inicializar el flujo de login (ver seccion 5).

---

## 4. Sistema de Rutas

### Rutas Principales (app.routes.ts)

```typescript
export const routes: Routes = [
  {
    path: 'auth',
    canActivate: [guestGuard],       // Solo usuarios NO autenticados
    loadChildren: () =>
      import('./pages/auth/auth.routes').then((m) => m.AUTH_ROUTES),
  },
  {
    path: 'platform',
    canActivate: [authGuard],        // Solo usuarios autenticados
    loadChildren: () =>
      import('./pages/platform/platform.routes').then((m) => m.PLATFORM_ROUTES),
  },
  {
    path: '',
    redirectTo: '/auth',
    pathMatch: 'full',
  },
];
```

### Patron de Rutas Hijas con Layout

Cada modulo usa un **LayoutComponent como padre** que contiene el `<router-outlet>` para sus hijos. Esto permite tener layouts distintos para auth y platform.

**Auth Routes (solo Google OAuth, sin register/recover):**
```typescript
export const AUTH_ROUTES: Routes = [
  {
    path: '',
    component: LayoutComponent,        // Layout de auth (two-column)
    children: [
      {
        path: 'login',
        loadComponent: () =>
          import('./login/login.component').then((m) => m.LoginComponent),
      },
      {
        path: 'callback',              // Procesa el credential de Google e intercambia por JWT propio
        loadComponent: () =>
          import('./callback/callback.component').then((m) => m.CallbackComponent),
      },
      { path: '', redirectTo: 'login', pathMatch: 'full' },
    ],
  },
];
```

**Importante:** no hay rutas `register`, `recover-password`, ni `new-password`. La autenticacion es 100% delegada a Google Workspace (RT-12, RT-13, RNF-03).

**Platform Routes (con role guards):**
```typescript
export const PLATFORM_ROUTES: Routes = [
  {
    path: '',
    component: LayoutComponent,        // Layout de plataforma (sidebar + header)
    children: [
      // ─── Acceso general (alumno: todos los autenticados) ──────────
      {
        path: 'catalog',
        loadComponent: () =>
          import('./catalog/catalog.component').then((m) => m.CatalogComponent),
      },
      {
        path: 'courses/:courseId',
        loadComponent: () =>
          import('./course-detail/course-detail.component').then((m) => m.CourseDetailComponent),
      },
      {
        path: 'courses/:courseId/player',
        loadComponent: () =>
          import('./course-player/course-player.component').then((m) => m.CoursePlayerComponent),
      },
      {
        path: 'courses/:courseId/evaluation',
        loadComponent: () =>
          import('./evaluation/evaluation.component').then((m) => m.EvaluationComponent),
      },
      {
        path: 'my-courses',
        loadComponent: () =>
          import('./my-courses/my-courses.component').then((m) => m.MyCoursesComponent),
      },
      {
        path: 'certificates',
        loadComponent: () =>
          import('./certificates/certificates.component').then((m) => m.CertificatesComponent),
      },
      {
        path: 'badges',
        loadComponent: () =>
          import('./badges/badges.component').then((m) => m.BadgesComponent),
      },
      {
        path: 'profile',
        loadComponent: () =>
          import('./profile/profile.component').then((m) => m.ProfileComponent),
      },

      // ─── Rol creador ─────────────────────────────────────────────
      {
        path: 'creator',
        canActivate: [roleGuard],
        data: { roles: ['creador'] },
        loadChildren: () =>
          import('./creator/creator.routes').then((m) => m.CREATOR_ROUTES),
      },

      // ─── Rol supervisor ──────────────────────────────────────────
      {
        path: 'supervisor',
        canActivate: [roleGuard],
        data: { roles: ['supervisor'] },
        loadChildren: () =>
          import('./supervisor/supervisor.routes').then((m) => m.SUPERVISOR_ROUTES),
      },

      // ─── Rol administrador ───────────────────────────────────────
      {
        path: 'admin',
        canActivate: [roleGuard],
        data: { roles: ['administrador'] },
        loadChildren: () =>
          import('./admin/admin.routes').then((m) => m.ADMIN_ROUTES),
      },

      { path: '', redirectTo: 'catalog', pathMatch: 'full' },
    ],
  },
];
```

**Patron clave:**
- Cada componente se carga con `loadComponent` (lazy loading individual). Para sub-routers por rol se usa `loadChildren`.
- Las restricciones se definen en `data: { roles: [...] }` y se validan con `roleGuard`.
- Un usuario con mas de un rol (ej: alumno + creador) puede acceder a todas las secciones correspondientes (RF-03).

---

## 5. Sistema de Autenticacion (Google OAuth 2.0 + JWT propio)

### Vision general del flujo

```
1. Usuario hace clic en "Continuar con Google" (LoginComponent)
   └─> google.accounts.id.prompt()  ← abre el popup/one-tap de Google

2. Google devuelve un ID token (JWT firmado por Google) al callback configurado
   └─> AuthService.loginWithGoogle(idToken)

3. AuthService envia POST /api/auth/google al backend con el ID token
   └─> Backend valida el ID token contra Google (idtoken.NewValidator)
   └─> Backend verifica que el email pertenece al dominio corporativo (RT-13)
   └─> Backend crea/actualiza el user en su BD (google_sub, email, nombre)
   └─> Backend emite un JWT propio con sub, email, nombre, roles, exp
   └─> Backend retorna { access_token, refresh_token?, user, roles }

4. AuthService guarda el JWT en localStorage y carga roles en Signals
5. authTokenInterceptor inyecta `Authorization: Bearer <jwt>` en cada peticion
6. Si el JWT expira: authRefreshInterceptor refresca via /api/auth/refresh
```

**Por que no usar email/password:** RNF-03 dice explicitamente que **la autenticacion se delega en Google Workspace, por lo que el sistema no almacena contrasenas**. Tampoco hay registro manual ni recuperacion de contrasena.

### AuthService (core/services/authService/auth.service.ts)

Servicio central que gestiona todo el ciclo de vida de autenticacion usando **Angular Signals** para estado reactivo.

```typescript
@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly baseUrl = `${environment.apiBaseUrl}/auth`;

  // ─── Estado reactivo (Signals) ─────────────────────────────
  public user: WritableSignal<User | null> = signal<User | null>(null);
  public sessionExpired: WritableSignal<boolean> = signal(false);
  public userRoles: WritableSignal<UserRole[]> = signal<UserRole[]>([]);

  // ─── Proteccion contra race conditions en refresh ──────────
  private isRefreshing = false;
  private refreshPromise: Promise<RefreshTokenResponse> | null = null;
  private expirationTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(
    private readonly httpBuilder: HttpPromiseBuilderService,
    private readonly router: Router,
  ) {
    this.restoreSession();  // Restaura sesion al iniciar la app
  }
```

**Tipo `UserRole`** (alineado con el modelo de datos, seccion 2.1 del doc de datos):

```typescript
export type UserRole = 'alumno' | 'creador' | 'supervisor' | 'administrador';
```

#### Inicio del login con Google (LoginComponent)

```typescript
declare const google: any;  // Carga global de Google Identity Services (index.html)

@Component({
  selector: 'app-login',
  standalone: true,
  template: `
    <button pButton class="p-button-primary" (click)="signInWithGoogle()">
      <i class="pi pi-google mr-2"></i>
      Continuar con Google
    </button>
  `,
})
export class LoginComponent implements OnInit {
  private readonly authService = inject(AuthService);
  private readonly router = inject(Router);
  private readonly dialog = inject(UiDialogService);

  ngOnInit(): void {
    google.accounts.id.initialize({
      client_id: environment.googleClientId,
      hd: environment.googleHostedDomain,      // Restringe al dominio corporativo (RT-13)
      callback: (response: { credential: string }) => this.handleCredential(response.credential),
      auto_select: false,
      cancel_on_tap_outside: true,
    });
  }

  signInWithGoogle(): void {
    google.accounts.id.prompt();  // Abre el flujo One Tap / popup
  }

  private async handleCredential(idToken: string): Promise<void> {
    try {
      await this.authService.loginWithGoogle(idToken);
      await this.router.navigate(['/platform']);
    } catch (err) {
      this.dialog.showError('No se pudo iniciar sesion', extractMessage(err));
    }
  }
}
```

**Notas:**
- `client_id`: el OAuth Client ID publico (no es un secret).
- `hd`: hosted domain — Google filtra cuentas que no pertenezcan a ese dominio. El backend igual debe revalidar (defensa en profundidad, RT-13).
- `auto_select: false`: evita que Google auto-loguee silenciosamente sin confirmacion del usuario.

#### Intercambio del ID token por JWT propio

```typescript
  async loginWithGoogle(idToken: string): Promise<LoginResponse> {
    const response = await this.httpBuilder
      .request<LoginResponse>()
      .post()
      .url(`${this.baseUrl}/google`)
      .body({ idToken })
      .send();

    this.setTokens(response.access_token, response.refresh_token);
    this.loadClaimsFromToken(response.access_token);

    const decoded = this.decodeToken(response.access_token);
    if (decoded) {
      const user: User = {
        id: decoded.sub,
        email: decoded.email,
        nombre: decoded.nombre ?? '',
      };
      this.user.set(user);
      localStorage.setItem('auth_user', JSON.stringify(user));
    }

    this.sessionExpired.set(false);
    this.startExpirationTimer();
    return response;
  }
```

#### Token Management

Tokens almacenados en `localStorage` con claves `auth_token` y `auth_refresh_token`:

```typescript
  getToken(): string | null {
    return localStorage.getItem('auth_token');
  }

  getRefreshToken(): string | null {
    return localStorage.getItem('auth_refresh_token');
  }

  private setTokens(accessToken: string, refreshToken: string | undefined): void {
    localStorage.setItem('auth_token', accessToken);
    if (refreshToken) localStorage.setItem('auth_refresh_token', refreshToken);
  }

  private clearTokens(): void {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_refresh_token');
    localStorage.removeItem('auth_user');
  }
```

> **Contrato con el backend:** el backend implementa `POST /api/auth/refresh` con **rotacion obligatoria** del refresh token y **deteccion de replay** (si un token rotado se reutiliza, se revocan todas las sesiones del usuario). Ver `backend/GUIA-BACK.md` seccion 9 → "Refresh Tokens" para el contrato completo (request body `{ refreshToken }`, response identica al login). El frontend debe respetar la rotacion: **persistir el NUEVO `refresh_token` que viene en la response** (no reutilizar el viejo).

#### JWT Decoding (sin librerias externas)

```typescript
  private decodeToken(token: string | null | undefined): JwtPayload | null {
    if (!token || typeof token !== 'string') return null;
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    try {
      const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
      return JSON.parse(payload);
    } catch {
      return null;
    }
  }

  isTokenExpired(): boolean {
    const decoded = this.decodeToken(this.getToken());
    if (!decoded?.exp) return true;
    return Date.now() >= decoded.exp * 1000;
  }
```

#### Carga de roles desde el JWT

```typescript
  private loadClaimsFromToken(token: string): void {
    const decoded = this.decodeToken(token);
    if (!decoded) return;

    // RT-15: el JWT incluye id, email y roles del usuario
    this.userRoles.set((decoded.roles ?? []) as UserRole[]);
  }
```

#### Token Refresh con Singleton Pattern (anti race-condition)

Mismo patron que la guia original: una sola promise compartida para evitar refresh duplicados cuando varias peticiones reciben 401 simultaneamente.

```typescript
  async refresh(): Promise<RefreshTokenResponse> {
    if (this.isRefreshing && this.refreshPromise) {
      return this.refreshPromise;
    }
    this.isRefreshing = true;
    this.refreshPromise = this._doRefresh();
    try {
      return await this.refreshPromise;
    } finally {
      this.isRefreshing = false;
      this.refreshPromise = null;
    }
  }

  private async _doRefresh(): Promise<RefreshTokenResponse> {
    const refreshToken = this.getRefreshToken();
    if (!refreshToken) throw new Error('No refresh token available');

    const response = await this.httpBuilder
      .request<RefreshTokenResponse>()
      .post()
      .url(`${this.baseUrl}/refresh`)
      .body({ refreshToken })
      .send();

    this.setTokens(response.access_token, response.refresh_token ?? refreshToken);
    this.loadClaimsFromToken(response.access_token);
    this.startExpirationTimer();
    return response;
  }
```

#### Timer de Expiracion Automatico

Refresca el token 5 minutos antes de que expire. Si falla, activa la senal `sessionExpired`:

```typescript
  private startExpirationTimer(): void {
    this.stopExpirationTimer();
    const decoded = this.decodeToken(this.getToken());
    if (!decoded?.exp) return;

    const expiresIn = decoded.exp * 1000 - Date.now();
    const refreshBefore = 5 * 60 * 1000; // 5 minutos antes
    const timeout = Math.max(expiresIn - refreshBefore, 0);

    this.expirationTimer = setTimeout(async () => {
      try {
        await this.refreshWithRetry();
      } catch {
        this.sessionExpired.set(true);
      }
    }, timeout);
  }
```

#### Restauracion de Sesion

Al iniciar la app, intenta restaurar la sesion desde localStorage:

```typescript
  private restoreSession(): void {
    const token = this.getToken();
    const userJson = localStorage.getItem('auth_user');
    if (!token || !userJson) return;

    const user = JSON.parse(userJson) as User;
    this.user.set(user);
    this.loadClaimsFromToken(token);

    if (!this.isTokenExpired()) {
      this.startExpirationTimer();
    } else if (this.getRefreshToken()) {
      this.refreshWithRetry().catch(() => this.sessionExpired.set(true));
    } else {
      this.sessionExpired.set(true);
    }
  }
```

#### Logout

```typescript
  async logout(): Promise<void> {
    try {
      await this.httpBuilder
        .request<void>()
        .post()
        .url(`${this.baseUrl}/logout`)
        .silent()
        .send();
    } finally {
      // Indica a Google que la sesion termino (deshabilita auto-login silencioso)
      try { google.accounts.id.disableAutoSelect(); } catch { /* noop */ }
      this.clearTokens();
      this.user.set(null);
      this.userRoles.set([]);
      this.stopExpirationTimer();
      await this.router.navigate(['/auth']);
    }
  }
```

### Flujo Completo de Autenticacion

```
App Inicia
  └─> restoreSession()
       ├─ Token valido → startExpirationTimer()
       ├─ Token expirado + refresh valido → refreshWithRetry()
       └─ Ambos expirados → sessionExpired = true

Login
  └─> google.accounts.id.prompt()
       └─> credential (ID token de Google)
       └─> POST /api/auth/google  (backend valida domain + emite JWT propio)
            └─> setTokens() → loadClaims() → user.set() → startExpirationTimer()

Peticion HTTP
  └─> authTokenInterceptor: agrega Bearer token
       └─> Si 401 → authRefreshInterceptor: refresh() → reintenta peticion

Timer Expiracion (5 min antes)
  └─> refreshWithRetry(3 intentos, backoff exponencial)
       ├─ Exito → nuevos tokens → reinicia timer
       └─ Fallo → sessionExpired = true → modal de re-login

Logout
  └─> POST /api/auth/logout
       └─> google.accounts.id.disableAutoSelect()
       └─> clearTokens() → user = null → navegar a /auth
```

---

## 6. Guards Funcionales

### Auth Guard

Protege rutas que requieren autenticacion:

```typescript
export const authGuard: CanActivateFn = () => {
  const authService = inject(AuthService);
  const router = inject(Router);
  const token = authService.getToken();

  if (token && !authService.isTokenExpired()) return true;

  if (token && authService.isTokenExpired()) {
    const refreshToken = authService.getRefreshToken();
    if (refreshToken) return true; // El interceptor/timer refrescara
    authService.sessionExpired.set(true);
    return true;
  }

  void router.navigate(['/auth']);
  return false;
};
```

### Guest Guard

Impide que usuarios autenticados accedan a la pagina de login:

```typescript
export const guestGuard: CanActivateFn = () => {
  const authService = inject(AuthService);
  const router = inject(Router);
  if (!authService.getToken()) return true;
  router.navigate(['/platform']);
  return false;
};
```

### Role Guard

Valida que el usuario tenga al menos uno de los roles requeridos en `route.data`. Para SkillMaker es la forma primaria de control de acceso (RNF-04, RT-05).

```typescript
export const roleGuard: CanActivateFn = (route) => {
  const roleService = inject(RoleCheckService);
  const router = inject(Router);
  const requiredRoles = route.data?.['roles'] as UserRole[] | undefined;

  if (!requiredRoles || requiredRoles.length === 0) return true;
  if (roleService.hasAnyRole(requiredRoles)) return true;

  console.warn(`Acceso denegado: rol requerido [${requiredRoles.join(', ')}]`);
  router.navigate(['/platform']);
  return false;
};
```

---

## 7. Interceptors HTTP

### Auth Token Interceptor

Agrega automaticamente el header `Authorization: Bearer {token}` a todas las peticiones:

```typescript
export const authTokenInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);
  const token = authService.getToken();
  if (!token) return next(req);

  const authRequest = req.clone({
    setHeaders: { Authorization: `Bearer ${token}` },
  });
  return next(authRequest);
};
```

### Auth Refresh Interceptor

Captura errores 401, refresca el token y reintenta la peticion original:

```typescript
export const authRefreshInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);

  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      if (error.status !== 401) return throwError(() => error);
      if (req.url.includes('/auth/refresh')) return throwError(() => error);
      if (!authService.getRefreshToken()) return throwError(() => error);

      return new Observable<HttpEvent<unknown>>((observer) => {
        authService.refresh()
          .then((refreshResponse) => {
            const retryReq = req.clone({
              setHeaders: { Authorization: `Bearer ${refreshResponse.access_token}` },
            });
            next(retryReq).subscribe({
              next: (response) => observer.next(response),
              error: (err) => observer.error(err),
              complete: () => observer.complete(),
            });
          })
          .catch((refreshError) => observer.error(refreshError));
      });
    }),
  );
};
```

---

## 8. Sistema de Roles

El control de acceso del frontend se basa **exclusivamente en roles** (RF-02, RNF-04). No hay sistema de permisos granulares: el modelo de datos define solo la tabla `role`, sin tabla `permission`. Toda decision de autorizacion se toma evaluando si el usuario tiene uno de los 4 roles fijos.

### RoleCheckService

Los roles definidos por el dominio son fijos: `alumno`, `creador`, `supervisor`, `administrador` (seccion 2 del doc funcional + tabla `role` del modelo de datos). Un usuario puede tener varios simultaneamente (RF-03).

```typescript
@Injectable({ providedIn: 'root' })
export class RoleCheckService {
  constructor(private readonly authService: AuthService) {}

  get roles(): Signal<UserRole[]> {
    return this.authService.userRoles;
  }

  hasRole(role: UserRole): boolean {
    return this.authService.userRoles().includes(role);
  }

  hasAnyRole(roles: UserRole[]): boolean {
    if (!roles?.length) return true;
    const userRoles = this.authService.userRoles();
    return roles.some((r) => userRoles.includes(r));
  }

  isAdmin = (): boolean => this.hasRole('administrador');
  isCreator = (): boolean => this.hasRole('creador');
  isSupervisor = (): boolean => this.hasRole('supervisor');

  // Signal reactivo para usar en templates
  hasRole$(role: UserRole): Signal<boolean> {
    return computed(() => this.authService.userRoles().includes(role));
  }
}
```

### HasRole Directive

Directiva estructural que muestra/oculta elementos segun rol. Reacciona automaticamente via `effect()`:

```typescript
@Directive({ selector: '[hasRole]', standalone: true })
export class HasRoleDirective {
  private readonly roleService = inject(RoleCheckService);
  private readonly templateRef = inject(TemplateRef<unknown>);
  private readonly viewContainer = inject(ViewContainerRef);
  private readonly destroyRef = inject(DestroyRef);
  private roles: UserRole[] = [];
  private isRendered = false;

  @Input()
  set hasRole(value: UserRole | UserRole[]) {
    this.roles = Array.isArray(value) ? value : [value];
  }

  constructor() {
    const effectRef = effect(() => {
      this.roleService.roles();  // dependencia reactiva
      this.updateView();
    });
    this.destroyRef.onDestroy(() => effectRef.destroy());
  }

  private updateView(): void {
    const hasAccess = this.roleService.hasAnyRole(this.roles);
    if (hasAccess && !this.isRendered) {
      this.viewContainer.createEmbeddedView(this.templateRef);
      this.isRendered = true;
    } else if (!hasAccess && this.isRendered) {
      this.viewContainer.clear();
      this.isRendered = false;
    }
  }
}
```

**Uso en templates:**

```html
<!-- Un solo rol -->
<button *hasRole="'creador'">Crear curso</button>

<!-- Multiples roles (cualquiera basta) -->
<a routerLink="/platform/admin/approvals" *hasRole="['administrador']">Bandeja de aprobacion</a>
<a routerLink="/platform/supervisor/team-progress" *hasRole="['supervisor']">Avance del equipo</a>
```

### Reglas Mas Finas (Ownership)

El `RoleCheckService` y la directiva `*hasRole` solo cubren control por rol. Cuando una decision depende del rol **y** de la propiedad del recurso (ej: "el creador solo puede editar SUS cursos"), la validacion definitiva vive en el backend (service de Go). El frontend solo replica la heuristica via comparaciones simples:

```typescript
const isMine = course.creadorId === this.authService.user()?.id;
const canEdit = isMine && this.roleService.hasRole('creador');
```

---

## 9. Layouts

### Auth Layout (pages/auth/layout/)

Layout de dos columnas para la pagina de login:

- **Columna izquierda:** Card centrada con el boton "Continuar con Google" como CTA principal (`<router-outlet>` para login/callback).
- **Columna derecha:** Panel de bienvenida con branding institucional (oculto en mobile con `hidden lg:flex`).
- **Fondo:** color primario corporativo en el panel derecho.
- **Responsive:** Mobile-first, el panel derecho desaparece en pantallas pequenas.

### Platform Layout (pages/platform/layout/)

Layout principal de la aplicacion con tres zonas:

- **Sidebar (fijo, izquierda):**
  - Colapsable: 16rem expandido, 4rem colapsado.
  - Logo + titulo "SkillMaker".
  - Menu de navegacion organizado por rol:
    - **Aprendizaje (todos):** Catalogo, Mis cursos, Certificados, Insignias
    - **Creador (`*hasRole="'creador'"`):** Mi contenido, Crear curso
    - **Supervisor (`*hasRole="'supervisor'"`):** Avance del equipo
    - **Administracion (`*hasRole="'administrador'"`):** Aprobaciones, Usuarios y roles, Supervision, Reportes
  - Footer con "Soporte" y "Cerrar sesion".
  - Tooltips visibles cuando el sidebar esta colapsado.
  - Usa PrimeNG Ripple y Tooltip.

- **Header (sticky, top):**
  - Titulo de la app + icono a la izquierda.
  - Seccion de usuario a la derecha (avatar de Google, nombre, rol primario).
  - Popover menu con enlace a perfil y logout.

- **Area de contenido:**
  - Margen izquierdo ajustable segun estado del sidebar (`ml-60` o `ml-16`).
  - `<router-outlet>` para las paginas hijas.
  - Padding: `p-6`, fondo: `surface-50`.

**Patron clave:** ambos layouts son componentes padre en las rutas, conteniendo un `<router-outlet>` para los hijos. Asi se cambia completamente el layout entre auth y platform sin duplicar logica.

---

## 10. Servicios Core

### HttpPromiseBuilderService

Builder pattern para peticiones HTTP. Convierte Observables a Promises y maneja errores automaticamente.

```typescript
@Injectable({ providedIn: 'root' })
export class HttpPromiseBuilderService {
  constructor(
    private readonly http: HttpClient,
    private readonly uiDialogService: UiDialogService
  ) {}

  request<T = unknown>(): HttpPromiseRequestBuilder<T> {
    return new HttpPromiseRequestBuilder<T>(this.http, this.uiDialogService);
  }
}
```

**API fluida del builder:**

```typescript
// GET con paginacion (RT-10b: listados deben soportar paginacion/filtros)
const data = await this.httpBuilder
  .request<PaginatedResponse<CourseResponse>>()
  .get()
  .url(`${this.baseUrl}/courses`)
  .queryParam('page', 1)
  .queryParam('size', 12)
  .queryParam('q', 'angular')
  .send();

// POST con body
const result = await this.httpBuilder
  .request<EnrollmentResponse>()
  .post()
  .url(`${this.baseUrl}/courses/${courseId}/enroll`)
  .send();

// Modo silencioso (sin toast automatico de errores de negocio)
const result = await this.httpBuilder
  .request<MyResponse>()
  .get()
  .url(`${this.baseUrl}/endpoint`)
  .silent()
  .send();
```

**Manejo automatico de errores:**
- Errores de negocio (`response.code !== 0`): toast de error con el mensaje del backend.
- Errores HTTP: toast con el mensaje extraido del payload.
- Modo `silent()`: suprime el toast automatico, el caller maneja el error.

### UiDialogService

Wrapper sobre PrimeNG `ConfirmationService` y `MessageService` con API simplificada basada en Promises:

```typescript
@Injectable({ providedIn: 'root' })
export class UiDialogService {
  confirm(options: ConfirmOptions): Promise<boolean>
  alert(options: ConfirmOptions): Promise<boolean>        // Un solo boton
  confirmDelete(event?, itemLabel?): Promise<boolean>      // Header "Zona peligrosa"
  confirmApprove(event?, itemLabel?): Promise<boolean>     // Header "Aprobacion"

  showSuccess(summary, detail, life = 3000): void
  showError(summary, detail, life = 4000): void
  showInfo(summary, detail, life = 3000): void
  showWarn(summary, detail, life = 3000): void
}
```

**Uso tipico (aprobacion de curso, RF-15):**

```typescript
const confirmed = await this.uiDialogService.confirmApprove(
  event,
  '¿Aprobar el curso "Introduccion a Angular"?'
);
if (confirmed) {
  await this.approvalsService.approve(courseId, { comentario: '' });
  this.uiDialogService.showSuccess('Curso aprobado', 'El creador sera notificado');
}
```

### Patron de Servicios de Dominio

Cada entidad de negocio sigue la misma estructura de 3 archivos:

```
[entidad]Service/
├── [entidad].service.ts       # Logica de negocio + llamadas HTTP
├── [entidad].req.dto.ts       # Interfaces para request bodies
└── [entidad].res.dto.ts       # Interfaces para responses
```

**Servicios de dominio en SkillMaker (mapeados al modelo de datos):**

| Servicio | Endpoint base | Cubre |
|----------|--------------|-------|
| `AuthService` | `/auth` | Google OAuth, JWT, sesion |
| `UserService` | `/users` | RF-04, gestion de usuarios (admin) |
| `RoleService` | `/roles` | RF-04, asignacion de roles (admin) |
| `SupervisionService` | `/supervisions` | RF-04b, asignacion supervisor-empleados |
| `CourseService` | `/courses` | RF-05 a RF-09, catalogo (RF-18) |
| `SectionService` | `/courses/:id/sections` | secciones de un curso |
| `VideoService` | `/sections/:id/videos` | videos de una seccion |
| `MaterialService` | `/courses/:id/materials` | RF-07, material adjunto (URLs prefirmadas) |
| `EvaluationService` | `/courses/:id/evaluation` | RF-08, evaluacion del curso |
| `AttemptService` | `/evaluations/:id/attempts` | RF-13, intentos y respuestas |
| `EnrollmentService` | `/courses/:id/enroll` | RF-18, inscripcion |
| `CertificateService` | `/certificates` | RF-21, certificados (URL de descarga PDF) |
| `BadgeService` | `/badges` | RF-22, insignias y ranking |
| `ApprovalService` | `/approvals` | RF-15, revisar/aprobar/rechazar cursos |
| `ReportingService` | `/reports` | RF-23 a RF-25, reportes admin y supervisor |

Todos los servicios usan `HttpPromiseBuilderService` y soportan paginacion/filtros donde corresponde (RT-10b).

**Ejemplo tipico (CourseService):**

```typescript
@Injectable({ providedIn: 'root' })
export class CourseService {
  private readonly baseUrl = `${environment.apiBaseUrl}/courses`;

  constructor(private readonly httpBuilder: HttpPromiseBuilderService) {}

  async getCatalog(params: CatalogQuery): Promise<PaginatedResponse<CourseResponse>> {
    return this.httpBuilder
      .request<PaginatedResponse<CourseResponse>>()
      .get()
      .url(this.baseUrl)
      .queryParam('page', params.page)
      .queryParam('size', params.size)
      .queryParam('q', params.search ?? '')
      .send();
  }

  async getById(id: string): Promise<CourseDetailResponse> {
    return this.httpBuilder
      .request<CourseDetailResponse>()
      .get()
      .url(`${this.baseUrl}/${id}`)
      .send();
  }

  async create(payload: CreateCourseRequest): Promise<CourseResponse> {
    return this.httpBuilder
      .request<CourseResponse>()
      .post()
      .url(this.baseUrl)
      .body(payload)
      .send();
  }

  async submitForReview(id: string): Promise<void> {  // RF-10
    return this.httpBuilder
      .request<void>()
      .post()
      .url(`${this.baseUrl}/${id}/submit`)
      .send();
  }
}
```

---

## 11. Componentes del Dominio

### VideoEmbed (shared/components/video-embed/)

Componente que recibe un `url` y un `proveedor` (`'youtube' | 'vimeo'`) y renderiza un `<iframe>` con el embed correspondiente (RT-24). Hace sanitization de URLs para evitar XSS y construye la URL canonica de embed segun el proveedor.

```typescript
@Component({
  selector: 'app-video-embed',
  standalone: true,
  template: `
    <div class="aspect-video w-full">
      <iframe
        [src]="safeUrl()"
        class="w-full h-full rounded"
        frameborder="0"
        allow="autoplay; encrypted-media; picture-in-picture"
        allowfullscreen></iframe>
    </div>
  `,
})
export class VideoEmbedComponent {
  url = input.required<string>();
  proveedor = input.required<'youtube' | 'vimeo'>();
  private readonly sanitizer = inject(DomSanitizer);

  safeUrl = computed(() => {
    const embed = this.proveedor() === 'youtube'
      ? toYoutubeEmbed(this.url())
      : toVimeoEmbed(this.url());
    return this.sanitizer.bypassSecurityTrustResourceUrl(embed);
  });
}
```

### MaterialUploader (shared/components/material-uploader/)

Encapsula el flujo de upload via URL prefirmada (descrito en seccion 13.6 de `GUIA-MONOREPO.md`):

1. `POST /materials/presign` → obtener `{ uploadUrl, key }`.
2. `PUT uploadUrl` (directo al object storage) con el archivo.
3. `POST /materials` con la `key` y metadata para que el backend lo asocie al curso.

Valida tipo MIME y tamano antes de subir (RT-24b) para fallar rapido sin pegarle al backend.

### CourseCard (shared/components/course-card/)

Tarjeta reutilizable usada en Catalogo, Mis Cursos y Mi Contenido. Recibe un `CourseResponse` como input y emite eventos para navegacion/inscripcion.

---

## 12. Estilos y Temas

### Tailwind CSS (tailwind.css)

```css
@import "tailwindcss";
@plugin "tailwindcss-primeui";
@layer tailwind, primeng;
@source "../**/*.html";
@source "../**/*.ts";
```

Integra Tailwind con PrimeNG via el plugin `tailwindcss-primeui`, permitiendo usar clases como `primary-500`, `surface-50`, etc.

### Estilos Globales (styles.sass)

Importa PrimeIcons y define overrides globales para componentes PrimeNG:
- `.p-button-outlined`: mantiene borde de 1px.
- `.p-datatable`: headers en bold, font mas pequeno en el body.

### Tema PrimeNG

Configurado en `app.config.ts` usando el preset **Aura** como base. El color primario se mapea a una paleta de PrimeNG (`blue`, `indigo`, `green`, etc.). Para SkillMaker el placeholder es `blue`; ajustar segun la identidad visual definitiva.

Dark mode esta deshabilitado en el MVP (`darkModeSelector: 'none'`).

### angular.json (configuracion de estilos)

```json
{
  "styles": ["src/tailwind.css", "src/styles.sass"],
  "stylePreprocessorOptions": {
    "includePaths": ["src/app/sass"]
  }
}
```

---

## 13. Variables de Entorno

**Desarrollo** (`src/environments/environment.ts`):

```typescript
export const environment = {
  production: false,
  apiBaseUrl: 'http://localhost:3000/api',
  googleClientId: 'TU_CLIENT_ID.apps.googleusercontent.com',
  googleHostedDomain: 'tuempresa.com',
};
```

**Produccion** (`src/environments/environment.prod.ts`):

```typescript
export const environment = {
  production: true,
  apiBaseUrl: '/api',
  googleClientId: 'TU_CLIENT_ID.apps.googleusercontent.com',
  googleHostedDomain: 'tuempresa.com',
};
```

**Notas:**
- `apiBaseUrl` es absoluto en dev (`:4200` → `:3000`) y relativo en prod (nginx proxy, mismo origen).
- `googleClientId` es publico por diseno del flujo OAuth; no es un secret.
- `googleHostedDomain` restringe el dominio aceptado (RT-13). El backend igual debe revalidar.
- No incluye credenciales de prueba (a diferencia del template original): el sistema no maneja contrasenas (RNF-03).

---

## 14. Mapeo de Pantallas a Requerimientos Funcionales

Trazabilidad entre vistas del frontend y RFs del documento de requerimientos:

| Vista | RF | Notas |
|-------|----|----|
| `login` | RF-01 | Login con Google Workspace via GIS |
| `catalog` | RF-18, RF-18b | Catalogo + busqueda + paginacion |
| `course-detail` | RF-19 | Videos + material descargable |
| `course-player` | RF-19 | Reproductor embebido YouTube/Vimeo |
| `evaluation` | RF-11, RF-12, RF-13 | Opcion multiple + V/F + auto-calculo de nota |
| `my-courses` | RF-14, RF-20b | Cursos inscritos con puntaje y estado de completado |
| `certificates` | RF-21 | Descarga PDF de certificados |
| `badges` | RF-22 | Insignias y ranking |
| `creator/my-content` | RF-05, RF-09 | Cursos creados, edicion mientras no esten aprobados |
| `creator/course-edit` | RF-05, RF-06, RF-07, RF-08 | Crear curso + videos + material + evaluacion |
| `creator/submit-review` | RF-10 | Enviar curso a revision (cambio de estado) |
| `admin/approvals` | RF-15, RF-16, RF-17, RF-17b | Aprobar/rechazar + comentario + reenvio |
| `admin/user-management` | RF-04 | Gestion de usuarios y asignacion de roles |
| `admin/supervision-setup` | RF-04b | Asignar empleados a un supervisor |
| `admin/global-reports` | RF-23, RF-25 | Reportes globales de uso y puntajes |
| `supervisor/team-progress` | RF-24 | Avance y puntajes de los empleados a cargo |
| `profile` | (lectura) | Datos del usuario derivados del JWT |

Esta tabla deberia mantenerse actualizada cuando se agreguen nuevas vistas o cuando los RFs evolucionen.

---

## 15. Patrones Arquitectonicos (Resumen)

| Patron | Descripcion |
|---|---|
| **Standalone Components** | Sin NgModules, imports directos en cada componente |
| **Zoneless** | Change detection via Signals, sin Zone.js |
| **Functional Guards** | `CanActivateFn` en vez de clases con interface |
| **Functional Interceptors** | `HttpInterceptorFn` registrados con `withInterceptors()` |
| **Signals** | Estado reactivo para user, roles, session |
| **Builder Pattern** | `HttpPromiseBuilderService` con API fluida |
| **Singleton Refresh** | Una sola promise de refresh para evitar race conditions |
| **Lazy Loading** | `loadChildren` para modulos, `loadComponent` para componentes |
| **Layout como Padre** | Layouts como componentes padre en rutas con `<router-outlet>` |
| **DTOs separados** | `req.dto.ts` y `res.dto.ts` por cada servicio |
| **Auth delegada a Google** | OAuth ID token + JWT propio; sin contrasenas locales (RNF-03) |
| **Roles en JWT** | 4 roles fijos derivados del modelo de datos, validados en guards/directivas |
| **Upload via URL prefirmada** | Material adjunto sube directo a S3/MinIO, no pasa por el backend |
| **Embeds seguros** | Videos referenciados, nunca almacenados; sanitization en `VideoEmbed` |
| **Promise-based HTTP** | Todas las llamadas HTTP usan async/await via el builder |
| **OpenAPI como contrato** | Tipos TS generados desde `backend/docs/swagger.json` |

---

## 16. Contrato OpenAPI y Codegen de Tipos

### Por que

El backend expone su contrato via `backend/docs/swagger.json` (generado por swaggo/swag desde anotaciones Go). En vez de duplicar DTOs a mano en el frontend (propenso a desincronizacion silenciosa), usamos `openapi-typescript` para generar `src/app/api/types.ts` automaticamente.

### Prerequisito

`swag` debe estar en PATH. Si no lo esta:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### Comando

Desde la raiz del monorepo:

```bash
make types
```

Esto:
1. Ejecuta `make swagger` — regenera `backend/docs/swagger.json` desde las anotaciones Go.
2. Crea `frontend/src/app/api/` si no existe.
3. Ejecuta `openapi-typescript` contra el JSON generado → escribe `frontend/src/app/api/types.ts`.

### Cuando ejecutarlo

Cada vez que se agregue, renombre o elimine un campo en un DTO del backend o se modifique una anotacion swag en un handler. El diff de `types.ts` en el PR hace visible el cambio de contrato.

### El archivo generado ES parte del repositorio

`frontend/src/app/api/types.ts` esta comprometido en git, al igual que `backend/docs/swagger.json`. Esto permite:
- Que el IDE y el compilador resuelvan los tipos sin ejecutar `make types` primero.
- Que los cambios de contrato sean visibles en los diffs de PR.
- Que los contribuidores no-Go puedan trabajar sin recompilar el backend.

**No editar `types.ts` a mano.** Cualquier cambio manual se sobreescribira en la proxima ejecucion de `make types`.

### Notacion de corchetes (obligatoria)

`tsconfig.json` tiene `"noPropertyAccessFromIndexSignature": true`. Por eso, al acceder a rutas del objeto `paths`, se DEBE usar notacion de corchetes:

```typescript
import type { paths } from '@api/types';

// CORRECTO (notacion de corchetes)
type AuthPost = paths['/auth/google']['post'];

// ERROR de compilacion (notacion de punto no permitida en index types)
// type AuthPost = paths./auth/google.post;  // invalido sintacticamente de todas formas
```

Esto es comportamiento esperado de `openapi-typescript`, no un defecto. No "corregir" relajando la regla de tsconfig.

### Asercion de compilacion (`types.contract.ts`)

`frontend/src/app/api/types.contract.ts` es un archivo comprometido que actua como test de compilacion:

```typescript
import type { paths } from '@api/types';

// Si /auth/google o post desaparecen del spec, tsc falla aqui.
type _AuthPost = paths['/auth/google']['post'];
```

Para verificar el contrato:

```bash
cd frontend && npx tsc --noEmit --project tsconfig.app.json
```

Exit 0 = contrato valido. Exit no-0 = el spec cambio y hay que actualizar los consumidores.
