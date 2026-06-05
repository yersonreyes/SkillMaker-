/**
 * course-catalog.dto.ts — DTOs for the alumno-facing catalog + enrollment API (C2.4).
 *
 * These mirror the Go backend DTOs exactly (CatalogCourseCard, CoursePreviewResponse,
 * CourseDetailAlumnoResponse, EnrollmentResponse, MyCourseItem).
 * The Page<T> generic is defined locally (mirrors course.res.dto).
 *
 * NOTE: CatalogCourseCard is NOT in the auto-generated @api types.ts because the
 * swagger annotation for /catalog returns { [key: string]: unknown } — the pagination
 * wrapper hides the inner item type. These local types are the authoritative shapes.
 */

/** Generic pagination envelope — identical shape to Go's pagination.Page[T]. */
export interface Page<T> {
  items: T[];
  page: number;
  size: number;
  total: number;
  totalPages: number;
}

/** One approved-course card for the catalog list grid. */
export interface CatalogCourseCard {
  id: string;
  titulo: string;
  descripcion: string;
  creadorNombre: string;
  createdAt: string; // ISO 8601
}

/**
 * Preview response — non-enrolled caller.
 * STRUCTURAL ABSENCE: no secciones/materiales/videos fields (not omitted — absent).
 */
export interface CoursePreviewResponse {
  enrolled: false;
  id: string;
  titulo: string;
  descripcion: string;
  creadorNombre: string;
}

/** Material item inside the enrolled detail tree. */
export interface MaterialResponseItem {
  id: string;
  nombre: string;
  mimeType: string;
  tamanoBytes: number;
  createdAt: string;
}

/**
 * Video item inside a section. Intentionally partial: only the fields the
 * templates render. The backend struct also returns sectionId/duracionS/createdAt,
 * which are ignored at runtime (extra JSON keys are dropped). Extend if consumed.
 */
export interface VideoResponseItem {
  id: string;
  titulo: string;
  url: string;
  proveedor: 'youtube' | 'vimeo';
  orden: number;
}

/** Section with its videos — used in the enrolled detail tree. */
export interface SectionWithVideosItem {
  id: string;
  titulo: string;
  orden: number;
  videos: VideoResponseItem[];
}

/** Full enrolled detail — includes the content tree. */
export interface CourseDetailAlumnoResponse {
  enrolled: true;
  id: string;
  titulo: string;
  descripcion: string;
  creadorNombre: string;
  secciones: SectionWithVideosItem[];
  materiales: MaterialResponseItem[];
}

/** Discriminated union for getDetail() return type. */
export type CourseDetailResponse = CoursePreviewResponse | CourseDetailAlumnoResponse;

/** Response from POST /catalog/:id/enroll. */
export interface EnrollmentResponse {
  courseId: string;
  enrolled: boolean;
}

/** One row from GET /users/me/courses. */
export interface MyCourseItem {
  courseId: string;
  titulo: string;
  creadorNombre: string;
  completado: boolean;
  inscritoEn: string; // ISO 8601
}
