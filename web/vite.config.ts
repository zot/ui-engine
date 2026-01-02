import { defineConfig } from 'vite';

export default defineConfig({
  build: {
    outDir: '../site/html',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: 'index.html',
        worker: 'src/worker.ts',
      },
      output: {
        entryFileNames: '[name]-[hash].js',
      },
    },
  },
  server: {
    proxy: {
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
      '/api': {
        target: 'http://localhost:8080',
      },
    },
  },
});
