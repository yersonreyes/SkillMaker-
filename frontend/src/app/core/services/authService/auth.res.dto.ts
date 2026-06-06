export type UserRole = 'alumno' | 'creador' | 'supervisor' | 'administrador';

export interface UserPublic {
  id: string;
  email: string;
  nombre: string;
  roles: UserRole[];
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_at: string; // ISO 8601
  user: UserPublic;
}

export type RefreshTokenResponse = LoginResponse;

export interface JwtPayload {
  sub: string;
  email: string;
  nombre: string;
  roles: UserRole[];
  exp: number;
  iat: number;
}

/** C8.1 — one active refresh-token session for the caller. */
export interface SessionResponse {
  id: string;
  ip?: string;
  userAgent?: string;
  createdAt: string; // ISO 8601
  expiresAt: string; // ISO 8601
  usedAt?: string;   // ISO 8601 — null if session has never rotated
}
