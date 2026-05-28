package router

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/deformal/kastql/internal/planner"
	"github.com/deformal/kastql/internal/security"
)

// graphql-transport-ws message types
const (
	msgConnectionInit = "connection_init"
	msgConnectionAck  = "connection_ack"
	msgSubscribe      = "subscribe"
	msgNext           = "next"
	msgError          = "error"
	msgComplete       = "complete"
	msgPing           = "ping"
	msgPong           = "pong"
)

type wsMessage struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type subscribePayload struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
	OperationName string         `json:"operationName"`
}

type wsHandler struct {
	planner *planner.Planner
	log     *zap.Logger
	secMgr  *security.Manager // nil = security disabled
}

func (h *wsHandler) upgrader() websocket.Upgrader {
	return websocket.Upgrader{
		Subprotocols: []string{"graphql-transport-ws", "graphql-ws"},
		CheckOrigin: func(r *http.Request) bool {
			if h.secMgr == nil {
				return true
			}
			cfg := h.secMgr.Config()
			if !cfg.CORSEnabled {
				return true
			}
			origin := r.Header.Get("Origin")
			return security.IsAllowedOrigin(origin, h.secMgr)
		},
	}
}

func (h *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// WebSocket connection limit.
	if h.secMgr != nil && !h.secMgr.WSAcquire() {
		http.Error(w, "too many WebSocket connections", http.StatusServiceUnavailable)
		return
	}

	u := h.upgrader()
	conn, err := u.Upgrade(w, r, nil)
	if err != nil {
		if h.secMgr != nil {
			h.secMgr.WSRelease()
		}
		h.log.Warn("ws upgrade failed", zap.Error(err))
		return
	}
	defer func() {
		conn.Close()
		if h.secMgr != nil {
			h.secMgr.WSRelease()
		}
	}()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	h.handleConn(ctx, conn, forwardHeaders(r))
}

func (h *wsHandler) handleConn(ctx context.Context, client *websocket.Conn, headers map[string]string) {
	// subscriptions maps client operation ID → cancel func for that upstream WS
	subs := map[string]context.CancelFunc{}
	var subsMu sync.Mutex

	for {
		_, raw, err := client.ReadMessage()
		if err != nil {
			break
		}

		var msg wsMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case msgConnectionInit:
			send(client, wsMessage{Type: msgConnectionAck})

		case msgPing:
			send(client, wsMessage{Type: msgPong})

		case msgSubscribe:
			if msg.ID == "" {
				continue
			}
			var payload subscribePayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				sendError(client, msg.ID, "invalid subscribe payload")
				continue
			}

			svcURL := h.resolveSubscriptionURL(payload.Query)
			if svcURL == "" {
				sendError(client, msg.ID, "no service found for subscription")
				continue
			}

			subCtx, subCancel := context.WithCancel(ctx)
			subsMu.Lock()
			subs[msg.ID] = subCancel
			subsMu.Unlock()

			go func(id, url string, p subscribePayload) {
				defer func() {
					subsMu.Lock()
					delete(subs, id)
					subsMu.Unlock()
					send(client, wsMessage{ID: id, Type: msgComplete})
				}()
				h.forwardSubscription(subCtx, client, id, url, headers, raw)
			}(msg.ID, svcURL, payload)

		case msgComplete:
			if msg.ID == "" {
				continue
			}
			subsMu.Lock()
			if cancel, ok := subs[msg.ID]; ok {
				cancel()
				delete(subs, msg.ID)
			}
			subsMu.Unlock()
		}
	}

	// Cancel all open subscriptions when client disconnects.
	subsMu.Lock()
	for _, cancel := range subs {
		cancel()
	}
	subsMu.Unlock()
}

// forwardSubscription opens a WebSocket to the upstream service and relays
// messages bidirectionally until ctx is cancelled or the upstream closes.
func (h *wsHandler) forwardSubscription(
	ctx context.Context,
	client *websocket.Conn,
	opID string,
	upstreamURL string,
	headers map[string]string,
	subscribeMsg []byte,
) {
	dialer := websocket.Dialer{}
	reqHdr := http.Header{}
	reqHdr.Set("Sec-Websocket-Protocol", "graphql-transport-ws")
	for k, v := range headers {
		reqHdr.Set(k, v)
	}

	upstream, _, err := dialer.DialContext(ctx, upstreamURL, reqHdr)
	if err != nil {
		h.log.Warn("ws upstream dial failed", zap.String("url", upstreamURL), zap.Error(err))
		sendError(client, opID, "upstream connection failed: "+err.Error())
		return
	}
	defer upstream.Close()

	// Send ConnectionInit to upstream.
	if err := upstream.WriteMessage(websocket.TextMessage, []byte(`{"type":"connection_init"}`)); err != nil {
		return
	}

	// Wait for ConnectionAck.
	_, ackRaw, err := upstream.ReadMessage()
	if err != nil {
		return
	}
	var ack wsMessage
	if err := json.Unmarshal(ackRaw, &ack); err != nil || ack.Type != msgConnectionAck {
		sendError(client, opID, "upstream did not ack connection")
		return
	}

	// Forward the Subscribe message (with the same operation ID).
	if err := upstream.WriteMessage(websocket.TextMessage, subscribeMsg); err != nil {
		return
	}

	// Relay upstream messages to client until done.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, raw, err := upstream.ReadMessage()
			if err != nil {
				return
			}
			var msg wsMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			// Ensure the ID matches the client's operation ID.
			msg.ID = opID
			if err := send(client, msg); err != nil {
				return
			}
			if msg.Type == msgComplete || msg.Type == msgError {
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		// Client cancelled — send Complete to upstream.
		upstream.WriteMessage(websocket.TextMessage, []byte(`{"id":"`+opID+`","type":"complete"}`))
	case <-done:
	}
}

// resolveSubscriptionURL finds the upstream WebSocket URL for a subscription
// query by inspecting the first root field and looking up its owning service.
func (h *wsHandler) resolveSubscriptionURL(query string) string {
	return h.planner.ResolveSubscriptionURL(query)
}

// ---- helpers ----

func send(conn *websocket.Conn, msg wsMessage) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, raw)
}

func sendError(conn *websocket.Conn, id, message string) {
	payload, _ := json.Marshal([]map[string]any{{"message": message}})
	send(conn, wsMessage{ID: id, Type: msgError, Payload: payload})
}
