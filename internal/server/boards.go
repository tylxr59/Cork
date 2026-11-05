package server

import (
	"net/http"
	"strings"
	"time"
)

type Board struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"is_public"`
	PinCount    int       `json:"pin_count"`
	Covers      []Cover   `json:"covers"`
	Owner       string    `json:"owner"`
	CanEdit     bool      `json:"can_edit"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Cover struct {
	URL   string `json:"url"`
	Color string `json:"color"`
}

type boardRow struct {
	Board
	userID string
}

const boardCols = `b.id, b.user_id, u.username, b.name, b.description, b.is_public, b.created_at, b.updated_at,
	(SELECT COUNT(*) FROM pins p WHERE p.board_id = b.id)`

func scanBoard(sc interface{ Scan(...any) error }) (*boardRow, error) {
	var b boardRow
	var pub int
	var created, updated int64
	if err := sc.Scan(&b.ID, &b.userID, &b.Owner, &b.Name, &b.Description, &pub, &created, &updated, &b.PinCount); err != nil {
		return nil, err
	}
	b.IsPublic = pub == 1
	b.CreatedAt, b.UpdatedAt = msTime(created), msTime(updated)
	b.Covers = []Cover{}
	return &b, nil
}

func (s *Server) loadBoard(id string) (*boardRow, error) {
	return scanBoard(s.db.QueryRow(`SELECT `+boardCols+` FROM boards b JOIN users u ON u.id = b.user_id WHERE b.id = ?`, id))
}

// ownedBoard loads a board and writes a 404 unless u owns it.
func (s *Server) ownedBoard(w http.ResponseWriter, id string, u *User) *boardRow {
	b, err := s.loadBoard(id)
	if err != nil {
		if isNoRows(err) {
			writeError(w, http.StatusNotFound, "board not found")
		} else {
			serverError(w, err)
		}
		return nil
	}
	if b.userID != u.ID {
		writeError(w, http.StatusNotFound, "board not found")
		return nil
	}
	b.CanEdit = true
	return b
}

func (s *Server) handleListBoards(w http.ResponseWriter, r *http.Request, u *User) {
	rows, err := s.db.Query(`SELECT `+boardCols+` FROM boards b JOIN users u ON u.id = b.user_id
		WHERE b.user_id = ? ORDER BY b.position, b.created_at`, u.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	boards := []*Board{}
	byID := map[string]*Board{}
	for rows.Next() {
		b, err := scanBoard(rows)
		if err != nil {
			serverError(w, err)
			return
		}
		b.CanEdit = true
		boards = append(boards, &b.Board)
		byID[b.ID] = &b.Board
	}
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	rows.Close()

	// up to 3 cover images per board, in board order
	covers, err := s.db.Query(`
		SELECT board_id, image_key, color FROM (
			SELECT p.board_id, p.image_key, p.color,
				ROW_NUMBER() OVER (PARTITION BY p.board_id ORDER BY p.position, p.created_at DESC) AS rn
			FROM pins p WHERE p.user_id = ?
		) WHERE rn <= 3 ORDER BY board_id, rn`, u.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	defer covers.Close()
	for covers.Next() {
		var boardID, key, color string
		if err := covers.Scan(&boardID, &key, &color); err != nil {
			serverError(w, err)
			return
		}
		if b := byID[boardID]; b != nil {
			b.Covers = append(b.Covers, Cover{URL: thumbURL(key), Color: color})
		}
	}
	writeJSON(w, http.StatusOK, boards)
}

type boardInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsPublic    *bool   `json:"is_public"`
}

func (in *boardInput) validate(creating bool) string {
	if in.Name != nil {
		*in.Name = strings.TrimSpace(*in.Name)
		if *in.Name == "" {
			return "board name is required"
		}
		if len(*in.Name) > 100 {
			return "board name is too long"
		}
	} else if creating {
		return "board name is required"
	}
	if in.Description != nil {
		*in.Description = strings.TrimSpace(*in.Description)
		if len(*in.Description) > 2000 {
			return "description is too long"
		}
	}
	return ""
}

func (s *Server) handleCreateBoard(w http.ResponseWriter, r *http.Request, u *User) {
	var in boardInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := in.validate(true); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	desc := ""
	if in.Description != nil {
		desc = *in.Description
	}
	pub := in.IsPublic != nil && *in.IsPublic
	id, now := newID(12), nowMS()
	// new boards go first
	_, err := s.db.Exec(`INSERT INTO boards (id, user_id, name, description, is_public, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, (SELECT COALESCE(MIN(position), 0) - 1 FROM boards WHERE user_id = ?), ?, ?)`,
		id, u.ID, *in.Name, desc, boolInt(pub), u.ID, now, now)
	if err != nil {
		serverError(w, err)
		return
	}
	b, err := s.loadBoard(id)
	if err != nil {
		serverError(w, err)
		return
	}
	b.CanEdit = true
	writeJSON(w, http.StatusCreated, b.Board)
}

func (s *Server) handleGetBoard(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	b, err := s.loadBoard(r.PathValue("id"))
	if err != nil {
		if isNoRows(err) {
			writeError(w, http.StatusNotFound, "board not found")
		} else {
			serverError(w, err)
		}
		return
	}
	owner := u != nil && u.ID == b.userID
	if !owner && !b.IsPublic {
		writeError(w, http.StatusNotFound, "board not found")
		return
	}
	b.CanEdit = owner
	pins, err := s.queryPins(`WHERE p.board_id = ? ORDER BY p.position, p.created_at DESC`, b.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"board": b.Board, "pins": pins})
}

func (s *Server) handleUpdateBoard(w http.ResponseWriter, r *http.Request, u *User) {
	b := s.ownedBoard(w, r.PathValue("id"), u)
	if b == nil {
		return
	}
	var in boardInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := in.validate(false); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if in.Name != nil {
		b.Name = *in.Name
	}
	if in.Description != nil {
		b.Description = *in.Description
	}
	if in.IsPublic != nil {
		b.IsPublic = *in.IsPublic
	}
	now := nowMS()
	if _, err := s.db.Exec(`UPDATE boards SET name = ?, description = ?, is_public = ?, updated_at = ? WHERE id = ?`,
		b.Name, b.Description, boolInt(b.IsPublic), now, b.ID); err != nil {
		serverError(w, err)
		return
	}
	b.UpdatedAt = msTime(now)
	writeJSON(w, http.StatusOK, b.Board)
}

func (s *Server) handleDeleteBoard(w http.ResponseWriter, r *http.Request, u *User) {
	b := s.ownedBoard(w, r.PathValue("id"), u)
	if b == nil {
		return
	}
	keys, err := s.imageKeys(`SELECT image_key, image_ext FROM pins WHERE board_id = ?`, b.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	if _, err := s.db.Exec(`DELETE FROM boards WHERE id = ?`, b.ID); err != nil {
		serverError(w, err)
		return
	}
	s.removeImages(keys)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type orderInput struct {
	IDs []string `json:"ids"`
}

func (s *Server) handleOrderBoards(w http.ResponseWriter, r *http.Request, u *User) {
	var in orderInput
	if !readJSON(w, r, &in) {
		return
	}
	if err := s.applyOrder(`UPDATE boards SET position = ? WHERE id = ? AND user_id = ?`, in.IDs, u.ID); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleOrderPins(w http.ResponseWriter, r *http.Request, u *User) {
	b := s.ownedBoard(w, r.PathValue("id"), u)
	if b == nil {
		return
	}
	var in orderInput
	if !readJSON(w, r, &in) {
		return
	}
	if err := s.applyOrder(`UPDATE pins SET position = ? WHERE id = ? AND board_id = ?`, in.IDs, b.ID); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// applyOrder sets position = index for each id, scoped by the query's third
// parameter so callers can only reorder rows they own.
func (s *Server) applyOrder(query string, ids []string, scope string) error {
	if len(ids) > 10000 {
		ids = ids[:10000]
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, id := range ids {
		if _, err := stmt.Exec(float64(i), id, scope); err != nil {
			return err
		}
	}
	return tx.Commit()
}
