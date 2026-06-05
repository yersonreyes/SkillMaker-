/**
 * aprovaciones.component.ts — Admin approval queue (C4.1).
 *
 * Replaces the PendingViewComponent stub for the admin/approvals route.
 * Renders the list of courses awaiting admin review (estado=en_revision).
 * Approve: inline button. Reject: requires a non-empty comment (FE spec SEC-7).
 * Uses Cyanotype Workshop global primitives: .panel/.empty/.btn/.field/.tag.
 */
import {
  Component,
  inject,
  signal,
  computed,
  OnInit,
} from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DialogModule } from 'primeng/dialog';
import { TooltipModule } from 'primeng/tooltip';

import { ApprovalService } from '@core/services/approvalService/approval.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { PendingItem } from '@core/services/approvalService/approval.dto';

@Component({
  selector: 'app-aprovaciones',
  standalone: true,
  imports: [
    FormsModule,
    DatePipe,
    DialogModule,
    TooltipModule,
  ],
  templateUrl: './aprovaciones.component.html',
  styleUrl: './aprovaciones.component.sass',
})
export class AprovacionesComponent implements OnInit {
  private readonly approvalService = inject(ApprovalService);
  private readonly ui = inject(UiDialogService);

  // ── State ──────────────────────────────────────────────────────────────────
  readonly pending = signal<PendingItem[]>([]);
  readonly loading = signal<boolean>(false);

  // ── Reject dialog state ─────────────────────────────────────────────────
  readonly rejectDialogVisible = signal<boolean>(false);
  readonly rejectCourseId = signal<string>('');
  readonly rejectComentario = signal<string>('');
  readonly rejecting = signal<boolean>(false);

  /** Disable confirm button while the reject comment is empty. */
  readonly rejectConfirmDisabled = computed(() => !this.rejectComentario().trim());

  // ── Approve busy tracker ────────────────────────────────────────────────
  readonly approvingId = signal<string | null>(null);

  ngOnInit(): void {
    void this.loadPending();
  }

  async loadPending(): Promise<void> {
    this.loading.set(true);
    try {
      const items = await this.approvalService.listPending();
      this.pending.set(items);
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }

  /**
   * Approve a course in review.
   * The admin can optionally add a comment (not required for approval).
   */
  async approve(courseId: string, comentario?: string): Promise<void> {
    this.approvingId.set(courseId);
    try {
      await this.approvalService.approve(courseId, comentario);
      this.ui.showSuccess('Curso aprobado');
      await this.loadPending();
    } catch {
      // Error toast shown by builder
    } finally {
      this.approvingId.set(null);
    }
  }

  /**
   * Open the reject dialog for a specific course.
   * The dialog collects a REQUIRED rejection comment before calling reject.
   */
  openRejectDialog(courseId: string): void {
    this.rejectCourseId.set(courseId);
    this.rejectComentario.set('');
    this.rejectDialogVisible.set(true);
  }

  /**
   * Reject a course. Called from the dialog confirm button OR directly in tests.
   * Guards: comment must be non-empty (SEC-7 — no API call without comment).
   */
  async reject(courseId: string): Promise<void> {
    const comentario = this.rejectComentario().trim();
    if (!comentario) return; // SEC-7: empty comment → do nothing

    this.rejecting.set(true);
    try {
      await this.approvalService.reject(courseId, comentario);
      this.ui.showSuccess('Curso rechazado');
      this.rejectComentario.set('');
      this.rejectDialogVisible.set(false);
      await this.loadPending();
    } catch {
      // Error toast shown by builder
    } finally {
      this.rejecting.set(false);
    }
  }

  /** Called from dialog confirm button. */
  async confirmReject(): Promise<void> {
    const courseId = this.rejectCourseId();
    if (!courseId) return;
    await this.reject(courseId);
  }
}
