// The "Pin to Cork" bookmarklet. It runs on the page being pinned, so it can
// read things the server can't fetch (JS-rendered prices, sites like Amazon
// that block bots). It passes the page's price text, currency (when the page
// states one) and language to /new, which parses the price. When a page has
// several images it shows a picker over the page, and the ticked ones go to /new
// in the URL hash, which keeps long lists out of server and proxy URL limits.
//
// This function is serialised with toString(), so it must be self-contained:
// no imports, no references to anything outside its own body.
async function pinToCork(origin: string) {
  const q = (s: string) => document.querySelector(s);
  const meta = (n: string) =>
    (q('meta[property="' + n + '"],meta[name="' + n + '"]') as HTMLMetaElement | null)?.content?.trim() || '';

  // a number with a currency symbol or code on either side, e.g. "$20.23", "1.299,00 €", "CA$ 45", "USD 12"
  const SYM =
    '(?:US\\$|CA\\$|AU\\$|NZ\\$|HK\\$|MX\\$|NT\\$|C\\$|A\\$|S\\$|R\\$|[$€£¥￥₹₩₽₺₪₱฿₫₴₦]|(?<![A-Za-z])(?:USD|EUR|GBP|CAD|AUD|NZD|JPY|CNY|CHF|SEK|NOK|DKK|PLN|CZK|INR|MXN|BRL|ZAR|SGD|HKD|zł|kr|Kč|Ft|RM|Rp)(?![A-Za-z]))';
  const NUM = "\\d{1,3}(?:[.,'\\s\\u00a0\\u202f]\\d{3})*(?:[.,]\\d{1,2})?(?!\\d)|\\d+(?:[.,]\\d{1,2})?(?!\\d)";
  const PRICE = new RegExp('(?:' + SYM + ')[\\s\\u00a0]?(?:' + NUM + ')|(?:' + NUM + ')[\\s\\u00a0]?(?:' + SYM + ')');
  const priceIn = (s: string | null | undefined) => (s ? (s.replace(/\s+/g, ' ').match(PRICE) || [''])[0] : '');

  let price = '';
  let currency = '';

  // 1. a price the user selected
  price = priceIn(String(getSelection() || '').slice(0, 200));

  // 2. Open Graph / product meta tags
  if (!price) {
    price = meta('product:price:amount') || meta('og:price:amount');
    currency = meta('product:price:currency') || meta('og:price:currency');
  }

  // 3. schema.org JSON-LD
  if (!price) {
    const walk = (o: any, depth: number): boolean => {
      if (!o || typeof o != 'object' || depth > 8) return false;
      if (Array.isArray(o)) return o.some((x) => walk(x, depth + 1));
      const types = [].concat(o['@type'] || []) as string[];
      const spec = [].concat(o.priceSpecification || [])[0] as any;
      const v = o.price ?? o.lowPrice ?? spec?.price;
      if (types.some((t) => /Offer$/.test(t)) && v != null && v !== '') {
        price = String(v);
        currency = o.priceCurrency || spec?.priceCurrency || '';
        return true;
      }
      return ['offers', '@graph', 'mainEntity', 'itemOffered', 'hasVariant'].some((k) => walk(o[k], depth + 1));
    };
    for (const s of document.querySelectorAll('script[type="application/ld+json"]')) {
      try {
        if (walk(JSON.parse(s.textContent || ''), 0)) break;
      } catch (e) {}
    }
  }

  // 4. microdata
  if (!price) {
    const e = q('[itemprop=price]');
    if (e) {
      price = e.getAttribute('content') || (e as HTMLElement).innerText || '';
      const c = q('[itemprop=priceCurrency]');
      currency = c?.getAttribute('content') || c?.textContent || '';
    }
  }

  // 5. known shops (Amazon first) that keep the price in a predictable place
  if (!price) {
    const amazonValue = (q('#twister-plus-price-data-price') as HTMLInputElement | null)?.value;
    if (amazonValue) {
      price = amazonValue;
      currency = (q('#twister-plus-price-data-price-unit') as HTMLInputElement | null)?.value || '';
    }
  }
  if (!price) {
    for (const s of [
      '#corePriceDisplay_desktop_feature_div .priceToPay .a-offscreen',
      '#corePriceDisplay_desktop_feature_div .a-offscreen',
      '#corePriceDisplay_desktop_feature_div .aok-offscreen',
      '#corePrice_feature_div .a-offscreen',
      '#corePrice_desktop .a-offscreen',
      '#apex_desktop .a-offscreen',
      '#tp_price_block_total_price_ww .a-offscreen',
      '#priceblock_dealprice',
      '#priceblock_ourprice',
      '#price_inside_buybox',
      '#kindle-price',
      '#sns-base-price',
      '.a-price.priceToPay .a-offscreen',
      '.x-price-primary', // eBay
      '[data-buy-box-region=price] p', // Etsy
      '[data-testid=customer-price]', // Best Buy
      '[data-test=product-price]', // Target
      '[data-testid=price-wrap] [itemprop=price]', // Walmart
      '.price__current, .price-item--sale, .product__price', // Shopify themes
    ]) {
      for (const e of document.querySelectorAll(s)) {
        if ((price = priceIn(e.textContent))) break;
      }
      if (price) break;
    }
  }

  // 6. scan the page for the most prominent price-looking text
  // (not on Reddit, where the only prices are in its ads and comments)
  const reddit = /(^|\.)reddit\.com$/.test(location.hostname);
  if (!price && !reddit) {
    const bad = /was|old|list|strike|compare|original|regular|before|saving|save|ship|deliver|instal|month|unit|coupon|rrp|msrp|reward|points/i;
    const els = document.body.getElementsByTagName('*');
    let best = 0;
    for (let i = 0; i < els.length && i < 30000; i++) {
      const el = els[i] as HTMLElement;
      if (el.childElementCount > 8 || /^(SCRIPT|STYLE|NOSCRIPT|TEMPLATE|OPTION|SELECT|svg)$/.test(el.tagName)) continue;
      const text = el.textContent || '';
      if (text.length > 80) continue;
      const found = priceIn(text);
      if (!found || el.closest('s,del,strike,header,footer,nav') || el.querySelector('s,del,strike')) continue;
      const r = el.getBoundingClientRect();
      if (r.width < 1 || r.height < 1 || r.right < 0) continue;
      const cs = getComputedStyle(el);
      if (cs.visibility == 'hidden' || cs.textDecorationLine.includes('line-through')) continue;
      let score = parseFloat(cs.fontSize) || 12;
      if (+cs.fontWeight >= 600) score *= 1.2;
      const top = r.top + scrollY;
      if (top > innerHeight * 1.5) score *= 0.3; // far below the fold: recommendations, reviews…
      let ctx = '';
      for (let a: Element | null = el, k = 0; a && k < 4; a = a.parentElement, k++)
        ctx += ' ' + (a.getAttribute('class') || '') + ' ' + a.id + ' ' + (a.getAttribute('data-testid') || '');
      if (/price/i.test(ctx)) score *= 1.6;
      if (bad.test(ctx)) score *= 0.4;
      const around = el.parentElement?.textContent || '';
      if (around.length < 120 && /\/\s?mo\b|per month|\bwas\b|\bsave\b|\boff\b/i.test(around)) score *= 0.5;
      if (score > best) {
        best = score;
        price = found;
      }
    }
  }

  const P = new URLSearchParams({ url: location.href, title: document.title, popup: '1' });
  price = String(price).trim();
  if (price) {
    P.set('price', price);
    if (currency) P.set('currency', String(currency).trim());
    if (document.documentElement.lang) P.set('lang', document.documentElement.lang);
  }
  const images: string[] = [];
  const seen = new Set<string>();
  // the same picture at another size or on another host should count once:
  // Wikimedia's /thumb/a/b.jpg/330px-b.jpg, WordPress's b-300x200.jpg, Shopify's b_800x.jpg
  const sameImage = (u: URL) =>
    /\.(jpe?g|png|gif|webp|avif)$/i.test(u.pathname)
      ? u.pathname.replace(/\/thumb(\/.+\/([^/]+))\/\d+px-\2(\.[a-z]+)?$/i, '$1').replace(/[-_]\d+x\d*(?=\.[a-z]+$)/i, '')
      : u.pathname + u.search;
  const add = (raw: string | null | undefined) => {
    if (!raw || images.length >= 30) return;
    try {
      const u = new URL(raw, location.href);
      if (!/^https?:$/.test(u.protocol) || /\.svg$/i.test(u.pathname) || seen.has(sameImage(u))) return;
      seen.add(sameImage(u));
      images.push(u.href);
    } catch (e) {}
  };

  // Amazon has no og:image. Its gallery's full-size URLs are only in an inline script.
  const landing = q('#landingImage') as HTMLImageElement | null;
  if (landing) {
    add(landing.getAttribute('data-old-hires') || landing.src);
    for (const s of document.scripts) {
      if (!s.textContent?.includes('colorImages')) continue;
      for (const m of s.textContent.matchAll(/"hiRes":(?:"([^"]+)"|null),"thumb":"[^"]*","large":"([^"]+)"/g)) add(m[1] || m[2]);
    }
  }

  // Reddit: the post's own images, as originals. Every gallery image appears as
  // both a preview.redd.it and an i.redd.it URL, and this makes them the same.
  // The old design, which www.reddit.com also serves if it's set in your account,
  // only has small previews of a gallery, but they carry the same IDs.
  for (const e of document.querySelectorAll(
    'shreddit-post gallery-carousel figure img, shreddit-post img#post-image, #siteTable .media-gallery .gallery-tile img'
  )) {
    const src = e.getAttribute('src') || e.getAttribute('data-lazy-src') || '';
    const id = src.match(/^https:\/\/preview\.redd\.it\/(?:[^/?]*-)?([a-z0-9]+\.[a-z]+)(?:\?|$)/i);
    add(id ? 'https://i.redd.it/' + id[1] : src);
  }

  // Anything else on the old design, like single images, comes from the post's data.
  const post = location.pathname.match(/\/comments\/(\w+)/);
  if (reddit && post && !images.length && !q('shreddit-app')) {
    try {
      const res = await fetch('/comments/' + post[1] + '.json');
      const listing = (await res.json())[0].data.children[0].data;
      const data = listing.crosspost_parent_list?.[0] || listing;
      for (const { media_id } of data.gallery_data?.items || []) {
        const type = data.media_metadata?.[media_id]?.m || '';
        if (type.startsWith('image/')) add('https://i.redd.it/' + media_id + '.' + type.slice(6));
      }
      if (!images.length && /^https:\/\/i\.redd\.it\//.test(data.url)) add(data.url);
    } catch (e) {}
  }

  // anywhere else: the page's declared image, then what's on screen, biggest first
  if (!images.length) {
    add(meta('og:image') || meta('twitter:image'));
    const largest = (e: HTMLImageElement) => {
      // sites use data-pin-media to name the best version of an image for pinning
      let best = e.getAttribute('data-pin-media') || '';
      if (best) return best;
      best = e.currentSrc || e.src || e.getAttribute('data-src') || e.getAttribute('data-lazy-src') || '';
      let bestW = 0;
      for (const part of (e.getAttribute('srcset') || '').split(/,\s+/)) {
        const [u, w] = part.trim().split(/\s+/);
        if (u && parseFloat(w) > bestW) {
          best = u;
          bestW = parseFloat(w);
        }
      }
      return best;
    };
    const area = (e: HTMLImageElement) => {
      const r = e.getBoundingClientRect();
      const optedOut = e.hasAttribute('data-pin-nopin') || e.hasAttribute('nopin');
      const banner = (e.naturalWidth || r.width) > (e.naturalHeight || r.height) * 3;
      const tiny = r.width < 120 || r.height < 80 || (e.naturalWidth && e.naturalWidth < 120);
      return optedOut || banner || tiny ? 0 : r.width * r.height;
    };
    const onScreen = [...document.images].filter(area).sort((a, b) => area(b) - area(a));
    // anything far smaller than the biggest image is page furniture
    onScreen.filter((e) => area(e) > area(onScreen[0]) / 30).forEach((e) => add(largest(e)));
  }

  const send = (picked: string[]) => {
    const hash = picked.length ? '#' + new URLSearchParams(picked.map((i) => ['image', i])) : '';
    const u = origin + '/new?' + P + hash;
    if (!window.open(u, 'cork', 'width=920,height=820')) location.href = u;
  };

  document.getElementById('cork-picker')?.remove();
  if (images.length < 2) {
    send(images);
    return;
  }

  // The picker lives in a shadow root so the page's CSS can't reach it. It's built
  // without innerHTML or style attributes, which strict sites block.
  const el = <K extends keyof HTMLElementTagNameMap>(tag: K, cls: string, text?: string) => {
    const e = document.createElement(tag);
    e.className = cls;
    if (text) e.textContent = text;
    return e;
  };
  const host = document.createElement('div');
  host.id = 'cork-picker';
  const root = host.attachShadow({ mode: 'open' });
  const sheet = new CSSStyleSheet();
  sheet.replaceSync(
    ':host{all:initial;position:fixed;inset:0;z-index:2147483647}' +
      '.back{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;padding:24px;background:#0009;font:15px/1.4 system-ui,sans-serif;color-scheme:light dark}' +
      '.box{display:flex;flex-direction:column;gap:14px;width:min(860px,100%);max-height:100%;box-sizing:border-box;padding:20px;border-radius:16px;background:Canvas;color:CanvasText;box-shadow:0 12px 48px #0008}' +
      'h2{margin:0;font-size:18px}' +
      '.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(150px,1fr));gap:10px;overflow:auto;padding:3px}' +
      'label{position:relative;display:block;aspect-ratio:1;border-radius:10px;overflow:hidden;cursor:pointer;outline:2px solid #8884}' +
      'label:has(:checked){outline:3px solid #6d5dfc}' +
      'label:has(:focus-visible){outline:3px solid Highlight}' +
      'label img{width:100%;height:100%;object-fit:cover;display:block}' +
      'input{position:absolute;top:8px;left:8px;width:20px;height:20px;margin:0;accent-color:#6d5dfc;cursor:pointer}' +
      '.foot{display:flex;justify-content:flex-end;gap:10px}' +
      'button{font:inherit;font-weight:600;padding:9px 16px;border-radius:10px;border:1px solid #8886;background:none;color:inherit;cursor:pointer}' +
      'button.go{border-color:#6d5dfc;background:#6d5dfc;color:#fff}' +
      'button:disabled{opacity:.5;cursor:default}'
  );
  root.adoptedStyleSheets = [sheet];

  const back = el('div', 'back');
  const box = el('div', 'box');
  const grid = el('div', 'grid');
  const foot = el('div', 'foot');
  const cancel = el('button', '', 'Cancel');
  const go = el('button', 'go');
  const boxes: HTMLInputElement[] = [];
  const close = () => {
    host.remove();
    document.removeEventListener('keydown', onKey, true);
  };
  const onKey = (e: KeyboardEvent) => {
    if (e.key != 'Escape') return;
    e.stopPropagation();
    close();
  };
  const picked = () => boxes.filter((b) => b.checked && b.isConnected).map((b) => b.value);
  const update = () => {
    const n = picked().length;
    go.disabled = !n;
    go.textContent = n > 1 ? 'Pin ' + n + ' images' : 'Pin image';
  };

  images.forEach((src, i) => {
    const tile = el('label', '');
    const check = el('input', '');
    check.type = 'checkbox';
    check.value = src;
    check.checked = i == 0;
    check.setAttribute('aria-label', 'Image ' + (i + 1));
    check.onchange = update;
    const pic = el('img', '');
    pic.alt = '';
    const drop = () => {
      tile.remove();
      update();
    };
    pic.onerror = drop;
    pic.onload = () => {
      if (pic.naturalWidth < 120 || pic.naturalHeight < 80) drop();
    };
    pic.src = src;
    boxes.push(check);
    tile.append(check, pic);
    grid.append(tile);
  });

  cancel.onclick = close;
  go.onclick = () => {
    const list = picked();
    close();
    send(list);
  };
  back.onclick = (e) => {
    if (e.target == back) close();
  };
  foot.append(cancel, go);
  box.append(el('h2', '', 'Choose images to pin'), grid, foot);
  back.append(box);
  root.append(back);
  document.documentElement.append(host);
  document.addEventListener('keydown', onKey, true);
  update();
  go.focus();
}

export function bookmarkletHref(origin: string): string {
  // void, because a javascript: URL that returns a value can replace the page with it
  const code = 'void (' + pinToCork.toString() + ')(' + JSON.stringify(origin) + ')';
  return 'javascript:' + encodeURIComponent(code.replace(/\n\s*/g, '\n'));
}
