/**
 * course.req.dto.ts — Request DTOs and query param shapes for the Courses API.
 * Updated in course-structure-v2: added nivel, horasPractico, categoriaIds.
 */

/** Query params for GET /api/courses?creator=me */
export interface CourseListParams {
  page?: number;
  size?: number;
}

export interface CreateCourseRequest {
  titulo: string;
  descripcion?: string;
  /** One of 'basico' | 'intermedio' | 'avanzado' (optional). */
  nivel?: string;
  /** Practical hours associated with this course (optional, >= 0). */
  horasPractico?: number;
  /** Replace-set of categoria IDs to associate (optional). */
  categoriaIds?: string[];
}

export interface UpdateCourseRequest {
  titulo?: string;
  descripcion?: string;
  /** One of 'basico' | 'intermedio' | 'avanzado' (optional). */
  nivel?: string;
  /** Practical hours associated with this course (optional, >= 0). */
  horasPractico?: number;
  /** Replace-set of categoria IDs to associate (optional). */
  categoriaIds?: string[];
}

/** POST /api/courses/:courseId/thumbnail/presign — request body. */
export interface ThumbnailPresignRequest {
  nombre: string;
  contentType: string;
  tamanoBytes: number;
}

/** POST /api/courses/:courseId/thumbnail/presign — response body. */
export interface ThumbnailPresignResponse {
  uploadUrl: string;
  key: string;
  expiresAt: string; // ISO-8601
}

/** POST /api/courses/:courseId/thumbnail — confirm body. */
export interface ThumbnailConfirmRequest {
  key: string;
}
