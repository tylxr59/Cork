// A tiny history-API router. Links are plain <a href>; clicks on same-origin
// links are intercepted globally. Pins opened from a grid render as a modal
// over the page they were opened from (history.state.bg holds that page).

interface Location {
  path: string;
  query: URLSearchParams;
  /** When set, `path` is shown as a modal over this background URL. */
  bg: string | null;
}

function read(): Location {
  const bg = (history.state?.bg as string | undefined) ?? null;
  return { path: location.pathname, query: new URLSearchParams(location.search), bg };
}

export const router = $state<{ current: Location }>({ current: read() });

// Identifies history entries pushed during this page load, so closing a modal
// only uses history.back() when the previous entry is really the background.
const LOAD_ID = Math.random().toString(36).slice(2);

export function navigate(href: string, opts: { replace?: boolean; modal?: boolean } = {}) {
  const state: { bg?: string; load?: string } = { load: LOAD_ID };
  if (opts.modal) {
    // keep the original background when hopping between modals
    state.bg = router.current.bg ?? location.pathname + location.search;
  }
  if (opts.replace) history.replaceState(state, '', href);
  else history.pushState(state, '', href);
  router.current = read();
  if (!opts.modal) window.scrollTo(0, 0);
}

/** Closes a pin modal by going back if we opened it, else to its background. */
export function closeModal() {
  const bg = router.current.bg;
  if (!bg) return;
  if (history.state?.load === LOAD_ID) history.back();
  else navigate(bg, { replace: true });
}

export function setQuery(params: Record<string, string>) {
  const q = new URLSearchParams(location.search);
  for (const [k, v] of Object.entries(params)) {
    if (v) q.set(k, v);
    else q.delete(k);
  }
  const s = q.toString();
  history.replaceState(history.state, '', location.pathname + (s ? `?${s}` : ''));
  router.current = read();
}

window.addEventListener('popstate', () => {
  router.current = read();
});

document.addEventListener('click', (e) => {
  if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
  const a = (e.target as Element).closest?.('a');
  if (!a || a.target || a.hasAttribute('download') || a.origin !== location.origin) return;
  if (a.pathname.startsWith('/api/') || a.pathname.startsWith('/media/')) return;
  e.preventDefault();
  navigate(a.pathname + a.search, { modal: a.dataset.modal !== undefined });
});

export function match(pattern: string, path: string): Record<string, string> | null {
  const p = pattern.split('/');
  const s = path.replace(/\/+$/, '').split('/');
  if (pattern === '/' && (path === '/' || path === '')) return {};
  if (p.length !== s.length) return null;
  const params: Record<string, string> = {};
  for (let i = 0; i < p.length; i++) {
    if (p[i].startsWith(':')) params[p[i].slice(1)] = decodeURIComponent(s[i]);
    else if (p[i] !== s[i]) return null;
  }
  return params;
}
