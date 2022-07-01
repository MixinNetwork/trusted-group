package main

// FIXME do rate limit based on IP
// POST /users
// POST /extra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
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
	router.GET("/assets/:id/contract", assetContract)
	router.GET("/assets/:contract/id", assetInfo)
	router.POST("/extra", encodeExtra)
	router.POST("/users", createUser)
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
		"withdrawal": "https://scan.mvm.dev/address/" + MVMWithdrawalContract,
		"storage":    "https://scan.mvm.dev/address/" + MVMStorageContract,

		"public_key_base64": CurvePublicKey(ServerPublic),
	})
}

func createUser(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if len(store.limiterAvailable(r.RemoteAddr)) > UserCreationLimit {
		render.New().JSON(w, http.StatusTooManyRequests, map[string]interface{}{"error": "too many request"})
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

func assetContract(w http.ResponseWriter, r *http.Request, params map[string]string) {
	id, _ := uuid.FromString(params["id"])
	if id.String() != params["id"] {
		render.New().JSON(w, http.StatusAccepted, map[string]interface{}{"error": fmt.Sprintf("invalid asset id %s", params["id"])})
		return
	}
	k := new(big.Int).SetBytes(id.Bytes())
	address, err := proxy.registry.Contracts(nil, k)
	if err != nil {
		render.New().JSON(w, http.StatusAccepted, map[string]interface{}{"error": err.Error()})
	}
	render.New().JSON(w, http.StatusOK, map[string]interface{}{"contract": address.String()})
}

func assetInfo(w http.ResponseWriter, r *http.Request, params map[string]string) {
	address := common.HexToAddress(params["contract"])
	num, err := proxy.registry.Assets(nil, address)
	if err != nil {
		render.New().JSON(w, http.StatusAccepted, map[string]interface{}{"error": err.Error()})
	}
	id, err := uuid.FromBytes(num.Bytes())
	if err != nil {
		render.New().JSON(w, http.StatusAccepted, map[string]interface{}{"uuid error": err.Error()})
	}
	render.New().JSON(w, http.StatusOK, map[string]interface{}{"asset_id": id.String()})
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
