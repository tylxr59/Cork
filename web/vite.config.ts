import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

const backend = process.env.CORK_BACKEND ?? 'http://localhost:8080';

export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/api': backend,
      '/media': backend,
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
});
