package main

// FIXME do rate limit based on IP
// POST /users
// GET /users/:id

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dimfeld/httptreemux"
	"github.com/unrolled/render"
)

var (
	proxy *Proxy
	store *Storage
)

func StartHTTP(p *Proxy, s *Storage) error {
	proxy, store = p, s
	router := httptreemux.New()
	router.POST("/extra", encodeExtra)
	router.POST("/users", createUser)
	return http.ListenAndServe(fmt.Sprintf(":%d", HTTPPort), router)
}

func createUser(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var body struct {
		PublicKey string `json:"public_key"`
		Signature string `json:"signature"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]interface{}{"error": err})
		return
	}
	user, err := proxy.createUser(r.Context(), store, body.PublicKey, body.Signature)
	if err != nil {
		render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err})
		return
	}
	render.New().JSON(w, http.StatusOK, map[string]interface{}{"user": user})
}

func encodeExtra(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var body Action
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]interface{}{"error": err})
		return
	}
	extra, err := encodeActionAsExtra(&body)
	if err != nil {
		render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err})
		return
	}
	render.New().JSON(w, http.StatusOK, map[string]interface{}{"extra": extra})
}
