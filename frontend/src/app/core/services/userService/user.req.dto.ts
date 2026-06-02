/**
 * user.req.dto.ts — Request DTOs and query param shapes for the Users API.
 */
import type { UserRole } from './user.res.dto';

/** Query params for GET /api/users */
export interface UserListParams {
  page?: number;
  size?: number;
  q?: string;
  role?: UserRole | '';
  active?: boolean;
}

export interface RolesPatchRequest {
  add: UserRole[];
  remove: UserRole[];
}

export interface ActivePatchRequest {
  active: boolean;
}

export interface SupervisionCreateRequest {
  supervisorId: string;
  empleadoId: string;
}
