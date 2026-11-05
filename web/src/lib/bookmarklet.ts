// The "Pin to Cork" bookmarklet. It runs on the page being pinned, so it can
// read things the server can't fetch (JS-rendered prices, sites like Amazon
// that block bots). It passes the page's price text, currency (when the page
// states one), language and main image to /new, which parses the price.
//
// This function is serialised with toString(), so it must be self-contained:
// no imports, no references to anything outside its own body.
function pinToCork(origin: string) {
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
  if (!price) {
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
  const img = meta('og:image') || meta('twitter:image');
  if (img)
    try {
      P.set('image', new URL(img, location.href).href);
    } catch (e) {}
  const u = origin + '/new?' + P;
  if (!window.open(u, 'cork', 'width=920,height=820')) location.href = u;
}

export function bookmarkletHref(origin: string): string {
  const code = '(' + pinToCork.toString() + ')(' + JSON.stringify(origin) + ')';
  return 'javascript:' + encodeURIComponent(code.replace(/\n\s*/g, '\n'));
}
