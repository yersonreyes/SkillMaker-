import type { paths } from '@api/types';

// Compile-time contract: fails if /auth/google or post is absent from the generated spec.
// Bracket notation required due to noPropertyAccessFromIndexSignature: true in tsconfig.json.
// Exported to prevent @typescript-eslint/no-unused-vars warning; the export is intentional —
// this type is never consumed at runtime, only at compile time.
export type _AuthPost = paths['/auth/google']['post'];
