package scrape

import (
	"encoding/json"
	"regexp"
	"strings"
)

var priceRE = regexp.MustCompile(`^\d+(\.\d+)?$`)

// NormalizePrice turns things like "$1,299.00", "1.299,00 €" or " 49.9 " into a
// plain decimal string ("1299.00", "1299.00", "49.9"). It returns "" if the input
// doesn't look like a price.
func NormalizePrice(s string) string {
	s = strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || r == '.' || r == ',' {
			return r
		}
		return -1
	}, s)
	s = strings.Trim(s, ".,")
	lastDot, lastComma := strings.LastIndex(s, "."), strings.LastIndex(s, ",")
	switch {
	case lastDot >= 0 && lastComma >= 0:
		// whichever separator comes last is the decimal point
		if lastComma > lastDot {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	case lastComma >= 0:
		// "12,50" is a decimal comma; "1,299" is a thousands separator
		if strings.Count(s, ",") == 1 && len(s)-lastComma-1 <= 2 {
			s = strings.Replace(s, ",", ".", 1)
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	case strings.Count(s, ".") > 1:
		s = strings.ReplaceAll(s, ".", "") // "1.299.000"
	}
	if !priceRE.MatchString(s) {
		return ""
	}
	return s
}

var currencySymbols = map[string]string{
	"$": "USD", "US$": "USD", "CA$": "CAD", "C$": "CAD", "A$": "AUD", "AU$": "AUD", "NZ$": "NZD", "HK$": "HKD",
	"S$": "SGD", "MX$": "MXN", "R$": "BRL", "NT$": "TWD", "€": "EUR", "£": "GBP", "¥": "JPY", "￥": "JPY",
	"₹": "INR", "₩": "KRW", "₽": "RUB", "₺": "TRY", "₪": "ILS", "₱": "PHP", "฿": "THB", "₫": "VND", "₴": "UAH",
	"₦": "NGN", "zł": "PLN", "Kč": "CZK",
}

// NormalizeCurrency returns an upper-case ISO 4217 code, mapping common symbols.
func NormalizeCurrency(s string) string {
	s = strings.TrimSpace(s)
	if c, ok := currencySymbols[s]; ok {
		return c
	}
	s = strings.ToUpper(s)
	if len(s) == 3 && strings.Trim(s, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") == "" {
		return s
	}
	return ""
}

// priceFromJSONLD finds the first schema.org Product/Offer price in a JSON-LD blob.
func priceFromJSONLD(raw string) (price, currency string) {
	var v any
	if json.Unmarshal([]byte(raw), &v) != nil {
		return "", ""
	}
	var walk func(any, int) bool
	walk = func(n any, depth int) bool {
		if depth > 8 {
			return false
		}
		switch t := n.(type) {
		case []any:
			for _, c := range t {
				if walk(c, depth+1) {
					return true
				}
			}
		case map[string]any:
			if isType(t["@type"], "Offer", "AggregateOffer") {
				p := firstString(t["price"], t["lowPrice"])
				if p == "" {
					if spec, ok := t["priceSpecification"].(map[string]any); ok {
						p = firstString(spec["price"])
						if currency == "" {
							currency = firstString(spec["priceCurrency"])
						}
					}
				}
				if p = NormalizePrice(p); p != "" {
					price = p
					if c := firstString(t["priceCurrency"]); c != "" {
						currency = c
					}
					return true
				}
			}
			// look in offers first, then @graph and anything else nested
			for _, k := range []string{"offers", "@graph", "mainEntity", "itemOffered"} {
				if c, ok := t[k]; ok && walk(c, depth+1) {
					return true
				}
			}
		}
		return false
	}
	walk(v, 0)
	return price, NormalizeCurrency(currency)
}

func isType(v any, names ...string) bool {
	check := func(s string) bool {
		s = strings.TrimPrefix(strings.TrimPrefix(s, "http://schema.org/"), "https://schema.org/")
		for _, n := range names {
			if s == n {
				return true
			}
		}
		return false
	}
	switch t := v.(type) {
	case string:
		return check(t)
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok && check(s) {
				return true
			}
		}
	}
	return false
}

func firstString(vals ...any) string {
	for _, v := range vals {
		switch t := v.(type) {
		case string:
			if t != "" {
				return t
			}
		case float64:
			return strings.TrimRight(strings.TrimRight(jsonNumber(t), "0"), ".")
		case json.Number:
			return t.String()
		}
	}
	return ""
}

func jsonNumber(f float64) string {
	b, _ := json.Marshal(f)
	s := string(b)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}
