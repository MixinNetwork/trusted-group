package main

// FIXME do rate limit based on IP
// POST /users
// POST /extra

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/dimfeld/httptreemux/v5"
	"github.com/gofrs/uuid"
	"github.com/unrolled/render"
)

var (
	proxy *Proxy
	store *Storage
)

func StartHTTP(p *Proxy, s *Storage) error {
	proxy, store = p, s
	router := httptreemux.New()
	router.GET("/", index)
	router.POST("/extra", encodeExtra)
	router.POST("/users", createUser)
	router.GET("/collectibles/:collection/:id", getTokenMeta)
	handler := handleCORS(router)
	handler = handleLog(handler)
	return http.ListenAndServe(fmt.Sprintf(":%d", HTTPPort), handler)
}

// TODO make a bridge web interface
func index(w http.ResponseWriter, r *http.Request, params map[string]string) {
	render.New().JSON(w, http.StatusOK, map[string]any{
		"code":       "https://github.com/MixinNetwork/trusted-group/tree/master/mvm/quorum/bridge",
		"process":    MVMRegistryId,
		"registry":   "https://scan.mvm.dev/address/" + MVMRegistryContract,
		"bridge":     "https://scan.mvm.dev/address/" + MVMBridgeContract,
		"mirror":     "https://scan.mvm.dev/address/" + MVMMirrorContract,
		"withdrawal": "https://scan.mvm.dev/address/" + MVMWithdrawalContract,
		"storage":    "https://scan.mvm.dev/address/" + MVMStorageContract,
	})
}

func createUser(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var body struct {
		PublicKey string `json:"public_key"`
		Signature string `json:"signature"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}
	u, err := proxy.createUser(r.Context(), store, body.PublicKey, body.Signature)
	if err != nil {
		render.New().JSON(w, http.StatusInternalServerError, map[string]any{"error": err})
		return
	}
	render.New().JSON(w, http.StatusOK, map[string]any{"user": map[string]any{
		"user_id":    u.UserID,
		"session_id": u.SessionID,
		"full_name":  u.FullName,
		"created_at": u.CreatedAt,
		"key": map[string]any{
			"client_id":   u.Key.ClientID,
			"session_id":  u.Key.SessionID,
			"private_key": u.Key.PrivateKey,
		},
		"contract": u.Contract,
	}})
}

func encodeExtra(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var body Action
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}
	extra, err := encodeActionAsExtra(&body)
	if err != nil {
		render.New().JSON(w, http.StatusInternalServerError, map[string]any{"error": err})
		return
	}
	render.New().JSON(w, http.StatusOK, map[string]any{"extra": extra})
}

func getTokenMeta(w http.ResponseWriter, r *http.Request, params map[string]string) {
	cb, err := hex.DecodeString(params["collection"])
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}
	collection := uuid.FromBytesOrNil(cb).String()

	tdt := fmt.Sprintf("https://thetrident.one/api/%s/%s", collection, params["id"])
	resp, err := http.Get(tdt)
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}
	defer resp.Body.Close()

	var body map[string]any
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}
	render.New().JSON(w, http.StatusOK, body)
}

// TODO may consider a whitelist in the case of Ethereum scams
func handleCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			handler.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type,X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,DELETE")
		w.Header().Set("Access-Control-Max-Age", "600")
		if r.Method == "OPTIONS" {
			render.New().JSON(w, http.StatusOK, map[string]any{})
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

func handleLog(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Verbosef("ServeHTTP(%v)", *r)
		handler.ServeHTTP(w, r)
	})
}
