package scrape

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParseHTML(t *testing.T) {
	base, _ := url.Parse("https://example.com/recipes/pie")
	doc := `<html><head>
		<title> Fallback   title </title>
		<meta property="og:title" content="Apple Pie &amp; Cream">
		<meta name="description" content="A classic.">
		<meta property="og:image" content="/img/hero.jpg">
		</head><body>
		<img src="/img/hero.jpg">
		<img src="pixel.gif" width="1" height="1">
		<img data-src="//cdn.example.com/lazy.png" src="data:image/gif;base64,AAAA">
		<img srcset="a-small.jpg 300w, a-large.jpg 1200w">
		<img src="logo.svg">
		</body></html>`
	p := parseHTML(strings.NewReader(doc), base)
	if p.Title != "Apple Pie & Cream" || p.Description != "A classic." {
		t.Fatalf("metadata: %+v", p)
	}
	want := []string{
		"https://example.com/img/hero.jpg",
		"https://cdn.example.com/lazy.png",
		"https://example.com/recipes/a-large.jpg",
	}
	if strings.Join(p.Images, " ") != strings.Join(want, " ") {
		t.Fatalf("images:\n got %v\nwant %v", p.Images, want)
	}
}

func TestIsPublic(t *testing.T) {
	for ip, want := range map[string]bool{
		"8.8.8.8": true, "127.0.0.1": false, "10.1.2.3": false, "192.168.1.5": false,
		"169.254.169.254": false, "100.100.1.1": false, "::1": false, "fd00::1": false,
		"2606:4700::1111": true,
	} {
		if got := isPublic(net.ParseIP(ip)); got != want {
			t.Errorf("isPublic(%s) = %v, want %v", ip, got, want)
		}
	}
}

func TestPrice(t *testing.T) {
	base, _ := url.Parse("https://shop.example.com/p/1")
	cases := map[string][2]string{
		`<script type="application/ld+json">{"@context":"https://schema.org","@graph":[{"@type":"WebPage"},
			{"@type":"Product","name":"Lamp","offers":[{"@type":"Offer","price":"129.00","priceCurrency":"usd"}]}]}</script>`: {"129.00", "USD"},
		`<script type="application/ld+json">{"@type":"Product","offers":{"@type":"AggregateOffer","lowPrice":49.5,"priceCurrency":"EUR"}}</script>`: {"49.5", "EUR"},
		`<meta property="product:price:amount" content="1,299.99"><meta property="product:price:currency" content="GBP">`:                           {"1299.99", "GBP"},
		`<span itemprop="price" content="1.299,00">1.299,00 €</span><meta itemprop="priceCurrency" content="EUR">`:                                  {"1299.00", "EUR"},
		`<p>No price here</p>`: {"", ""},
	}
	for doc, want := range cases {
		p := parseHTML(strings.NewReader(doc), base)
		if p.Price != want[0] || p.Currency != want[1] {
			t.Errorf("%.60s…: got %q %q, want %q %q", doc, p.Price, p.Currency, want[0], want[1])
		}
	}
}

func TestNormalizePrice(t *testing.T) {
	for in, want := range map[string]string{
		"$1,299.00": "1299.00", "1.299,00 €": "1299.00", "12,50": "12.50", "1,299": "1299",
		" 49.9 ": "49.9", "free": "", "1.299.000": "1299000",
	} {
		if got := NormalizePrice(in); got != want {
			t.Errorf("NormalizePrice(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFetchImageDoesNotAcceptAVIF(t *testing.T) {
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		w.Write([]byte("image"))
	}))
	defer srv.Close()

	if _, err := NewClient(true).FetchImage(context.Background(), srv.URL, 1<<20); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(accept, "avif") || strings.Contains(accept, "*") {
		t.Errorf("Accept %q allows a format Cork can't decode", accept)
	}
}
