// Package scrape fetches remote pages and images on behalf of users, and pulls
// pin metadata (title, description, candidate images) out of HTML.
package scrape

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html"
)

const userAgent = "Mozilla/5.0 (compatible; Cork/1.0; +https://github.com/tylxr59/Cork)"

var ErrBlockedAddress = errors.New("refusing to fetch a private or local network address")

// Client fetches remote URLs. Unless AllowPrivate is set it refuses to connect
// to loopback, private, link-local and other non-public addresses, so users
// can't use Cork to probe the network it runs on.
type Client struct {
	http *http.Client
}

func NewClient(allowPrivate bool) *Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if !allowPrivate {
		dialer.Control = func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || !isPublic(ip) {
				return ErrBlockedAddress
			}
			return nil
		}
	}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       60 * time.Second,
	}
	return &Client{http: &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("unsupported redirect scheme")
			}
			return nil
		},
	}}
}

func isPublic(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() ||
		// carrier-grade NAT 100.64.0.0/10 (also used by Tailscale)
		(ip.To4() != nil && ip.To4()[0] == 100 && ip.To4()[1]&0xC0 == 64))
}

// ParseURL validates a user-supplied http(s) URL.
func ParseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("enter a full http:// or https:// URL")
	}
	return u, nil
}

func (c *Client) get(ctx context.Context, u *url.URL, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", accept)
	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, ErrBlockedAddress) {
			return nil, ErrBlockedAddress
		}
		return nil, fmt.Errorf("could not fetch %s", u.Host)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		resp.Body.Close()
		return nil, fmt.Errorf("%s responded with %s", u.Host, resp.Status)
	}
	return resp, nil
}

// FetchImage downloads an image, returning at most maxBytes.
func (c *Client) FetchImage(ctx context.Context, rawURL string, maxBytes int64) ([]byte, error) {
	u, err := ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	resp, err := c.get(ctx, u, "image/avif;q=0.1,image/webp,image/png,image/jpeg,image/gif,image/*;q=0.8")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("could not download image: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("image is larger than %d MB", maxBytes>>20)
	}
	return data, nil
}

type Page struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	SiteName    string   `json:"site_name"`
	Price       string   `json:"price"`
	Currency    string   `json:"currency"`
	Images      []string `json:"images"`
}

// FetchPage loads a page and extracts pin metadata. If the URL is itself an
// image, it is returned as the only candidate.
func (c *Client) FetchPage(ctx context.Context, rawURL string) (*Page, error) {
	u, err := ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	resp, err := c.get(ctx, u, "text/html,application/xhtml+xml,image/*;q=0.9,*/*;q=0.5")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	final := resp.Request.URL
	mt, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if strings.HasPrefix(mt, "image/") {
		return &Page{URL: final.String(), Images: []string{final.String()}}, nil
	}
	if mt != "" && mt != "text/html" && mt != "application/xhtml+xml" {
		return nil, fmt.Errorf("that URL is not a web page or image (%s)", mt)
	}
	p := parseHTML(io.LimitReader(resp.Body, 4<<20), final)
	p.URL = u.String()
	return p, nil
}

func parseHTML(r io.Reader, base *url.URL) *Page {
	z := html.NewTokenizer(r)
	meta := map[string]string{}
	var title strings.Builder
	var inTitle, inLD bool
	var ldBlobs []string
	var itemPrice, itemCurrency string
	var primary, imgs []string

	resolve := func(ref string) string {
		ref = strings.TrimSpace(ref)
		if ref == "" || strings.HasPrefix(ref, "data:") {
			return ""
		}
		ru, err := base.Parse(ref)
		if err != nil || (ru.Scheme != "http" && ru.Scheme != "https") {
			return ""
		}
		if strings.HasSuffix(strings.ToLower(ru.Path), ".svg") {
			return "" // not decodable server-side
		}
		return ru.String()
	}

loop:
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			break loop
		case html.TextToken:
			if inTitle {
				title.Write(z.Text())
			}
			if inLD && len(ldBlobs) < 20 {
				ldBlobs = append(ldBlobs, string(z.Text()))
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			switch string(name) {
			case "title":
				inTitle = false
			case "script":
				inLD = false
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			attrs := map[string]string{}
			for hasAttr {
				var k, v []byte
				k, v, hasAttr = z.TagAttr()
				attrs[strings.ToLower(string(k))] = string(v)
			}
			// microdata: <meta itemprop="price" content="12.99"> or <span itemprop="price">
			switch strings.ToLower(attrs["itemprop"]) {
			case "price":
				if itemPrice == "" {
					itemPrice = attrs["content"]
				}
			case "pricecurrency":
				if itemCurrency == "" {
					itemCurrency = attrs["content"]
				}
			}
			switch string(name) {
			case "script":
				inLD = tt == html.StartTagToken && strings.Contains(strings.ToLower(attrs["type"]), "ld+json")
			case "title":
				inTitle = tt == html.StartTagToken && title.Len() == 0
			case "base":
				if h := attrs["href"]; h != "" {
					if bu, err := base.Parse(h); err == nil {
						base = bu
					}
				}
			case "meta":
				key := strings.ToLower(attrs["property"])
				if key == "" {
					key = strings.ToLower(attrs["name"])
				}
				if key != "" && attrs["content"] != "" {
					if _, seen := meta[key]; !seen {
						meta[key] = attrs["content"]
					}
					if key == "og:image" || key == "og:image:url" || key == "og:image:secure_url" || key == "twitter:image" || key == "twitter:image:src" {
						primary = append(primary, resolve(attrs["content"]))
					}
				}
			case "link":
				if strings.Contains(strings.ToLower(attrs["rel"]), "image_src") {
					primary = append(primary, resolve(attrs["href"]))
				}
			case "img":
				if len(imgs) < 60 {
					src := attrs["src"]
					for _, alt := range []string{"data-src", "data-lazy-src", "data-original"} {
						if attrs[alt] != "" {
							src = attrs[alt]
							break
						}
					}
					if src == "" || strings.HasPrefix(src, "data:") {
						src = firstSrcset(attrs["srcset"])
					}
					// skip obvious tracking pixels / icons
					if attrs["width"] == "1" || attrs["height"] == "1" {
						continue
					}
					imgs = append(imgs, resolve(src))
				}
			}
		}
	}

	pick := func(keys ...string) string {
		for _, k := range keys {
			if v := strings.TrimSpace(meta[k]); v != "" {
				return v
			}
		}
		return ""
	}
	p := &Page{
		Title:       pick("og:title", "twitter:title"),
		Description: pick("og:description", "twitter:description", "description"),
		SiteName:    pick("og:site_name", "application-name"),
	}
	if p.Title == "" {
		p.Title = strings.Join(strings.Fields(title.String()), " ")
	}

	// price: structured data first, then Open Graph / product meta, then microdata
	for _, blob := range ldBlobs {
		if pr, cur := priceFromJSONLD(blob); pr != "" {
			p.Price, p.Currency = pr, cur
			break
		}
	}
	if p.Price == "" {
		p.Price = NormalizePrice(pick("product:price:amount", "og:price:amount", "product:sale_price:amount"))
		p.Currency = NormalizeCurrency(pick("product:price:currency", "og:price:currency", "product:sale_price:currency"))
	}
	if p.Price == "" {
		p.Price = NormalizePrice(itemPrice)
		p.Currency = NormalizeCurrency(itemCurrency)
	}
	if p.Price == "" {
		p.Currency = ""
	}

	seen := map[string]bool{}
	for _, s := range append(primary, imgs...) {
		if s != "" && !seen[s] && len(p.Images) < 30 {
			seen[s] = true
			p.Images = append(p.Images, s)
		}
	}
	if p.Images == nil {
		p.Images = []string{}
	}
	return p
}

// firstSrcset returns the largest candidate in a srcset attribute.
func firstSrcset(srcset string) string {
	best, bestW := "", -1
	for _, part := range strings.Split(srcset, ",") {
		f := strings.Fields(strings.TrimSpace(part))
		if len(f) == 0 {
			continue
		}
		w := 0
		if len(f) > 1 {
			fmt.Sscanf(f[1], "%dw", &w)
		}
		if w > bestW {
			best, bestW = f[0], w
		}
	}
	return best
}
