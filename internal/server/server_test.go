package server

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

type client struct {
	t    *testing.T
	base string
	http *http.Client
}

func newClient(t *testing.T, base string) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, base: base, http: &http.Client{Jar: jar}}
}

func (c *client) do(method, path string, body any, out any) int {
	c.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, c.base+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cork", "1")
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

func pngBytes(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.NRGBA{200, 100, 50, 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func (c *client) createPin(boardID, title string) Pin {
	c.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("board_id", boardID)
	mw.WriteField("title", title)
	mw.WriteField("source_url", "https://example.com/"+title)
	mw.WriteField("price", "$1,299.50")
	mw.WriteField("currency", "usd")
	fw, _ := mw.CreateFormFile("image", "x.png")
	fw.Write(pngBytes(800, 400))
	mw.Close()
	req, _ := http.NewRequest("POST", c.base+"/api/pins", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Cork", "1")
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	var p Pin
	json.NewDecoder(resp.Body).Decode(&p)
	if resp.StatusCode != http.StatusCreated {
		c.t.Fatalf("create pin: status %d", resp.StatusCode)
	}
	return p
}

func TestFlow(t *testing.T) {
	srv, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	alice := newClient(t, ts.URL)
	var status map[string]any
	alice.do("GET", "/api/status", nil, &status)
	if status["needs_setup"] != true {
		t.Fatalf("expected needs_setup, got %v", status)
	}

	var me User
	if code := alice.do("POST", "/api/auth/register", credentials{"alice", "password123"}, &me); code != 201 || !me.IsAdmin {
		t.Fatalf("first user should be admin: %d %+v", code, me)
	}

	bob := newClient(t, ts.URL)
	if code := bob.do("POST", "/api/auth/register", credentials{"bob", "password123"}, nil); code != 403 {
		t.Fatalf("registration should be closed, got %d", code)
	}
	if code := alice.do("POST", "/api/admin/users", map[string]any{"username": "bob", "password": "password123"}, nil); code != 201 {
		t.Fatalf("admin create user: %d", code)
	}
	if code := bob.do("POST", "/api/auth/login", credentials{"bob", "password123"}, nil); code != 200 {
		t.Fatalf("bob login: %d", code)
	}
	if code := bob.do("GET", "/api/admin/users", nil, nil); code != 403 {
		t.Fatalf("non-admin got admin endpoint: %d", code)
	}

	// CSRF header required
	req, _ := http.NewRequest("POST", ts.URL+"/api/boards", bytes.NewBufferString(`{"name":"x"}`))
	if resp, _ := alice.http.Do(req); resp.StatusCode != 403 {
		t.Fatalf("missing X-Cork should be rejected, got %d", resp.StatusCode)
	}

	var board, other Board
	alice.do("POST", "/api/boards", map[string]any{"name": "Kitchens"}, &board)
	alice.do("POST", "/api/boards", map[string]any{"name": "Gardens"}, &other)
	var boards []Board
	alice.do("GET", "/api/boards", nil, &boards)
	if len(boards) != 2 || boards[0].Name != "Gardens" {
		t.Fatalf("expected newest board first: %+v", boards)
	}

	p1 := alice.createPin(board.ID, "one")
	p2 := alice.createPin(board.ID, "two")
	p3 := alice.createPin(board.ID, "three")
	if p1.Price != "1299.50" || p1.Currency != "USD" {
		t.Fatalf("price not normalized: %q %q", p1.Price, p1.Currency)
	}
	if p1.Width != 800 || p1.Height != 400 || p1.Color != "#c86432" {
		t.Fatalf("unexpected image metadata: %+v", p1)
	}

	var got struct {
		Board Board `json:"board"`
		Pins  []Pin `json:"pins"`
	}
	alice.do("GET", "/api/boards/"+board.ID, nil, &got)
	if len(got.Pins) != 3 || got.Pins[0].ID != p3.ID {
		t.Fatalf("new pins should be first: %+v", got.Pins)
	}

	alice.do("PUT", "/api/boards/"+board.ID+"/order", orderInput{IDs: []string{p1.ID, p3.ID, p2.ID}}, nil)
	alice.do("GET", "/api/boards/"+board.ID, nil, &got)
	if got.Pins[0].ID != p1.ID || got.Pins[1].ID != p3.ID || got.Pins[2].ID != p2.ID {
		t.Fatalf("reorder not applied")
	}

	// bob can't see alice's private board, pins or images
	if code := bob.do("GET", "/api/boards/"+board.ID, nil, nil); code != 404 {
		t.Fatalf("private board visible to other user: %d", code)
	}
	if resp, _ := bob.http.Get(ts.URL + p1.ThumbURL); resp.StatusCode != 404 {
		t.Fatalf("private image visible to other user: %d", resp.StatusCode)
	}
	if code := bob.do("PUT", "/api/boards/"+board.ID+"/order", orderInput{IDs: []string{p2.ID}}, nil); code != 404 {
		t.Fatalf("bob reordered alice's board: %d", code)
	}
	if code := bob.do("DELETE", "/api/pins/"+p1.ID, nil, nil); code != 404 {
		t.Fatalf("bob deleted alice's pin: %d", code)
	}

	// making the board public exposes it read-only
	alice.do("PATCH", "/api/boards/"+board.ID, map[string]any{"is_public": true}, nil)
	anon := newClient(t, ts.URL)
	if code := anon.do("GET", "/api/boards/"+board.ID, nil, &got); code != 200 || got.Board.CanEdit {
		t.Fatalf("public board: %d can_edit=%v", code, got.Board.CanEdit)
	}
	resp, _ := anon.http.Get(ts.URL + p1.ThumbURL)
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("public thumb: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}

	// move a pin to the other board
	var moved Pin
	alice.do("PATCH", "/api/pins/"+p2.ID, map[string]any{"board_id": other.ID, "description": "moved"}, &moved)
	if moved.BoardID != other.ID || moved.Description != "moved" {
		t.Fatalf("move failed: %+v", moved)
	}

	var search struct {
		Pins []Pin `json:"pins"`
	}
	alice.do("GET", "/api/pins?q=thr", nil, &search)
	if len(search.Pins) != 1 || search.Pins[0].ID != p3.ID {
		t.Fatalf("search: %+v", search.Pins)
	}

	if code := alice.do("DELETE", "/api/boards/"+board.ID, nil, nil); code != 200 {
		t.Fatalf("delete board: %d", code)
	}
	if resp, _ := alice.http.Get(ts.URL + p1.ImageURL); resp.StatusCode != 404 {
		t.Fatalf("deleted pin image still served: %d", resp.StatusCode)
	}
}

func TestCurrencyValidation(t *testing.T) {
	for in, want := range map[string]string{"usd": "USD", "EUR": "EUR", "$": "$", "kr": "kr", "CA$": "CA$", "": ""} {
		c := in
		f := pinFields{Currency: &c}
		if msg := f.validate(); msg != "" || c != want {
			t.Errorf("%q: got %q (%s), want %q", in, c, msg, want)
		}
	}
	for _, in := range []string{"dollars", "12", "<b>", "US D"} {
		c := in
		f := pinFields{Currency: &c}
		if f.validate() == "" {
			t.Errorf("%q: expected an error", in)
		}
	}
}
