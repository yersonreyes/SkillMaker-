/**
 * reporting.dto.ts — DTOs for the reporting API (C6.1).
 *
 * These mirror the Go backend DTOs exactly.
 * Month is an ISO 8601 string in "yyyy-MM" format (e.g. "2026-01").
 */

/** Month bucket with count — used in usersPerMonth and approvedCoursesPerMonth. */
export interface MonthCountItem {
  month: string; // ISO "2006-01" / "yyyy-MM"
  total: number;
}

/** Course count by estado. */
export interface CoursesByEstadoItem {
  estado: string;
  total: number;
}

/** Creator ranked by aprobado course count. */
export interface TopCreatorItem {
  nombre: string;
  total: number;
}

/** System-wide aggregate metrics — GET /api/reports/global. */
export interface GlobalReportResponse {
  activeUsers: number;
  totalAttempts: number;
  certificatesIssued: number;
  coursesByEstado: CoursesByEstadoItem[];
  topCreators: TopCreatorItem[];
  usersPerMonth: MonthCountItem[];
  approvedCoursesPerMonth: MonthCountItem[];
}

/** Per-course report item — GET /api/reports/courses. */
export interface CourseReportItem {
  courseId: string;
  titulo: string;
  estado: string;
  enrollments: number;
  attempts: number;
  /** range [0.0, 1.0]; frontend renders as % */
  approvalRate: number;
}

/** User progress aggregate — GET /api/reports/users/:id/progress. */
export interface UserProgressResponse {
  enrolledCount: number;
  completedCount: number;
  attemptsCount: number;
  passedAttemptsCount: number;
  certificatesCount: number;
}

/** Team member progress row — GET /api/reports/team. */
export interface TeamReportItem {
  empleadoId: string;
  empleadoNombre: string;
  enrolledCount: number;
  completedCount: number;
  /** ISO 8601 date string or null if no attempts. */
  lastAttemptDate?: string | null;
}
