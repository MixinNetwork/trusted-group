package services

import (
	"multisig/configs"
	"net/http"
	"net/url"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/gorilla/websocket"
)

func ConnectMixinBlaze() (*websocket.Conn, error) {
	mixin := configs.AppConfig.Mixin
	token, err := bot.SignAuthenticationToken(mixin.AppID, mixin.SessionID, mixin.PrivateKey, "GET", "/", "")
	if err != nil {
		return nil, err
	}
	header := make(http.Header)
	header.Add("Authorization", "Bearer "+token)
	u := url.URL{Scheme: "wss", Host: "blaze.mixin.one", Path: "/"}
	dialer := &websocket.Dialer{
		Subprotocols: []string{"Mixin-Blaze-1"},
	}
	conn, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
