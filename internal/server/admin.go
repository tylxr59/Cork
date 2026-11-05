package server

import (
	"errors"
	"net/http"
	"strings"
)

type adminUser struct {
	User
	BoardCount int `json:"board_count"`
	PinCount   int `json:"pin_count"`
}

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request, _ *User) {
	rows, err := s.db.Query(`SELECT u.id, u.username, u.is_admin, u.created_at,
			(SELECT COUNT(*) FROM boards b WHERE b.user_id = u.id),
			(SELECT COUNT(*) FROM pins p WHERE p.user_id = u.id)
		FROM users u ORDER BY u.created_at`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	users := []adminUser{}
	for rows.Next() {
		var u adminUser
		var isAdmin int
		var created int64
		if err := rows.Scan(&u.ID, &u.Username, &isAdmin, &created, &u.BoardCount, &u.PinCount); err != nil {
			serverError(w, err)
			return
		}
		u.IsAdmin = isAdmin == 1
		u.CreatedAt = msTime(created)
		users = append(users, u)
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request, _ *User) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"is_admin"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if msg := validateCredentials(in.Username, in.Password); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	u, err := s.createUser(in.Username, in.Password, in.IsAdmin)
	if err != nil {
		if errors.Is(err, errUsernameTaken) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, adminUser{User: *u})
}

func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request, me *User) {
	id := r.PathValue("id")
	var in struct {
		IsAdmin  *bool   `json:"is_admin"`
		Password *string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	var username string
	if err := s.db.QueryRow(`SELECT username FROM users WHERE id = ?`, id).Scan(&username); err != nil {
		if isNoRows(err) {
			writeError(w, http.StatusNotFound, "user not found")
		} else {
			serverError(w, err)
		}
		return
	}
	if in.IsAdmin != nil {
		if id == me.ID && !*in.IsAdmin {
			writeError(w, http.StatusBadRequest, "you can't remove your own admin role")
			return
		}
		if _, err := s.db.Exec(`UPDATE users SET is_admin = ? WHERE id = ?`, boolInt(*in.IsAdmin), id); err != nil {
			serverError(w, err)
			return
		}
	}
	if in.Password != nil {
		if msg := validateCredentials(username, *in.Password); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		if err := s.setPassword(id, *in.Password); err != nil {
			serverError(w, err)
			return
		}
		if id != me.ID {
			s.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, id)
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request, me *User) {
	id := r.PathValue("id")
	if id == me.ID {
		writeError(w, http.StatusBadRequest, "you can't delete your own account")
		return
	}
	keys, err := s.imageKeys(`SELECT image_key, image_ext FROM pins WHERE user_id = ?`, id)
	if err != nil {
		serverError(w, err)
		return
	}
	res, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	s.removeImages(keys)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type settingsBody struct {
	RegistrationOpen bool `json:"registration_open"`
}

func (s *Server) handleAdminGetSettings(w http.ResponseWriter, r *http.Request, _ *User) {
	writeJSON(w, http.StatusOK, settingsBody{RegistrationOpen: s.registrationOpen()})
}

func (s *Server) handleAdminPutSettings(w http.ResponseWriter, r *http.Request, _ *User) {
	var in settingsBody
	if !readJSON(w, r, &in) {
		return
	}
	v := "false"
	if in.RegistrationOpen {
		v = "true"
	}
	if err := s.setSetting("registration_open", v); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, in)
}
