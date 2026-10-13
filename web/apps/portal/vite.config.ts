import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import type { IncomingMessage, ServerResponse } from 'node:http';
import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import { isDocumentRequest, isPortalEntryRequest } from './src/document-policy';

const root = fileURLToPath(new URL('.', import.meta.url));
const env = loadEnv('development', fileURLToPath(new URL('../../../', import.meta.url)), ['JUSTIX_API', 'HTTP_ADDR']);
const api = env.JUSTIX_API || `http://${env.HTTP_ADDR || '127.0.0.1:8080'}`;

function earlyGuard(req: IncomingMessage, res: ServerResponse, next: () => void, allowHtmlProxy = false) {
  const path = (req.url ?? '/').split('?')[0] || '/';
  const htmlProxy = allowHtmlProxy && /\?html-proxy&index=\d+\.js$/.test(req.url ?? '');
  let decoded = '';
  try { decoded = decodeURIComponent(path); } catch { /* fallback rejects malformed paths */ }
  if (!htmlProxy && /\.html$/i.test(decoded) && !isDocumentRequest(req)) {
    res.writeHead(404, { 'Content-Type': 'text/plain; charset=utf-8' });
    res.end('Not found');
    return;
  }
  next();
}

function entryFallback(html: (url: string) => Promise<string>) {
  return async (req: IncomingMessage, res: ServerResponse) => {
    if (!isPortalEntryRequest(req)) {
      res.writeHead(404, { 'Content-Type': 'text/plain; charset=utf-8' });
      res.end('Not found');
      return;
    }
    try {
      const body = await html(req.url ?? '/');
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8', 'Cache-Control': 'no-store' });
      res.end(req.method === 'HEAD' ? undefined : body);
    } catch {
      res.writeHead(500, { 'Content-Type': 'text/plain; charset=utf-8' });
      res.end('Application entry unavailable');
    }
  };
}

export default defineConfig({
  root, base: '/', appType: 'custom',
  server: { port: 5191, strictPort: true, proxy: { '/api': api } },
  preview: { proxy: { '/api': api } },
  plugins: [react(), {
    name: 'portal-entry-html',
    configureServer(server) {
      server.middlewares.use((req, res, next) => earlyGuard(req, res, next, true));
      return () => server.middlewares.use(entryFallback(async (url) => server.transformIndexHtml('/index.html', await readFile(`${root}index.html`, 'utf8'), url)));
    },
    configurePreviewServer(server) {
      server.middlewares.use(earlyGuard);
      return () => server.middlewares.use(entryFallback(() => readFile(`${root}dist/index.html`, 'utf8')));
    },
  }],
});
