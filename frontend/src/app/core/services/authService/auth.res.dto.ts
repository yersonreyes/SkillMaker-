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
