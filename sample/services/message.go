package services

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"multisig/models"
	"multisig/session"
	"strings"
	"sync"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

const (
	keepAlivePeriod = 5 * time.Second
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = (pongWait * 9) / 10
)

type BlazeMessage struct {
	ID     string                 `json:"id"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params,omitempty"`
	Data   interface{}            `json:"data,omitempty"`
	Error  *bot.Error             `json:"error,omitempty"`
}

type MessageView struct {
	ConversationID string    `json:"conversation_id"`
	UserID         string    `json:"user_id"`
	MessageID      string    `json:"message_id"`
	Category       string    `json:"category"`
	Data           string    `json:"data"`
	Status         string    `json:"status"`
	Source         string    `json:"source"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TransferView struct {
	Type       string    `json:"type"`
	SnapshotID string    `json:"snapshot_id"`
	OpponentID string    `json:"opponent_id"`
	AssetId    string    `json:"asset_id"`
	Amount     string    `json:"amount"`
	TraceID    string    `json:"trace_id"`
	Memo       string    `json:"memo"`
	CreatedAt  time.Time `json:"created_at"`
}

type MessageService struct{}

type MessageContext struct {
	Transactions *tmap
	ReadDone     chan bool
	WriteDone    chan bool
	ReadBuffer   chan MessageView
	WriteBuffer  chan []byte
}

func (service *MessageService) Run(ctx context.Context) error {
	for {
		err := service.loop(ctx)
		if err != nil {
			session.Logger(ctx).Error(err)
		}
		session.Logger(ctx).Info("connection loop end")
		time.Sleep(300 * time.Millisecond)
	}
}

func (service *MessageService) loop(ctx context.Context) error {
	conn, err := ConnectMixinBlaze()
	if err != nil {
		return err
	}
	defer conn.Close()

	mc := &MessageContext{
		Transactions: newTmap(),
		ReadDone:     make(chan bool, 1),
		WriteDone:    make(chan bool, 1),
		ReadBuffer:   make(chan MessageView, 102400),
		WriteBuffer:  make(chan []byte, 102400),
	}

	go writePump(ctx, conn, mc)
	go readPump(ctx, conn, mc)

	err = writeMessageAndWait(ctx, mc, "LIST_PENDING_MESSAGES", nil)
	if err != nil {
		return err
	}

	for {
		select {
		case <-mc.ReadDone:
			return nil
		case msg := <-mc.ReadBuffer:
			if strings.Contains(msg.Category, "PLAIN_") {
				codeID, err := models.HandleMessage(ctx, msg.UserID)
				if err != nil {
					return err
				}
				data := fmt.Sprintf("CNB will be refund: mixin://codes/%s", codeID)
				params := map[string]interface{}{
					"conversation_id": msg.ConversationID,
					"message_id":      uuid.Must(uuid.NewV4()).String(),
					"category":        "PLAIN_TEXT",
					"data":            base64.StdEncoding.EncodeToString([]byte(data)),
				}
				err = writeMessageAndWait(ctx, mc, "CREATE_MESSAGE", params)
				if err != nil {
					return err
				}
			}
			params := map[string]interface{}{"message_id": msg.MessageID, "status": "READ"}
			err = writeMessageAndWait(ctx, mc, "ACKNOWLEDGE_MESSAGE_RECEIPT", params)
			if err != nil {
				return err
			}
		}
	}
}

func readPump(ctx context.Context, conn *websocket.Conn, mc *MessageContext) error {
	defer func() {
		conn.Close()
		mc.WriteDone <- true
		mc.ReadDone <- true
	}()
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		messageType, wsReader, err := conn.NextReader()
		if err != nil {
			return session.BlazeServerError(ctx, err)
		}
		if messageType != websocket.BinaryMessage {
			return session.BlazeServerError(ctx, fmt.Errorf("invalid message type %d", messageType))
		}
		err = parseMessage(ctx, mc, wsReader)
		if err != nil {
			return session.BlazeServerError(ctx, err)
		}
	}
}

func writePump(ctx context.Context, conn *websocket.Conn, mc *MessageContext) error {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		conn.Close()
	}()
	for {
		select {
		case data := <-mc.WriteBuffer:
			err := writeGzipToConn(ctx, conn, data)
			if err != nil {
				return err
			}
		case <-mc.WriteDone:
			return nil
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				return err
			}
		}
	}
}

func writeMessageAndWait(ctx context.Context, mc *MessageContext, action string, params map[string]interface{}) error {
	var resp = make(chan BlazeMessage, 1)
	var id = bot.UuidNewV4().String()
	mc.Transactions.set(id, func(t BlazeMessage) error {
		select {
		case resp <- t:
		case <-time.After(1 * time.Second):
			return fmt.Errorf("timeout to hook %s %s", action, id)
		}
		return nil
	})

	blazeMessage, err := json.Marshal(BlazeMessage{ID: id, Action: action, Params: params})
	if err != nil {
		return err
	}
	select {
	case <-time.After(keepAlivePeriod):
		return fmt.Errorf("timeout to write %s %v", action, params)
	case mc.WriteBuffer <- blazeMessage:
	}

	select {
	case <-time.After(keepAlivePeriod):
		return fmt.Errorf("timeout to wait %s %v", action, params)
	case t := <-resp:
		if t.Error != nil && t.Error.Code != 403 {
			return writeMessageAndWait(ctx, mc, action, params)
		}
	}
	return nil
}

func writeGzipToConn(ctx context.Context, conn *websocket.Conn, msg []byte) error {
	conn.SetWriteDeadline(time.Now().Add(writeWait))
	wsWriter, err := conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	gzWriter, err := gzip.NewWriterLevel(wsWriter, 3)
	if err != nil {
		return err
	}
	if _, err := gzWriter.Write(msg); err != nil {
		return err
	}

	if err := gzWriter.Close(); err != nil {
		return err
	}
	if err := wsWriter.Close(); err != nil {
		return err
	}
	return nil
}

func parseMessage(ctx context.Context, mc *MessageContext, wsReader io.Reader) error {
	var message BlazeMessage
	gzReader, err := gzip.NewReader(wsReader)
	if err != nil {
		return err
	}
	defer gzReader.Close()
	if err = json.NewDecoder(gzReader).Decode(&message); err != nil {
		return err
	}

	transaction := mc.Transactions.retrive(message.ID)
	if transaction != nil {
		return transaction(message)
	}

	if message.Action != "CREATE_MESSAGE" {
		return nil
	}

	data, err := json.Marshal(message.Data)
	if err != nil {
		return err
	}
	var msg MessageView
	err = json.Unmarshal(data, &msg)
	if err != nil {
		return err
	}

	select {
	case <-time.After(keepAlivePeriod):
		return fmt.Errorf("timeout to handle %s %s", msg.Category, msg.MessageID)
	case mc.ReadBuffer <- msg:
	}
	return nil
}

type tmap struct {
	mutex sync.Mutex
	m     map[string]mixinTransaction
}

type mixinTransaction func(BlazeMessage) error

func newTmap() *tmap {
	return &tmap{
		m: make(map[string]mixinTransaction),
	}
}

func (m *tmap) retrive(key string) mixinTransaction {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	defer delete(m.m, key)
	return m.m[key]
}

func (m *tmap) set(key string, t mixinTransaction) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.m[key] = t
}

func sendAppButton(ctx context.Context, mc *MessageContext, msg MessageView, appID string) error {
	btns, err := json.Marshal([]interface{}{
		map[string]string{
			"label":  "Transfer 1 CNB, will be refunded",
			"action": fmt.Sprintf("https://mixin.one/pay?recipient=%s&asset=%s&amount=1&trace=%s&memo=", appID, models.CNBAssetID, bot.UuidNewV4().String()),
			"color":  "#FF5733",
		},
	})
	if err != nil {
		return err
	}
	params := map[string]interface{}{
		"conversation_id": msg.ConversationID,
		"message_id":      bot.UuidNewV4().String(),
		"category":        "APP_BUTTON_GROUP",
		"data":            base64.StdEncoding.EncodeToString(btns),
	}
	return writeMessageAndWait(ctx, mc, "CREATE_MESSAGE", params)
}
