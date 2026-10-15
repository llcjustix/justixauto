export interface DocumentRequest {
  method?: string;
  url?: string;
  headers?: Record<string, string | string[] | undefined>;
}

const header = (request: DocumentRequest, name: string) => {
  const value = request.headers?.[name] ?? request.headers?.[name.toLowerCase()];
  return Array.isArray(value) ? value.join(',') : value;
};

export function isDocumentRequest(request: DocumentRequest): boolean {
  if (!['GET', 'HEAD'].includes(request.method ?? '')) return false;
  const acceptsHtml = header(request, 'accept')
    ?.split(',')
    .some((value) => {
      const [media, ...parameters] = value
        .trim()
        .toLowerCase()
        .split(';')
        .map((part) => part.trim());
      const quality = parameters.find((part) => part.startsWith('q='))?.slice(2);
      return media === 'text/html' && (quality === undefined || (Number(quality) > 0 && Number(quality) <= 1));
    });
  const destination = header(request, 'sec-fetch-dest');
  const mode = header(request, 'sec-fetch-mode');
  return Boolean(
    acceptsHtml &&
    (!destination || ['document', 'iframe', 'frame'].includes(destination)) &&
    (!mode || mode === 'navigate'),
  );
}

export function isPortalEntryRequest(request: DocumentRequest): boolean {
  if (!isDocumentRequest(request)) return false;
  const raw = (request.url ?? '/').split(/[?#]/, 1)[0] || '/';
  let path: string;
  try {
    path = decodeURIComponent(raw);
  } catch {
    return false;
  }
  if (!path.startsWith('/') || path.includes('\\') || path.split('/').includes('..')) return false;
  if (/^\/(?:api|healthz|readyz|assets|src|node_modules)(?:\/|$)/.test(path)) return false;
  if (path !== '/index.html' && /[.@%]/.test(path)) return false;
  return true;
}
