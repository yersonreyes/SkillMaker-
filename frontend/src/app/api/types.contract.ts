import type { paths } from '@api/types';

// Compile-time contract: fails if /auth/google or post is absent from the generated spec.
// Bracket notation required due to noPropertyAccessFromIndexSignature: true in tsconfig.json.
type _AuthPost = paths['/auth/google']['post'];
