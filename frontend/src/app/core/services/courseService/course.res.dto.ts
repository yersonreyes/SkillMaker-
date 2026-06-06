/**
 * course.res.dto.ts — Response DTOs for the Courses API.
 *
 * Note: `Page<T>` is defined locally because the generated types.ts generic
 * pagination type is erased to `object`. Mirror the same shape as user.res.dto.
 * Updated in course-structure-v2: CourseDetail gains nivel, horasPractico, categoriaIds.
 */

export type CourseEstado = 'borrador' | 'en_revision' | 'aprobado' | 'rechazado';

/** Generic pagination envelope — mirrors Go's pagination.Page[T] JSON shape. */
export interface Page<T> {
  items: T[];
  page: number;
  size: number;
  total: number;
  totalPages: number;
}

export interface CourseListItem {
  id: string;
  titulo: string;
  estado: CourseEstado;
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

export interface CourseDetail {
  id: string;
  creadorId: string;
  titulo: string;
  descripcion: string;
  estado: CourseEstado;
  /** C2.2: true when the course has at least one video (via any section). */
  hasContent: boolean;
  /** Optional nivel: 'basico' | 'intermedio' | 'avanzado' (null if not set). */
  nivel?: string | null;
  /** Practical hours (stored, not computed). */
  horasPractico?: number;
  /** Presigned miniatura URL (null if not set). */
  miniaturaUrl?: string | null;
  /** Computed video hours (read-only). */
  horasVideo?: number;
  /** Computed video count (read-only). */
  cantidadClases?: number;
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}
