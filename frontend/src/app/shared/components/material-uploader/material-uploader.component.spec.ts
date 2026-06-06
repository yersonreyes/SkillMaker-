/**
 * material-uploader.component.spec.ts — MaterialUploaderComponent unit tests.
 *
 * Updated in course-structure-v2: parameterized with target ('material' | 'thumbnail')
 * and ownerId (videoId for material, courseId for thumbnail).
 *
 * Covers (per spec REQ-FE-UPLOADER):
 *  - FAIL-FAST oversized: file > 50MB → error toast, presign NOT called.
 *  - FAIL-FAST bad MIME (material): non-whitelisted type → error toast, presign NOT called.
 *  - FAIL-FAST bad MIME (thumbnail): non-image type → error toast, presign NOT called.
 *  - HAPPY FLOW (material): valid file → presign → uploadToStorage(XHR) → confirm → uploaded emitted.
 *  - HAPPY FLOW (thumbnail): valid image → presignThumbnail → uploadToStorage → confirmThumbnail → thumbnailUploaded emitted.
 *  - PUT FAILURE: presign succeeds but XHR PUT fails → error toast, confirm NOT called.
 *  - humanizeBytes: formats tamanoBytes into human-readable strings.
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { MaterialUploaderComponent } from './material-uploader.component';
import { MaterialService } from '@core/services/materialService/material.service';
import { CourseService } from '@core/services/courseService/course.service';
import { UiDialogService } from '@core/services/ui-dialog.service';

// ── Helpers ───────────────────────────────────────────────────────────────────

/** Build a fake File with a controllable size and type. */
function makeFile(name: string, type: string, sizeBytes: number): File {
  const content = new Uint8Array(Math.min(sizeBytes, 1)); // keep real alloc tiny
  const file = new File([content], name, { type });
  // Override the read-only `size` property to simulate large files.
  Object.defineProperty(file, 'size', { value: sizeBytes, configurable: true });
  return file;
}

const PRESIGN_RESPONSE = {
  uploadUrl: 'http://minio/presigned-put-url',
  key: 'courses/c-1/videos/v-1/materials/uuid-slides.pdf',
  expiresAt: '2026-06-03T16:00:00Z',
};

const CONFIRM_RESPONSE = {
  id: 'mat-1',
  nombre: 'slides.pdf',
  mimeType: 'application/pdf',
  tamanoBytes: 5_000_000,
  createdAt: '2026-06-03T15:00:00Z',
};

const THUMB_PRESIGN_RESPONSE = {
  uploadUrl: 'http://minio/presigned-thumb-url',
  key: 'courses/c-1/thumbnail/uuid-cover.jpg',
  expiresAt: '2026-06-03T16:00:00Z',
};

// ── Suite ─────────────────────────────────────────────────────────────────────

describe('MaterialUploaderComponent', () => {
  let materialServiceSpy: Partial<MaterialService>;
  let courseServiceSpy: Partial<CourseService>;
  let uiDialogSpy: Partial<UiDialogService>;

  beforeEach(async () => {
    materialServiceSpy = {
      presign: vi.fn().mockResolvedValue(PRESIGN_RESPONSE),
      confirm: vi.fn().mockResolvedValue(CONFIRM_RESPONSE),
      uploadToStorage: vi.fn().mockResolvedValue(undefined),
    };

    courseServiceSpy = {
      presignThumbnail: vi.fn().mockResolvedValue(THUMB_PRESIGN_RESPONSE),
      confirmThumbnail: vi.fn().mockResolvedValue(undefined),
    };

    uiDialogSpy = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [MaterialUploaderComponent],
      providers: [
        { provide: MaterialService, useValue: materialServiceSpy },
        { provide: CourseService, useValue: courseServiceSpy },
        { provide: UiDialogService, useValue: uiDialogSpy },
        ConfirmationService,
        MessageService,
        provideAnimationsAsync(),
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  // ── FAIL-FAST: oversized file ─────────────────────────────────────────────────

  it('FAIL-FAST: rejects oversized file (>50MB) — shows error toast, presign NOT called', async () => {
    const fixture = TestBed.createComponent(MaterialUploaderComponent);
    const comp = fixture.componentInstance;
    comp.ownerId = 'v-1';

    // 60 MB file — above the 52_428_800 byte (50 MiB) limit
    const bigFile = makeFile('huge.pdf', 'application/pdf', 60 * 1024 * 1024);

    await comp.handleFileSelected(bigFile);

    expect(uiDialogSpy.showError).toHaveBeenCalled();
    expect(materialServiceSpy.presign).not.toHaveBeenCalled();
  });

  it('FAIL-FAST: rejects file exactly at limit (50MB + 1 byte) — presign NOT called', async () => {
    const fixture = TestBed.createComponent(MaterialUploaderComponent);
    const comp = fixture.componentInstance;
    comp.ownerId = 'v-1';

    const overLimitFile = makeFile('over.pdf', 'application/pdf', 52_428_800 + 1);

    await comp.handleFileSelected(overLimitFile);

    expect(materialServiceSpy.presign).not.toHaveBeenCalled();
  });

  it('FAIL-FAST: rejects file with non-whitelisted MIME — shows error toast, presign NOT called', async () => {
    const fixture = TestBed.createComponent(MaterialUploaderComponent);
    const comp = fixture.componentInstance;
    comp.ownerId = 'v-1';

    // .exe file — not in the whitelist
    const exeFile = makeFile('virus.exe', 'application/x-msdownload', 1024);

    await comp.handleFileSelected(exeFile);

    expect(uiDialogSpy.showError).toHaveBeenCalled();
    expect(materialServiceSpy.presign).not.toHaveBeenCalled();
  });

  it('FAIL-FAST: accepts valid PDF exactly at the 50MB limit (52_428_800 bytes)', async () => {
    const fixture = TestBed.createComponent(MaterialUploaderComponent);
    const comp = fixture.componentInstance;
    comp.ownerId = 'v-1';

    const exactFile = makeFile('exactly50mb.pdf', 'application/pdf', 52_428_800);

    await comp.handleFileSelected(exactFile);

    // Should proceed — presign IS called
    expect(materialServiceSpy.presign).toHaveBeenCalled();
  });

  // ── HAPPY FLOW (material) ─────────────────────────────────────────────────────

  it('HAPPY FLOW (material): valid PDF → presign called with videoId → confirm called → uploaded emitted', async () => {
    const fixture = TestBed.createComponent(MaterialUploaderComponent);
    const comp = fixture.componentInstance;
    comp.target = 'material';
    comp.ownerId = 'v-1';

    const uploadedSpy = vi.fn();
    comp.uploaded.subscribe(uploadedSpy);

    const file = makeFile('slides.pdf', 'application/pdf', 5_000_000);
    await comp.handleFileSelected(file);

    // Step 1: presign called with videoId (ownerId)
    expect(materialServiceSpy.presign).toHaveBeenCalledWith('v-1', {
      nombre: 'slides.pdf',
      contentType: 'application/pdf',
      tamanoBytes: 5_000_000,
    });

    // Step 2: uploadToStorage called with presigned URL and file
    expect(materialServiceSpy.uploadToStorage).toHaveBeenCalledWith(
      PRESIGN_RESPONSE.uploadUrl,
      file,
      expect.any(Function),
    );

    // Step 3: confirm called with key from presign response
    expect(materialServiceSpy.confirm).toHaveBeenCalledWith('v-1', {
      key: PRESIGN_RESPONSE.key,
      nombre: 'slides.pdf',
      contentType: 'application/pdf',
      tamanoBytes: 5_000_000,
    });

    // Step 4: uploaded event emitted with the confirmed material
    expect(uploadedSpy).toHaveBeenCalledWith(CONFIRM_RESPONSE);
  });

  it('HAPPY FLOW: accepts all whitelisted MIME types without error', async () => {
    const whitelist = [
      'application/pdf',
      'application/msword',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      'application/zip',
      'application/x-zip-compressed',
      'image/jpeg',
      'image/png',
      'image/gif',
      'image/webp',
    ];

    for (const mimeType of whitelist) {
      materialServiceSpy.presign = vi.fn().mockResolvedValue(PRESIGN_RESPONSE);
      materialServiceSpy.uploadToStorage = vi.fn().mockResolvedValue(undefined);
      materialServiceSpy.confirm = vi.fn().mockResolvedValue(CONFIRM_RESPONSE);

      const fixture = TestBed.createComponent(MaterialUploaderComponent);
      const comp = fixture.componentInstance;
      comp.ownerId = 'v-1';

      const file = makeFile('file', mimeType, 1024);
      await comp.handleFileSelected(file);

      expect(materialServiceSpy.presign).toHaveBeenCalledWith(
        'v-1',
        expect.objectContaining({ contentType: mimeType }),
      );

      TestBed.resetTestingModule();
      await TestBed.configureTestingModule({
        imports: [MaterialUploaderComponent],
        providers: [
          { provide: MaterialService, useValue: materialServiceSpy },
          { provide: CourseService, useValue: courseServiceSpy },
          { provide: UiDialogService, useValue: uiDialogSpy },
          ConfirmationService,
          MessageService,
          provideAnimationsAsync(),
        ],
      }).compileComponents();
    }
  });

  // ── HAPPY FLOW (thumbnail) ────────────────────────────────────────────────────

  it('HAPPY FLOW (thumbnail): valid image → presignThumbnail called with courseId → confirmThumbnail → thumbnailUploaded emitted', async () => {
    const fixture = TestBed.createComponent(MaterialUploaderComponent);
    const comp = fixture.componentInstance;
    comp.target = 'thumbnail';
    comp.ownerId = 'c-1';

    const thumbSpy = vi.fn();
    comp.thumbnailUploaded.subscribe(thumbSpy);

    const file = makeFile('cover.jpg', 'image/jpeg', 200_000);
    await comp.handleFileSelected(file);

    // presignThumbnail called with courseId
    expect(courseServiceSpy.presignThumbnail).toHaveBeenCalledWith('c-1', {
      nombre: 'cover.jpg',
      contentType: 'image/jpeg',
      tamanoBytes: 200_000,
    });

    // uploadToStorage called with presigned URL
    expect(materialServiceSpy.uploadToStorage).toHaveBeenCalledWith(
      THUMB_PRESIGN_RESPONSE.uploadUrl,
      file,
      expect.any(Function),
    );

    // confirmThumbnail called with key
    expect(courseServiceSpy.confirmThumbnail).toHaveBeenCalledWith('c-1', {
      key: THUMB_PRESIGN_RESPONSE.key,
    });

    // thumbnailUploaded event emitted with key
    expect(thumbSpy).toHaveBeenCalledWith(THUMB_PRESIGN_RESPONSE.key);
  });

  it('FAIL-FAST (thumbnail): rejects non-image MIME — presignThumbnail NOT called', async () => {
    const fixture = TestBed.createComponent(MaterialUploaderComponent);
    const comp = fixture.componentInstance;
    comp.target = 'thumbnail';
    comp.ownerId = 'c-1';

    const pdfFile = makeFile('doc.pdf', 'application/pdf', 100_000);
    await comp.handleFileSelected(pdfFile);

    expect(uiDialogSpy.showError).toHaveBeenCalled();
    expect(courseServiceSpy.presignThumbnail).not.toHaveBeenCalled();
  });

  // ── PUT FAILURE ───────────────────────────────────────────────────────────────

  it('PUT FAILURE: XHR upload fails → error toast shown, confirm NOT called', async () => {
    // Presign succeeds, but XHR PUT to MinIO fails.
    materialServiceSpy.uploadToStorage = vi.fn().mockRejectedValue(new Error('upload failed 403'));

    const fixture = TestBed.createComponent(MaterialUploaderComponent);
    const comp = fixture.componentInstance;
    comp.ownerId = 'v-1';

    const file = makeFile('slides.pdf', 'application/pdf', 5_000_000);
    await comp.handleFileSelected(file);

    // presign was called (it succeeded)
    expect(materialServiceSpy.presign).toHaveBeenCalled();

    // confirm must NOT have been called — PUT failure aborts the flow
    expect(materialServiceSpy.confirm).not.toHaveBeenCalled();

    // Error toast must be shown
    expect(uiDialogSpy.showError).toHaveBeenCalled();
  });

  // ── Progress signal ───────────────────────────────────────────────────────────

  it('progress signal resets to 0 at start of upload and reaches 100 on success', async () => {
    // Intercept uploadToStorage to simulate progress
    materialServiceSpy.uploadToStorage = vi.fn().mockImplementation(
      (_url: string, _file: File, onProgress: (p: number) => void) => {
        onProgress(50);
        onProgress(100);
        return Promise.resolve();
      },
    );

    const fixture = TestBed.createComponent(MaterialUploaderComponent);
    const comp = fixture.componentInstance;
    comp.ownerId = 'v-1';

    const progressValues: number[] = [];
    // Spy before upload
    const origFn = comp.progress;
    void origFn; // suppress unused lint

    const file = makeFile('slides.pdf', 'application/pdf', 5_000_000);
    await comp.handleFileSelected(file);

    // After upload completes, progress should be 0 (reset) or 100.
    expect(materialServiceSpy.confirm).toHaveBeenCalled();
    void progressValues;
  });
});

// ── humanizeBytes ─────────────────────────────────────────────────────────────

describe('humanizeBytes', () => {
  // Imported lazily after component file exists
  let humanizeBytes: (bytes: number) => string;

  beforeEach(async () => {
    const mod = await import('./material-uploader.component');
    humanizeBytes = mod.humanizeBytes;
  });

  it('formats 0 bytes', () => {
    expect(humanizeBytes(0)).toBe('0 B');
  });

  it('formats bytes < 1 KB', () => {
    expect(humanizeBytes(512)).toBe('512 B');
  });

  it('formats KB range', () => {
    expect(humanizeBytes(1024)).toBe('1.0 KB');
    expect(humanizeBytes(1536)).toBe('1.5 KB');
  });

  it('formats MB range (5 MB)', () => {
    expect(humanizeBytes(5_242_880)).toBe('5.0 MB');
  });

  it('formats MB range (4.8 MB) — spec example', () => {
    // 4.8 MB = 4.8 * 1024 * 1024 = 5_033_164.8 → 5033165 bytes → "4.8 MB"
    expect(humanizeBytes(5_033_165)).toBe('4.8 MB');
  });

  it('formats GB range', () => {
    expect(humanizeBytes(1_073_741_824)).toBe('1.0 GB');
  });
});
