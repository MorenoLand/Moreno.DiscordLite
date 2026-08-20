import { defineConfig } from 'vite';

export default defineConfig({
  server: {
    host: '127.0.0.1',
    port: Number(process.env.WAILS_VITE_PORT || 9245)
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});
