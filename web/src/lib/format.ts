export function domain(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, '');
  } catch {
    return '';
  }
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

export function plural(n: number, word: string) {
  return `${n} ${word}${n === 1 ? '' : 's'}`;
}

/**
 * Formats a stored price for display: ("1299.5", "USD") → "$1,299.50". The
 * currency can also be a bare symbol when the code wasn't knowable, e.g.
 * ("20", "kr") → "20 kr"; ("20", "$") → "$20".
 */
export function formatPrice(price: string, currency: string): string {
  if (!price) return '';
  const n = Number(price);
  if (!Number.isFinite(n)) return price;
  if (/^[A-Z]{3}$/.test(currency)) {
    try {
      return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(n);
    } catch {
      // unknown currency code: fall through
    }
  }
  const s = new Intl.NumberFormat(undefined, { minimumFractionDigits: 0, maximumFractionDigits: 2 }).format(n);
  if (!currency) return s;
  // codes and letter symbols ("kr", "zł", "CHF") go after the number; signs like "$" before
  return /[A-Za-z]$/.test(currency) ? `${s} ${currency}` : `${currency}${s}`;
}
