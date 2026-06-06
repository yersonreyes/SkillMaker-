/**
 * course-catalog.dto.ts — DTOs for the alumno-facing catalog + enrollment API (C2.4).
 *
 * These mirror the Go backend DTOs exactly (CatalogCourseCard, CoursePreviewResponse,
 * CourseDetailAlumnoResponse, EnrollmentResponse, MyCourseItem).
 * The Page<T> generic is defined locally (mirrors course.res.dto).
 *
 * Updated in course-structure-v2:
 * - CatalogCourseCard gains: miniaturaUrl, nivel, categorias[], cantidadClases, horasVideo, horasPractico
 * - VideoResponseItem gains: descripcion, materiales[]
 * - CourseDetailAlumnoResponse DROPS course-level materiales; gains metadata fields
 * - CoursePreviewResponse gains metadata fields (never had content — structural absence preserved)
 * - CategoriaItem added
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

/** One categoria reference (used inside course cards/details). */
export interface CategoriaItem {
  id: string;
  nombre: string;
  /** Slug from backend CategoriaResponse — matches categoriaService/categoria.dto.ts. */
  slug?: string;
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
 * Video item inside a section.
 * Updated in course-structure-v2: gains descripcion and materiales[].
 */
export interface VideoResponseItem {
  id: string;
  titulo: string;
  url: string;
  proveedor: 'youtube' | 'vimeo';
  orden: number;
  /** Description for this video (default ''). */
  descripcion: string;
  /** Per-video materials attached to this video. */
  materiales: MaterialResponseItem[];
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
  /** Course-level materiales REMOVED in course-structure-v2 (now per-video). */
  /** Metadata fields added in course-structure-v2: */
  nivel: string | null;
  categorias: CategoriaItem[];
  cantidadClases: number;
  horasVideo: number;
  horasPractico: number;
  /** Presigned miniatura URL (empty string when no miniatura set — backend never sends null). */
  miniaturaUrl: string;
}

/**
 * One approved-course card for the catalog list grid.
 * Updated in course-structure-v2: gains metadata fields.
 */
export interface CatalogCourseCard {
  id: string;
  titulo: string;
  descripcion: string;
  creadorNombre: string;
  createdAt: string; // ISO 8601
  /** Presigned miniatura URL (empty string when no miniatura set — backend never sends null). */
  miniaturaUrl: string;
  /** Nivel: 'basico' | 'intermedio' | 'avanzado' | null. */
  nivel: string | null;
  /** Associated categorias. */
  categorias: CategoriaItem[];
  /** Computed count of videos (across all sections). */
  cantidadClases: number;
  /** Computed total video hours (1 decimal). */
  horasVideo: number;
  /** Practical hours (stored). */
  horasPractico: number;
}

/**
 * Preview response — non-enrolled caller.
 * STRUCTURAL ABSENCE: no secciones/materiales/videos fields (not omitted — absent).
 * Updated in course-structure-v2: gains metadata fields (no content leak).
 */
export interface CoursePreviewResponse {
  enrolled: false;
  id: string;
  titulo: string;
  descripcion: string;
  creadorNombre: string;
  /** Metadata fields added in course-structure-v2: */
  nivel: string | null;
  categorias: CategoriaItem[];
  cantidadClases: number;
  horasVideo: number;
  horasPractico: number;
  /** Presigned miniatura URL (empty string when no miniatura set — backend never sends null). */
  miniaturaUrl: string;
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
