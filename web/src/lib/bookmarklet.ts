// Source of the "Pin to Cork" bookmarklet. It runs on the page being pinned,
// so it can read things the server can't fetch (JS-rendered prices, sites that
// block bots): the page's price, currency and main image are passed to /new.
const source = `
(() => {
  const q = (s) => document.querySelector(s);
  const m = (n) => (q('meta[property="' + n + '"],meta[name="' + n + '"]') || {}).content || '';
  let p = m('product:price:amount') || m('og:price:amount');
  let c = m('product:price:currency') || m('og:price:currency');
  if (!p) {
    const f = (o) => {
      if (!o || typeof o != 'object') return false;
      if (Array.isArray(o)) return o.some(f);
      const t = [].concat(o['@type'] || []);
      const v = o.price != null ? o.price : o.lowPrice;
      if (t.some((x) => /Offer$/.test(x)) && v != null) { p = String(v); c = o.priceCurrency || c; return true; }
      return f(o.offers) || f(o['@graph']) || f(o.mainEntity);
    };
    for (const s of document.querySelectorAll('script[type="application/ld+json"]')) {
      try { if (f(JSON.parse(s.textContent))) break; } catch (e) {}
    }
  }
  if (!p) {
    const e = q('[itemprop=price]');
    if (e) { p = e.getAttribute('content') || e.textContent; c = (q('[itemprop=priceCurrency]') || { getAttribute: () => c }).getAttribute('content') || c; }
  }
  const P = new URLSearchParams({ url: location.href, title: document.title, popup: '1' });
  if (p) { P.set('price', String(p).trim()); if (c) P.set('currency', c); }
  const i = m('og:image') || m('twitter:image');
  if (i) try { P.set('image', new URL(i, location.href).href); } catch (e) {}
  const u = ORIGIN + '/new?' + P;
  if (!window.open(u, 'cork', 'width=920,height=820')) location.href = u;
})()`;

export function bookmarkletHref(origin: string): string {
  const code = source
    .replace('ORIGIN', JSON.stringify(origin))
    .replace(/\n\s*/g, '')
    .replace(/\s*([=(){}[\],;:?|&+<>!])\s*/g, '$1');
  return 'javascript:' + encodeURIComponent(code).replace(/%20/g, ' ');
}
