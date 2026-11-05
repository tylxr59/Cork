// Turning price text found on a page ("$20.23", "1.299,00 €", "CA$ 45") into
// an amount and a currency, and working out which currency a bare "$" means.

const EURO = 'AT BE BG CY DE EE ES FI FR GR HR IE IT LT LU LV MT NL PT SI SK'.split(' ');

// ISO region (or country-code TLD) → currency
const REGION: Record<string, string> = {
  US: 'USD', CA: 'CAD', GB: 'GBP', UK: 'GBP', AU: 'AUD', NZ: 'NZD', IN: 'INR', JP: 'JPY', CN: 'CNY', HK: 'HKD',
  SG: 'SGD', MX: 'MXN', BR: 'BRL', CH: 'CHF', SE: 'SEK', NO: 'NOK', DK: 'DKK', IS: 'ISK', PL: 'PLN', CZ: 'CZK',
  HU: 'HUF', RO: 'RON', TR: 'TRY', IL: 'ILS', ZA: 'ZAR', KR: 'KRW', TW: 'TWD', TH: 'THB', PH: 'PHP', MY: 'MYR',
  ID: 'IDR', VN: 'VND', AE: 'AED', SA: 'SAR', RU: 'RUB', UA: 'UAH', AR: 'ARS', CL: 'CLP', CO: 'COP', PE: 'PEN',
  EG: 'EGP', NG: 'NGN',
  ...Object.fromEntries(EURO.map((r) => [r, 'EUR'])),
};

// Symbols, longest first so "CA$" wins over "$". `null` means ambiguous: see AMBIGUOUS.
const SYMBOLS: [string, string | null][] = [
  ['US$', 'USD'], ['CA$', 'CAD'], ['AU$', 'AUD'], ['NZ$', 'NZD'], ['HK$', 'HKD'], ['MX$', 'MXN'], ['NT$', 'TWD'],
  ['C$', 'CAD'], ['A$', 'AUD'], ['S$', 'SGD'], ['R$', 'BRL'],
  ['€', 'EUR'], ['£', 'GBP'], ['₹', 'INR'], ['₩', 'KRW'], ['₽', 'RUB'], ['₺', 'TRY'], ['₪', 'ILS'], ['₱', 'PHP'],
  ['฿', 'THB'], ['₫', 'VND'], ['₴', 'UAH'], ['₦', 'NGN'], ['zł', 'PLN'], ['Kč', 'CZK'], ['Ft', 'HUF'], ['RM', 'MYR'],
  ['Rp', 'IDR'], ['kr', null], ['$', null], ['¥', null], ['￥', null],
];

const AMBIGUOUS: Record<string, { options: string[]; fallback: string }> = {
  $: { options: ['USD', 'CAD', 'AUD', 'NZD', 'MXN', 'SGD', 'HKD', 'TWD', 'ARS', 'CLP', 'COP'], fallback: 'USD' },
  '¥': { options: ['JPY', 'CNY'], fallback: 'JPY' },
  '￥': { options: ['JPY', 'CNY'], fallback: 'JPY' },
  // no good default between SEK, NOK, DKK and ISK: keep the symbol itself
  kr: { options: ['SEK', 'NOK', 'DKK', 'ISK'], fallback: 'kr' },
};

let known: Set<string> | undefined;
export function isCurrencyCode(s: string): boolean {
  if (!/^[A-Z]{3}$/.test(s)) return false;
  known ??= new Set([...Object.values(REGION), ...(Intl.supportedValuesOf?.('currency') ?? [])]);
  return known.has(s);
}

function regionOf(locale: string): string {
  try {
    return new Intl.Locale(locale).maximize().region ?? '';
  } catch {
    return '';
  }
}

/** The currency of the user's locale, e.g. "CAD" for en-CA. */
export function localCurrency(): string {
  for (const l of navigator.languages ?? [navigator.language]) {
    const c = REGION[regionOf(l)];
    if (c) return c;
  }
  return 'USD';
}

/** Currency suggested by a site's country TLD (amazon.co.uk → GBP), or "". */
export function siteCurrency(url: string): string {
  try {
    const tld = new URL(url).hostname.split('.').pop()!.toUpperCase();
    return tld.length === 2 ? (REGION[tld] ?? '') : '';
  } catch {
    return '';
  }
}

/** Currency suggested by a page's lang attribute (en-CA → CAD), or "" if it has no region. */
export function langCurrency(lang: string): string {
  return lang && /[-_]/.test(lang) ? (REGION[regionOf(lang)] ?? '') : '';
}

/**
 * Picks the currency an ambiguous symbol most likely means. `hints` are
 * currencies in order of preference (the site's country, the page language,
 * the user's own locale); the first one that uses the symbol wins.
 */
function resolveSymbol(sym: string, hints: string[]): string {
  const a = AMBIGUOUS[sym];
  if (!a) return sym;
  return hints.find((h) => a.options.includes(h)) ?? a.fallback;
}

/** "$1,299.00" → "1299.00", "1.299,00 €" → "1299.00", "12,50" → "12.50". "" if there's no number. */
export function normalizeAmount(s: string): string {
  const m = s.match(/\d[\d.,'\s  ]*/);
  if (!m) return '';
  let n = m[0].replace(/[^\d.,]/g, '').replace(/[.,]+$/, '');
  const dot = n.lastIndexOf('.');
  const comma = n.lastIndexOf(',');
  if (dot >= 0 && comma >= 0) {
    // whichever separator comes last is the decimal point
    n = comma > dot ? n.replace(/\./g, '').replace(',', '.') : n.replace(/,/g, '');
  } else if (comma >= 0) {
    // "12,50" is a decimal comma; "1,299" is a thousands separator
    n = n.split(',').length === 2 && n.length - comma - 1 <= 2 ? n.replace(',', '.') : n.replace(/,/g, '');
  } else if (n.split('.').length > 2) {
    n = n.replace(/\./g, ''); // "1.299.000"
  }
  return /^\d+(\.\d+)?$/.test(n) ? n : '';
}

/** Finds the currency named in price text: an ISO code ("USD 20") or a symbol ("$20"). "" if none. */
export function detectCurrency(text: string, hints: string[] = []): string {
  // upper case only: "all" or "top" in prose aren't the lek or the pa'anga
  for (const m of text.matchAll(/(?<![A-Za-z])[A-Z]{3}(?![A-Za-z])/g)) {
    if (isCurrencyCode(m[0])) return m[0];
  }
  for (const [sym, code] of SYMBOLS) {
    if (text.includes(sym)) return code ?? resolveSymbol(sym, hints);
  }
  return '';
}

/**
 * Splits price text into an amount and currency. A currency already given
 * (e.g. from structured data) wins; otherwise it's read from the text, and
 * `hints` settle ambiguous symbols like "$".
 */
export function parsePrice(text: string, currency = '', hints: string[] = []): { amount: string; currency: string } {
  const amount = normalizeAmount(text);
  const given = currency.trim();
  const c = isCurrencyCode(given.toUpperCase()) ? given.toUpperCase() : detectCurrency(given || text, hints);
  return { amount, currency: amount ? c || given : '' };
}

/** Currencies to try for a pin, most likely first: the site's country, the page language, then the user's locale. */
export function currencyHints(url: string, lang = ''): string[] {
  return [...new Set([siteCurrency(url), langCurrency(lang), localCurrency()].filter(Boolean))];
}

/**
 * Cleans up a price the user typed: "$20.23" becomes amount "20.23" with the
 * currency taken from the text (else the one already set). Returns null if
 * the text has no number in it.
 */
export function tidyPrice(price: string, currency: string, hints: string[]): { price: string; currency: string } | null {
  const amount = normalizeAmount(price);
  if (!amount) return null;
  return { price: amount, currency: detectCurrency(price, hints) || currency.trim() };
}
