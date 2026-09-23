import { defineConfig } from 'vite';

const target = process.env.API_PROXY_TARGET || 'http://127.0.0.1:8080';
const proxy = Object.fromEntries(['/v1', '/swagger', '/graph.json'].map(path => [path, {target, changeOrigin: true}]));

export default defineConfig({
  server: {port: 5173, strictPort: true, proxy},
  preview: {port: 5173, strictPort: true, proxy},
  build: {outDir: 'dist', emptyOutDir: true, rollupOptions: {output: {manualChunks: {graph: ['cytoscape']}}}},
});
