/**
 * course.req.dto.ts — Request DTOs and query param shapes for the Courses API.
 */

/** Query params for GET /api/courses?creator=me */
export interface CourseListParams {
  page?: number;
  size?: number;
}

export interface CreateCourseRequest {
  titulo: string;
  descripcion?: string;
}

export interface UpdateCourseRequest {
  titulo?: string;
  descripcion?: string;
}
