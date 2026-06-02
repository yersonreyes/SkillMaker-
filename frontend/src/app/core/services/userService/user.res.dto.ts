/**
 * user.res.dto.ts — Response DTOs for the Users API.
 *
 * Note: `Page<T>` is defined here because the generated types.ts generic
 * pagination type is erased to `object` (PR-C must define it locally).
 */

export type UserRole = 'alumno' | 'creador' | 'supervisor' | 'administrador';

/** Generic pagination envelope — mirrors Go's pagination.Page[T] JSON shape. */
export interface Page<T> {
  items: T[];
  page: number;
  size: number;
  total: number;
  totalPages: number;
}

export interface UserListItem {
  id: string;
  email: string;
  nombre: string;
  activo: boolean;
  roles: UserRole[];
}

export interface UserDetail {
  id: string;
  email: string;
  nombre: string;
  activo: boolean;
  roles: UserRole[];
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

export interface SupervisionItem {
  id: string;
  supervisorId: string;
  supervisorName: string;
  empleadoId: string;
  empleadoName: string;
  creadoEn: string; // ISO 8601
}
