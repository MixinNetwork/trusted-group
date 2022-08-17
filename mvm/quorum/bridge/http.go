package main

// FIXME do rate limit based on IP
// POST /users
// POST /extra

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/dimfeld/httptreemux"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gofrs/uuid"
	"github.com/gorilla/handlers"
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
	router.GET("/assets/:id", assetInfo)
	router.POST("/extra", encodeExtra)
	router.POST("/users", createUser)
	router.GET("/collectibles/:collection/:id", getTokenMeta)
	handler := handleCORS(router)
	handler = State(handler)
	handler = handlers.ProxyHeaders(handler)
	return http.ListenAndServe(fmt.Sprintf(":%d", HTTPPort), handler)
}

func State(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("INFO -- : Started %s '%s'", r.Method, r.URL)
		defer func() {
			log.Printf("INFO -- : Completed %s in %fms", r.Method, time.Now().Sub(start).Seconds())
		}()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err})
			return
		}
		if len(body) > 0 {
			log.Printf("INFO -- : Paremeters %s", string(body))
		}
		r.Body.Close()
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		handler.ServeHTTP(w, r)
	})
}

// TODO make a bridge web interface
func index(w http.ResponseWriter, r *http.Request, params map[string]string) {
	render.New().JSON(w, http.StatusOK, map[string]interface{}{
		"code":       "https://github.com/MixinNetwork/trusted-group/tree/master/mvm/quorum/bridge",
		"process":    MVMRegistryId,
		"registry":   "https://scan.mvm.dev/address/" + MVMRegistryContract,
		"bridge":     "https://scan.mvm.dev/address/" + MVMBridgeContract,
		"mirror":     "https://scan.mvm.dev/address/" + MVMMirrorContract,
		"withdrawal": "https://scan.mvm.dev/address/" + MVMWithdrawalContract,
		"storage":    "https://scan.mvm.dev/address/" + MVMStorageContract,

		"public_key_hex": CurvePublicKey(ServerPublic),
	})
}

func createUser(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if len(store.limiterAvailable(r.RemoteAddr)) > UserCreationLimit {
		render.New().JSON(w, http.StatusTooManyRequests, map[string]interface{}{"error": "too many request"})
		return
	}
	var body struct {
		PublicKey string `json:"public_key"`
		Signature string `json:"signature"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]interface{}{"error": err})
		return
	}
	err = store.writeLimiter(r.RemoteAddr)
	if err != nil {
		render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err})
		return
	}
	u, err := proxy.createUser(r.Context(), store, body.PublicKey, body.Signature)
	if err != nil {
		render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err})
		return
	}
	render.New().JSON(w, http.StatusOK, map[string]interface{}{"user": map[string]interface{}{
		"user_id":    u.UserID,
		"session_id": u.SessionID,
		"full_name":  u.FullName,
		"created_at": u.CreatedAt,
		"key": map[string]interface{}{
			"client_id":   u.Key.ClientID,
			"session_id":  u.Key.SessionID,
			"private_key": u.Key.PrivateKey,
		},
		"contract": u.Contract,
	}})
}

func assetInfo(w http.ResponseWriter, r *http.Request, params map[string]string) {
	id := strings.ToLower(strings.TrimSpace(params["id"]))
	aid, _ := uuid.FromString(id)
	var address common.Address
	var err error
	if aid.String() == id {
		k := new(big.Int).SetBytes(aid.Bytes())
		address, err = proxy.registry.Contracts(nil, k)
	} else {
		address = common.HexToAddress(id)
		var num *big.Int
		num, err = proxy.registry.Assets(nil, address)
		if err == nil {
			aid = uuid.FromBytesOrNil(num.Bytes())
		}
	}
	if err != nil {
		render.New().JSON(w, http.StatusAccepted, map[string]interface{}{"error": err.Error()})
		return
	}
	render.New().JSON(w, http.StatusOK, map[string]interface{}{"asset_id": aid.String(), "contract": address.String()})
}

func encodeExtra(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var body struct {
		PublicKey string `json:"public_key"`
		Action    Action `json:"action"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]interface{}{"error": err})
		return
	}
	pub, err := hex.DecodeString(body.PublicKey)
	if err != nil || len(pub) != 32 {
		render.New().JSON(w, http.StatusBadRequest, map[string]interface{}{"error": fmt.Errorf("invalid public key: %s", body.PublicKey)})
		return
	}
	extra, err := encodeActionAsExtra(pub, &body.Action)
	if err != nil {
		render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err})
		return
	}
	render.New().JSON(w, http.StatusOK, map[string]interface{}{"extra": extra})
}

func getTokenMeta(w http.ResponseWriter, r *http.Request, params map[string]string) {
	cb, err := hex.DecodeString(params["collection"])
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]interface{}{"error": err})
		return
	}
	collection := uuid.FromBytesOrNil(cb).String()

	tdt := fmt.Sprintf("https://thetrident.one/api/%s/%s", collection, params["id"])
	resp, err := http.Get(tdt)
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]interface{}{"error": err})
		return
	}
	defer resp.Body.Close()

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		render.New().JSON(w, http.StatusBadRequest, map[string]interface{}{"error": err})
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
			render.New().JSON(w, http.StatusOK, map[string]interface{}{})
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}
