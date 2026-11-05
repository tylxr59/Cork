package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tylxr59/cork/internal/imaging"
	"github.com/tylxr59/cork/internal/scrape"
)

type Pin struct {
	ID          string    `json:"id"`
	BoardID     string    `json:"board_id"`
	BoardName   string    `json:"board_name"`
	Owner       string    `json:"owner"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	SourceURL   string    `json:"source_url"`
	Price       string    `json:"price"`
	Currency    string    `json:"currency"`
	ImageURL    string    `json:"image_url"`
	ThumbURL    string    `json:"thumb_url"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CanEdit     bool      `json:"can_edit"`
}

func imageURL(key string) string { return "/media/" + key + "/full" }
func thumbURL(key string) string { return "/media/" + key + "/thumb" }

const pinCols = `p.id, p.board_id, b.name, u.username, p.user_id, b.is_public, p.title, p.description, p.source_url,
	p.price, p.currency, p.image_key, p.width, p.height, p.color, p.created_at, p.updated_at`

type pinRow struct {
	Pin
	userID   string
	isPublic bool
}

func scanPin(sc interface{ Scan(...any) error }) (*pinRow, error) {
	var p pinRow
	var key string
	var pub int
	var created, updated int64
	if err := sc.Scan(&p.ID, &p.BoardID, &p.BoardName, &p.Owner, &p.userID, &pub, &p.Title, &p.Description, &p.SourceURL,
		&p.Price, &p.Currency, &key, &p.Width, &p.Height, &p.Color, &created, &updated); err != nil {
		return nil, err
	}
	p.isPublic = pub == 1
	p.ImageURL, p.ThumbURL = imageURL(key), thumbURL(key)
	p.CreatedAt, p.UpdatedAt = msTime(created), msTime(updated)
	return &p, nil
}

func (s *Server) queryPins(where string, args ...any) ([]Pin, error) {
	rows, err := s.db.Query(`SELECT `+pinCols+` FROM pins p JOIN boards b ON b.id = p.board_id JOIN users u ON u.id = p.user_id `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pins := []Pin{}
	for rows.Next() {
		p, err := scanPin(rows)
		if err != nil {
			return nil, err
		}
		pins = append(pins, p.Pin)
	}
	return pins, rows.Err()
}

func (s *Server) loadPin(id string) (*pinRow, error) {
	return scanPin(s.db.QueryRow(`SELECT `+pinCols+` FROM pins p JOIN boards b ON b.id = p.board_id JOIN users u ON u.id = p.user_id WHERE p.id = ?`, id))
}

func (s *Server) ownedPin(w http.ResponseWriter, id string, u *User) *pinRow {
	p, err := s.loadPin(id)
	if err != nil || p.userID != u.ID {
		if err == nil || isNoRows(err) {
			writeError(w, http.StatusNotFound, "pin not found")
		} else {
			serverError(w, err)
		}
		return nil
	}
	p.CanEdit = true
	return p
}

// GET /api/pins?q=&limit=&offset= — the signed-in user's pins, newest first.
func (s *Server) handleListPins(w http.ResponseWriter, r *http.Request, u *User) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 60
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	offset = max(0, offset)

	where := `WHERE p.user_id = ?`
	args := []any{u.ID}
	if term := strings.TrimSpace(q.Get("q")); term != "" {
		for _, word := range strings.Fields(term) {
			like := "%" + escapeLike(word) + "%"
			where += ` AND (p.title LIKE ? ESCAPE '\' OR p.description LIKE ? ESCAPE '\' OR p.source_url LIKE ? ESCAPE '\' OR b.name LIKE ? ESCAPE '\')`
			args = append(args, like, like, like, like)
		}
	}
	args = append(args, limit+1, offset)
	pins, err := s.queryPins(where+` ORDER BY p.created_at DESC, p.id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		serverError(w, err)
		return
	}
	more := len(pins) > limit
	if more {
		pins = pins[:limit]
	}
	for i := range pins {
		pins[i].CanEdit = true
	}
	writeJSON(w, http.StatusOK, map[string]any{"pins": pins, "has_more": more})
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (s *Server) handleGetPin(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	p, err := s.loadPin(r.PathValue("id"))
	if err != nil {
		if isNoRows(err) {
			writeError(w, http.StatusNotFound, "pin not found")
		} else {
			serverError(w, err)
		}
		return
	}
	owner := u != nil && u.ID == p.userID
	if !owner && !p.isPublic {
		writeError(w, http.StatusNotFound, "pin not found")
		return
	}
	p.CanEdit = owner
	writeJSON(w, http.StatusOK, p.Pin)
}

type pinFields struct {
	BoardID     *string `json:"board_id"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	SourceURL   *string `json:"source_url"`
	Price       *string `json:"price"`
	Currency    *string `json:"currency"`
}

var (
	currencyRE = regexp.MustCompile(`^[A-Za-z]{3}$`)
	// a currency symbol, for when the code isn't knowable (a bare "kr" could be SEK, NOK or DKK)
	currencySymbolRE = regexp.MustCompile(`^[^\s\d<>"'&]{1,4}$`)
)

func (f *pinFields) validate() string {
	trim := func(p *string, maxLen int, name string) string {
		if p == nil {
			return ""
		}
		*p = strings.TrimSpace(*p)
		if len(*p) > maxLen {
			return name + " is too long"
		}
		return ""
	}
	for _, msg := range []string{
		trim(f.Title, 300, "title"),
		trim(f.Description, 10000, "description"),
		trim(f.SourceURL, 4000, "link"),
		trim(f.Price, 32, "price"),
		trim(f.Currency, 8, "currency"),
	} {
		if msg != "" {
			return msg
		}
	}
	if f.Price != nil && *f.Price != "" {
		if *f.Price = scrape.NormalizePrice(*f.Price); *f.Price == "" {
			return "price must be a number, like 24.99"
		}
	}
	if f.Currency != nil {
		switch c := *f.Currency; {
		case c == "":
		case currencyRE.MatchString(c):
			*f.Currency = strings.ToUpper(c)
		case !currencySymbolRE.MatchString(c):
			return "currency must be a code like USD or a symbol like $"
		}
	}
	if f.SourceURL != nil && *f.SourceURL != "" {
		u, err := url.Parse(*f.SourceURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "link must be a full http:// or https:// URL"
		}
	}
	return ""
}

// POST /api/pins (multipart): board_id, title, description, source_url and
// either an "image" file or an "image_url" to download.
func (s *Server) handleCreatePin(w http.ResponseWriter, r *http.Request, u *User) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "upload too large or malformed")
		return
	}
	defer r.MultipartForm.RemoveAll()

	str := func(k string) *string { v := r.FormValue(k); return &v }
	f := pinFields{BoardID: str("board_id"), Title: str("title"), Description: str("description"), SourceURL: str("source_url"),
		Price: str("price"), Currency: str("currency")}
	if msg := f.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	board := s.ownedBoard(w, *f.BoardID, u)
	if board == nil {
		return
	}

	data, status, msg := s.readImageInput(r)
	if msg != "" {
		writeError(w, status, msg)
		return
	}
	img, err := imaging.Process(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	key := newID(20)
	if err := s.saveImage(key, img, data); err != nil {
		serverError(w, err)
		return
	}

	id, now := newID(12), nowMS()
	_, err = s.db.Exec(`INSERT INTO pins (id, board_id, user_id, title, description, source_url, price, currency, image_key, image_ext,
			width, height, color, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, (SELECT COALESCE(MIN(position), 0) - 1 FROM pins WHERE board_id = ?), ?, ?)`,
		id, board.ID, u.ID, *f.Title, *f.Description, *f.SourceURL, *f.Price, *f.Currency, key, img.Ext,
		img.Width, img.Height, img.Color, board.ID, now, now)
	if err != nil {
		s.removeImages([]imageRef{{key, img.Ext}})
		serverError(w, err)
		return
	}
	s.db.Exec(`UPDATE boards SET updated_at = ? WHERE id = ?`, now, board.ID)
	p, err := s.loadPin(id)
	if err != nil {
		serverError(w, err)
		return
	}
	p.CanEdit = true
	writeJSON(w, http.StatusCreated, p.Pin)
}

// readImageInput returns the image bytes from a parsed multipart form.
func (s *Server) readImageInput(r *http.Request) ([]byte, int, string) {
	if file, hdr, err := r.FormFile("image"); err == nil {
		defer file.Close()
		if hdr.Size > s.cfg.MaxUploadBytes {
			return nil, http.StatusRequestEntityTooLarge, "image is too large"
		}
		data, err := io.ReadAll(io.LimitReader(file, s.cfg.MaxUploadBytes+1))
		if err != nil {
			return nil, http.StatusBadRequest, "could not read upload"
		}
		return data, 0, ""
	}
	if raw := strings.TrimSpace(r.FormValue("image_url")); raw != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		data, err := s.fetch.FetchImage(ctx, raw, s.cfg.MaxUploadBytes)
		if err != nil {
			return nil, http.StatusBadRequest, err.Error()
		}
		return data, 0, ""
	}
	return nil, http.StatusBadRequest, "add an image to pin"
}

func (s *Server) handleUpdatePin(w http.ResponseWriter, r *http.Request, u *User) {
	p := s.ownedPin(w, r.PathValue("id"), u)
	if p == nil {
		return
	}
	var f pinFields
	if !readJSON(w, r, &f) {
		return
	}
	if msg := f.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if f.Title != nil {
		p.Title = *f.Title
	}
	if f.Description != nil {
		p.Description = *f.Description
	}
	if f.SourceURL != nil {
		p.SourceURL = *f.SourceURL
	}
	if f.Price != nil {
		p.Price = *f.Price
	}
	if f.Currency != nil {
		p.Currency = *f.Currency
	}
	moving := f.BoardID != nil && *f.BoardID != p.BoardID
	// check before opening the transaction: the pool has a single connection
	if moving && s.ownedBoard(w, *f.BoardID, u) == nil {
		return
	}
	now := nowMS()
	tx, err := s.db.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	if moving {
		// moved pins land at the top of their new board
		if _, err := tx.Exec(`UPDATE pins SET board_id = ?, position = (SELECT COALESCE(MIN(position), 0) - 1 FROM pins WHERE board_id = ?) WHERE id = ?`,
			*f.BoardID, *f.BoardID, p.ID); err != nil {
			serverError(w, err)
			return
		}
	}
	if _, err := tx.Exec(`UPDATE pins SET title = ?, description = ?, source_url = ?, price = ?, currency = ?, updated_at = ? WHERE id = ?`,
		p.Title, p.Description, p.SourceURL, p.Price, p.Currency, now, p.ID); err != nil {
		serverError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	updated, err := s.loadPin(p.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	updated.CanEdit = true
	writeJSON(w, http.StatusOK, updated.Pin)
}

// PUT /api/pins/{id}/image (multipart "image" or "image_url") swaps a pin's picture.
func (s *Server) handleReplacePinImage(w http.ResponseWriter, r *http.Request, u *User) {
	p := s.ownedPin(w, r.PathValue("id"), u)
	if p == nil {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "upload too large or malformed")
		return
	}
	defer r.MultipartForm.RemoveAll()
	data, status, msg := s.readImageInput(r)
	if msg != "" {
		writeError(w, status, msg)
		return
	}
	img, err := imaging.Process(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	old, err := s.imageKeys(`SELECT image_key, image_ext FROM pins WHERE id = ?`, p.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	key := newID(20)
	if err := s.saveImage(key, img, data); err != nil {
		serverError(w, err)
		return
	}
	if _, err := s.db.Exec(`UPDATE pins SET image_key = ?, image_ext = ?, width = ?, height = ?, color = ?, updated_at = ? WHERE id = ?`,
		key, img.Ext, img.Width, img.Height, img.Color, nowMS(), p.ID); err != nil {
		s.removeImages([]imageRef{{key, img.Ext}})
		serverError(w, err)
		return
	}
	s.removeImages(old)
	updated, err := s.loadPin(p.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	updated.CanEdit = true
	writeJSON(w, http.StatusOK, updated.Pin)
}

func (s *Server) handleDeletePin(w http.ResponseWriter, r *http.Request, u *User) {
	p := s.ownedPin(w, r.PathValue("id"), u)
	if p == nil {
		return
	}
	keys, err := s.imageKeys(`SELECT image_key, image_ext FROM pins WHERE id = ?`, p.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	if _, err := s.db.Exec(`DELETE FROM pins WHERE id = ?`, p.ID); err != nil {
		serverError(w, err)
		return
	}
	s.removeImages(keys)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleScrape(w http.ResponseWriter, r *http.Request, u *User) {
	var in struct {
		URL string `json:"url"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	page, err := s.fetch.FetchPage(ctx, in.URL)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, scrape.ErrBlockedAddress) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// --- image storage ---

type imageRef struct{ key, ext string }

func (s *Server) imagePaths(key, ext string) (full, thumb string) {
	dir := filepath.Join(s.media, key[:2])
	return filepath.Join(dir, key+ext), filepath.Join(dir, key+"_thumb")
}

func (s *Server) saveImage(key string, img *imaging.Result, data []byte) error {
	full, thumb := s.imagePaths(key, img.Ext)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	if err := writeFileAtomic(full, data); err != nil {
		return err
	}
	if img.Thumb != nil {
		if err := writeFileAtomic(thumb, img.Thumb); err != nil {
			os.Remove(full)
			return err
		}
	}
	return nil
}

func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Server) imageKeys(query string, args ...any) ([]imageRef, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var refs []imageRef
	for rows.Next() {
		var ref imageRef
		if err := rows.Scan(&ref.key, &ref.ext); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

func (s *Server) removeImages(refs []imageRef) {
	for _, ref := range refs {
		full, thumb := s.imagePaths(ref.key, ref.ext)
		for _, p := range []string{full, thumb} {
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				slog.Warn("could not remove image", "path", p, "err", err)
			}
		}
	}
}

var extTypes = map[string]string{".jpg": "image/jpeg", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp"}

// GET /media/{key}/{full|thumb}
func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	key, variant := r.PathValue("key"), r.PathValue("variant")
	if len(key) != 20 || (variant != "full" && variant != "thumb") {
		http.NotFound(w, r)
		return
	}
	var ext, ownerID string
	var pub int
	err := s.db.QueryRow(`SELECT p.image_ext, p.user_id, b.is_public FROM pins p JOIN boards b ON b.id = p.board_id WHERE p.image_key = ?`, key).
		Scan(&ext, &ownerID, &pub)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := userFrom(r)
	if pub != 1 && (u == nil || u.ID != ownerID) {
		http.NotFound(w, r)
		return
	}
	full, thumb := s.imagePaths(key, ext)
	path, ctype := full, extTypes[ext]
	if variant == "thumb" {
		if _, err := os.Stat(thumb); err == nil {
			path, ctype = thumb, ""
		}
	}
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if ctype == "" {
		head := make([]byte, 512)
		n, _ := io.ReadFull(f, head)
		ctype = http.DetectContentType(head[:n])
		f.Seek(0, io.SeekStart)
	}
	w.Header().Set("Content-Type", ctype)
	// keys are immutable: replacing an image creates a new key
	if pub == 1 {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	}
	http.ServeContent(w, r, "", st.ModTime(), f)
}
