import { describe, expect, it } from 'vitest';
import { isDocumentRequest, isPortalEntryRequest } from './document-policy';

const request = (url: string, accept = 'text/html', method = 'GET', headers: Record<string, string> = {}) => ({ method, url, headers: { accept, ...headers } });

describe('portal document policy', () => {
  it('accepts navigation GET and HEAD requests, including the explicit entry', () => {
    expect(isPortalEntryRequest(request('/finance/applications'))).toBe(true);
    expect(isPortalEntryRequest(request('/index.html', 'text/html', 'HEAD'))).toBe(true);
    expect(isPortalEntryRequest(request('/admin/audit?x=1#history'))).toBe(true);
  });
  it('requires positive HTML intent and compatible Fetch Metadata', () => {
    expect(isDocumentRequest(request('/finance', 'text/html;q=0'))).toBe(false);
    expect(isDocumentRequest(request('/finance', '*/*'))).toBe(false);
    expect(isDocumentRequest(request('/finance', 'text/html', 'POST'))).toBe(false);
    expect(isDocumentRequest(request('/finance', 'text/html', 'GET', { 'sec-fetch-dest': 'script' }))).toBe(false);
    expect(isDocumentRequest(request('/finance', 'text/html', 'GET', { 'sec-fetch-mode': 'cors' }))).toBe(false);
  });
  it('never treats API, health, static, source, malformed, traversal, or file paths as entry HTML', () => {
    for (const path of ['/api', '/api/v1/session', '/healthz', '/readyz', '/assets/missing', '/assets/missing.js', '/src/main.tsx', '/node_modules/pkg', '/missing.css', '/%ZZ', '/%2e%2e/secret', '/a%2Fb%2Ejs']) {
      expect(isPortalEntryRequest(request(path))).toBe(false);
    }
  });
});
