# Guia de Proyecto Base - Angular

Documentacion de la arquitectura, estructura y patrones del proyecto **gest-indap-front**. Sirve como referencia para replicar esta base en futuros proyectos Angular.

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
│   ├── core/                           # Singleton: guards, interceptors, services
│   │   ├── constants/
│   │   │   └── texts.ts                # Constantes de texto
│   │   ├── guards/
│   │   │   ├── auth.guard.ts           # Protege rutas autenticadas
│   │   │   ├── guest.guard.ts          # Protege rutas publicas (login, register)
│   │   │   └── permission.guard.ts     # Protege rutas por permisos
│   │   ├── interceptors/
│   │   │   ├── auth-token.interceptor.ts    # Inyecta Bearer token
│   │   │   └── auth-refresh.interceptor.ts  # Maneja 401 y refresca token
│   │   └── services/
│   │       ├── http-promise-builder.service.ts  # Builder HTTP (Promise-based)
│   │       ├── ui-dialog.service.ts             # Dialogos y toasts
│   │       ├── authService/                     # Autenticacion
│   │       │   ├── auth.service.ts
│   │       │   ├── auth.req.dto.ts
│   │       │   └── auth.res.dto.ts
│   │       ├── common/
│   │       │   ├── error-response.dto.ts
│   │       │   └── permission-check.service.ts  # Verificacion de permisos
│   │       └── [entidad]Service/                # Un folder por entidad de negocio
│   │           ├── [entidad].service.ts
│   │           ├── [entidad].req.dto.ts
│   │           └── [entidad].res.dto.ts
│   │
│   ├── Pages/                          # Feature modules (lazy loaded)
│   │   ├── auth/                       # Modulo de autenticacion
│   │   │   ├── auth.routes.ts          # Rutas hijas de /auth
│   │   │   ├── layout/                 # Layout de auth (two-column)
│   │   │   ├── login/
│   │   │   ├── register/
│   │   │   ├── recover-password/
│   │   │   └── new-password/
│   │   └── platform/                   # Modulo principal de la app
│   │       ├── platform.routes.ts      # Rutas hijas de /platform
│   │       ├── layout/                 # Layout de plataforma (sidebar + header)
│   │       ├── profile/
│   │       ├── user-management/
│   │       ├── role-management/
│   │       └── [feature-management]/   # Un folder por feature
│   │
│   ├── shared/                         # Componentes y directivas reutilizables
│   │   ├── components/
│   │   │   └── loading-wall/           # Overlay de carga fullscreen
│   │   └── directives/
│   │       └── has-permission.directive.ts  # Directiva estructural de permisos
│   │
│   └── sass/
│       └── _colors.sass               # Variables de color
│
├── environments/
│   ├── environment.ts                  # Desarrollo
│   └── environment.prod.ts             # Produccion
│
├── index.html                          # HTML principal
├── main.ts                             # Bootstrap
├── styles.sass                         # Estilos globales y overrides PrimeNG
└── tailwind.css                        # Config Tailwind + plugin PrimeUI
```

### Convenciones de Nombres

- **Servicios de dominio:** `[entidad]Service/` con 3 archivos: `service.ts`, `req.dto.ts`, `res.dto.ts`
- **Componentes de pagina:** `[feature]-management/` con `.ts`, `.html`, `.sass`
- **Sub-componentes:** Dentro del folder de su pagina padre (ej: `compras-modal/oc-form-dialog/`)
- **Guards e interceptors:** Funcionales (no clases), exportados como `const`

### Path Aliases (tsconfig.app.json)

```json
{
  "paths": {
    "@core/*": ["./src/app/core/*"],
    "@pages/*": ["./src/app/Pages/*"],
    "@shared/*": ["./src/app/shared/*"]
  }
}
```

---

## 3. Configuracion Base (app.config.ts)

```typescript
import {
  ApplicationConfig,
  provideBrowserGlobalErrorListeners,
  provideZonelessChangeDetection,
} from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { providePrimeNG } from 'primeng/config';
import { definePreset } from '@primeuix/themes';
import Aura from '@primeuix/themes/aura';

// Tema personalizado: color primario basado en la paleta green
const IndapPreset = definePreset(Aura, {
  semantic: {
    primary: {
      50: '{green.50}', 100: '{green.100}', 200: '{green.200}',
      300: '{green.300}', 400: '{green.400}', 500: '{green.500}',
      600: '{green.600}', 700: '{green.700}', 800: '{green.800}',
      900: '{green.900}', 950: '{green.950}',
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
        preset: IndapPreset,
        options: { darkModeSelector: 'none' }, // Sin dark mode
      },
      translation: esLocale,
    }),
  ],
};
```

**Puntos clave:**
- `provideZonelessChangeDetection()` elimina Zone.js, usa Signals para change detection
- Interceptors se registran como funciones via `withInterceptors([])`
- PrimeNG se configura con tema custom (Aura base + colores propios) y locale espanol
- `ConfirmationService` y `MessageService` se proveen globalmente para dialogos/toasts

---

## 4. Sistema de Rutas

### Rutas Principales (app.routes.ts)

```typescript
export const routes: Routes = [
  {
    path: 'auth',
    canActivate: [guestGuard],       // Solo usuarios NO autenticados
    loadChildren: () =>
      import('./Pages/auth/auth.routes').then((m) => m.AUTH_ROUTES),
  },
  {
    path: 'platform',
    canActivate: [authGuard],        // Solo usuarios autenticados
    loadChildren: () =>
      import('./Pages/platform/platform.routes').then((m) => m.PLATFORM_ROUTES),
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

**Auth Routes:**
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
        path: 'register',
        loadComponent: () =>
          import('./register/register.component').then((m) => m.RegisterComponent),
      },
      // ... mas rutas
      { path: '', redirectTo: 'login', pathMatch: 'full' },
    ],
  },
];
```

**Platform Routes (con permission guards):**
```typescript
export const PLATFORM_ROUTES: Routes = [
  {
    path: '',
    component: LayoutComponent,        // Layout de plataforma (sidebar + header)
    children: [
      {
        path: 'userManagement',
        loadComponent: () =>
          import('./user-management/user-management').then((m) => m.UserManagement),
        canActivate: [permissionGuard],
        data: { permissions: ['users:read'] },    // Permisos requeridos
      },
      {
        path: 'roleManagement',
        loadComponent: () =>
          import('./role-management/role-management').then((m) => m.RoleManagement),
        canActivate: [permissionGuard],
        data: { permissions: ['permission:read'] },
      },
      // ... mas rutas con permisos
      {
        path: 'profile',
        loadComponent: () =>
          import('./profile/profile.component').then((m) => m.ProfileComponent),
        // Sin permission guard - todos los usuarios autenticados acceden
      },
      { path: '', redirectTo: 'userManagement', pathMatch: 'full' },
    ],
  },
];
```

**Patron clave:** Cada componente se carga con `loadComponent` (lazy loading individual). Los permisos se definen en `data: { permissions: [...] }` y se validan con `permissionGuard`.

---

## 5. Sistema de Autenticacion

### AuthService (core/services/authService/auth.service.ts)

Servicio central que gestiona todo el ciclo de vida de autenticacion usando **Angular Signals** para estado reactivo.

```typescript
@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly baseUrl = `${environment.apiBaseUrl}/auth`;

  // ─── Estado reactivo (Signals) ─────────────────────────────
  public user: WritableSignal<User | null> = signal<User | null>(null);
  public sessionExpired: WritableSignal<boolean> = signal(false);
  public userPermissions: WritableSignal<string[]> = signal<string[]>([]);
  public isSuperAdmin: WritableSignal<boolean> = signal<boolean>(false);

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

#### Token Management

Tokens almacenados en `localStorage` con claves `auth_token` y `auth_refresh_token`:

```typescript
  getToken(): string | null {
    return localStorage.getItem('auth_token');
  }

  getRefreshToken(): string | null {
    return localStorage.getItem('auth_refresh_token');
  }

  private setTokens(accessToken: string, refreshToken: string): void {
    localStorage.setItem('auth_token', accessToken);
    localStorage.setItem('auth_refresh_token', refreshToken);
  }

  private clearTokens(): void {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_refresh_token');
    localStorage.removeItem('auth_user');
  }
```

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

#### Flujo de Login

```typescript
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await this.httpBuilder
      .request<LoginResponse>()
      .post()
      .url(`${this.baseUrl}/login`)
      .body(credentials)
      .send();

    this.setTokens(response.access_token, response.refresh_token);
    this.loadPermissionsFromToken(response.access_token);

    // Derivar info del usuario desde el JWT payload
    const decoded = this.decodeToken(response.access_token);
    if (decoded) {
      const user: User = {
        id: decoded.sub,
        email: decoded.email,
        nombre: decoded.nombre ?? '',
        // ... mas campos del JWT
      };
      this.user.set(user);
      localStorage.setItem('auth_user', JSON.stringify(user));
    }

    this.sessionExpired.set(false);
    this.startExpirationTimer();
    return response;
  }
```

#### Token Refresh con Singleton Pattern (anti race-condition)

```typescript
  async refresh(): Promise<RefreshTokenResponse> {
    // Si ya hay un refresh en curso, retornar la misma promise
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
    this.loadPermissionsFromToken(response.access_token);
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

    if (token && userJson) {
      const user = JSON.parse(userJson) as User;
      this.user.set(user);
      this.loadPermissionsFromToken(token);

      if (!this.isTokenExpired()) {
        this.startExpirationTimer();
      } else if (!this.isRefreshTokenExpired()) {
        this.refreshWithRetry().catch(() => this.sessionExpired.set(true));
      } else {
        this.sessionExpired.set(true);
      }
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
  └─> POST /auth/login
       └─> setTokens() → loadPermissions() → user.set() → startExpirationTimer()

Peticion HTTP
  └─> authTokenInterceptor: agrega Bearer token
       └─> Si 401 → authRefreshInterceptor: refresh() → reintenta peticion

Timer Expiracion (5 min antes)
  └─> refreshWithRetry(3 intentos, backoff exponencial)
       ├─ Exito → nuevos tokens → reinicia timer
       └─ Fallo → sessionExpired = true → modal de re-login

Logout
  └─> POST /auth/logout → clearTokens() → user = null → navegar a /auth
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

  // Token valido
  if (token && !authService.isTokenExpired()) return true;

  // Token expirado pero refresh valido → permite (el interceptor/timer refresca)
  if (token && authService.isTokenExpired()) {
    const refreshToken = authService.getRefreshToken();
    if (refreshToken && !authService.isRefreshTokenExpired()) return true;
    // Ambos expirados → sesion expirada
    authService.sessionExpired.set(true);
    return true;
  }

  // Sin token → redirige a login
  void router.navigate(['/auth']);
  return false;
};
```

### Guest Guard

Impide que usuarios autenticados accedan a login/register:

```typescript
export const guestGuard: CanActivateFn = () => {
  const authService = inject(AuthService);
  const router = inject(Router);
  const token = authService.getToken();

  if (!token) return true;
  router.navigate(['/platform']);
  return false;
};
```

### Permission Guard

Valida permisos definidos en `route.data`:

```typescript
export const permissionGuard: CanActivateFn = (route) => {
  const permissionService = inject(PermissionCheckService);
  const router = inject(Router);
  const requiredPermissions = route.data?.['permissions'] as string[] | undefined;

  if (!requiredPermissions || requiredPermissions.length === 0) return true;

  const hasAccess = permissionService.hasAllPermissions(requiredPermissions);
  if (hasAccess) return true;

  console.warn(`Acceso denegado: faltan permisos [${requiredPermissions.join(', ')}]`);
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
      // No reintentar si la peticion es al endpoint de refresh (evita loop infinito)
      if (req.url.includes('/auth/refresh')) return throwError(() => error);
      if (!authService.getRefreshToken()) return throwError(() => error);

      // Convertir el refresh (Promise) a Observable y reintentar
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

## 8. Sistema de Permisos

### PermissionCheckService

Servicio central para verificar permisos del usuario, con soporte para Signals reactivos:

```typescript
@Injectable({ providedIn: 'root' })
export class PermissionCheckService {
  constructor(private readonly authService: AuthService) {}

  get permissions(): Signal<string[]> {
    return this.authService.userPermissions;
  }

  hasPermission(permission: string): boolean {
    return this.authService.userPermissions().includes(permission);
  }

  hasAllPermissions(permissions: string[]): boolean {
    if (!permissions || permissions.length === 0) return true;
    const userPerms = this.authService.userPermissions();
    return permissions.every((p) => userPerms.includes(p));
  }

  hasAnyPermission(permissions: string[]): boolean {
    if (!permissions || permissions.length === 0) return true;
    const userPerms = this.authService.userPermissions();
    return permissions.some((p) => userPerms.includes(p));
  }

  // Signals reactivos para templates
  hasPermission$(permission: string): Signal<boolean> {
    return computed(() => this.authService.userPermissions().includes(permission));
  }

  hasAllPermissions$(permissions: string[]): Signal<boolean> {
    return computed(() => {
      const userPerms = this.authService.userPermissions();
      return permissions.every((p) => userPerms.includes(p));
    });
  }
}
```

### HasPermission Directive

Directiva estructural que muestra/oculta elementos segun permisos. Reacciona automaticamente a cambios via `effect()`:

```typescript
@Directive({ selector: '[hasPermission]', standalone: true })
export class HasPermissionDirective {
  private readonly permissionService = inject(PermissionCheckService);
  private readonly templateRef = inject(TemplateRef<unknown>);
  private readonly viewContainer = inject(ViewContainerRef);
  private readonly destroyRef = inject(DestroyRef);
  private permissions: string[] = [];
  private isRendered = false;

  @Input()
  set hasPermission(value: string | string[]) {
    this.permissions = Array.isArray(value) ? value : [value];
  }

  constructor() {
    const effectRef = effect(() => {
      this.permissionService.permissions(); // Establece dependencia reactiva
      this.updateView();
    });
    this.destroyRef.onDestroy(() => effectRef.destroy());
  }

  private updateView(): void {
    const hasAccess = this.permissionService.hasAllPermissions(this.permissions);
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
<!-- Un solo permiso -->
<button *hasPermission="'users:update'">Editar Usuario</button>

<!-- Multiples permisos (requiere TODOS) -->
<div *hasPermission="['users:read', 'users:update']">
  Visible solo si tiene ambos permisos
</div>
```

---

## 9. Layouts

### Auth Layout (Pages/auth/layout/)

Layout de dos columnas para las paginas de autenticacion:

- **Columna izquierda:** Formulario centrado (flex, max-width-md) con `<router-outlet>` para login/register/recover
- **Columna derecha:** Panel de bienvenida con branding institucional (oculto en mobile con `hidden lg:flex`)
- **Fondo:** green-900 en el panel derecho (colores corporativos)
- **Responsive:** Mobile-first, el panel derecho desaparece en pantallas pequenas

### Platform Layout (Pages/platform/layout/)

Layout principal de la aplicacion con tres zonas:

- **Sidebar (fijo, izquierda):**
  - Colapsable: 16rem expandido, 4rem colapsado
  - Logo + titulo de la app
  - Menu de navegacion con secciones (items con icono PrimeIcons)
  - Footer con "Soporte y ayuda" y "Cerrar sesion"
  - Tooltips visibles cuando el sidebar esta colapsado
  - Usa PrimeNG Ripple y Tooltip

- **Header (sticky, top):**
  - Titulo de la app + icono a la izquierda
  - Seccion de usuario a la derecha (avatar, nombre, rol)
  - Popover menu con enlace a perfil y logout

- **Area de contenido:**
  - Margen izquierdo ajustable segun estado del sidebar (`ml-60` o `ml-16`)
  - `<router-outlet>` para las paginas hijas
  - Padding: `p-6`, fondo: `surface-50`

**Patron comun:** Ambos layouts son componentes padre en las rutas. Contienen un `<router-outlet>` donde se renderizan las paginas hijas. Esto permite cambiar completamente el layout entre auth y platform sin duplicar logica.

---

## 10. Servicios Core

### HttpPromiseBuilderService

Builder pattern para peticiones HTTP. Convierte Observables a Promises y maneja errores automaticamente:

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
// GET con query params
const data = await this.httpBuilder
  .request<MiResponse>()
  .get()
  .url(`${this.baseUrl}/endpoint`)
  .queryParam('page', 1)
  .queryParam('size', 10)
  .send();

// POST con body
const result = await this.httpBuilder
  .request<MiResponse>()
  .post()
  .url(`${this.baseUrl}/endpoint`)
  .body(payload)
  .send();

// Modo silencioso (sin toast automatico de errores de negocio)
const result = await this.httpBuilder
  .request<MiResponse>()
  .get()
  .url(`${this.baseUrl}/endpoint`)
  .silent()
  .send();
```

**Manejo automatico de errores:**
- Errores de negocio (`response.code !== 0`): muestra toast de error con el mensaje
- Errores HTTP: muestra toast con el mensaje extraido del error
- Modo `silent()`: suprime el toast automatico, el caller maneja el error

### UiDialogService

Wrapper sobre PrimeNG `ConfirmationService` y `MessageService` con API simplificada basada en Promises:

```typescript
@Injectable({ providedIn: 'root' })
export class UiDialogService {
  // Confirmaciones (devuelven Promise<boolean>)
  confirm(options: ConfirmOptions): Promise<boolean>
  alert(options: ConfirmOptions): Promise<boolean>        // Un solo boton
  confirmDelete(event?, itemLabel?): Promise<boolean>      // Header "Zona peligrosa"
  confirmApprove(event?, itemLabel?): Promise<boolean>     // Header "Aprobacion"

  // Toasts
  showSuccess(summary, detail, life = 3000): void   // Verde
  showError(summary, detail, life = 4000): void     // Rojo
  showInfo(summary, detail, life = 3000): void       // Azul
  showWarn(summary, detail, life = 3000): void       // Amarillo
}
```

**Uso tipico:**

```typescript
// Confirmacion de eliminacion
const confirmed = await this.uiDialogService.confirmDelete(event, '¿Desea eliminar este registro?');
if (confirmed) {
  await this.service.delete(id);
  this.uiDialogService.showSuccess('Eliminado', 'Registro eliminado correctamente');
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

Todos los servicios usan `HttpPromiseBuilderService` para las peticiones HTTP. Ejemplo tipico:

```typescript
@Injectable({ providedIn: 'root' })
export class MiEntidadService {
  private readonly baseUrl = `${environment.apiBaseUrl}/mi-entidad`;

  constructor(private readonly httpBuilder: HttpPromiseBuilderService) {}

  async getAll(): Promise<MiEntidadResponse[]> {
    return this.httpBuilder
      .request<MiEntidadResponse[]>()
      .get()
      .url(this.baseUrl)
      .send();
  }

  async create(payload: CreateMiEntidadRequest): Promise<MiEntidadResponse> {
    return this.httpBuilder
      .request<MiEntidadResponse>()
      .post()
      .url(this.baseUrl)
      .body(payload)
      .send();
  }
}
```

---

## 11. Estilos y Temas

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
- `.p-button-outlined`: mantiene borde de 1px
- `.p-datatable`: headers en bold, font mas pequeno en el body

### Tema PrimeNG

Configurado en `app.config.ts` usando el preset **Aura** como base. El color primario se mapea a la paleta **green** de PrimeNG. Dark mode esta deshabilitado (`darkModeSelector: 'none'`).

Para cambiar el color primario en otro proyecto, modificar el objeto `semantic.primary` en `definePreset()` usando otra paleta (ej: `{blue.50}`, `{indigo.50}`, etc.).

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

## 12. Patrones Arquitectonicos (Resumen)

| Patron | Descripcion |
|---|---|
| **Standalone Components** | Sin NgModules, imports directos en cada componente |
| **Zoneless** | Change detection via Signals, sin Zone.js |
| **Functional Guards** | `CanActivateFn` en vez de clases con interface |
| **Functional Interceptors** | `HttpInterceptorFn` registrados con `withInterceptors()` |
| **Signals** | Estado reactivo para user, permissions, session |
| **Builder Pattern** | `HttpPromiseBuilderService` con API fluida |
| **Singleton Refresh** | Una sola promise de refresh para evitar race conditions |
| **Lazy Loading** | `loadChildren` para modulos, `loadComponent` para componentes |
| **Layout como Padre** | Layouts como componentes padre en rutas con `<router-outlet>` |
| **DTOs separados** | `req.dto.ts` y `res.dto.ts` por cada servicio |
| **Permisos en JWT** | Permisos embebidos en el token, validados en guards/directivas/servicios |
| **Promise-based HTTP** | Todas las llamadas HTTP usan async/await via el builder |
