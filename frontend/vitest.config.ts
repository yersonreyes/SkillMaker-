import { defineConfig } from 'vitest/config';
import angular from '@analogjs/vite-plugin-angular';
import path from 'node:path';

export default defineConfig({
  plugins: [angular()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['src/test-setup.ts'],
    passWithNoTests: true,
  },
  resolve: {
    alias: {
      '@core': path.resolve(__dirname, 'src/app/core'),
      '@pages': path.resolve(__dirname, 'src/app/pages'),
      '@shared': path.resolve(__dirname, 'src/app/shared'),
    },
  },
});
