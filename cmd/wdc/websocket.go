package main

import (
	"context"
	"encoding/json"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
)

// ws is handle for websocket connection
type ws struct {
	c *websocket.Conn
	l *log.Logger
}

// webSocketCon creates a websocket connection to addr on endpoint ep
// and returns a handle. You can also specify a logger which will be used
// to log incoming messages.
func websocketCon(addr, ep string, l *log.Logger) *ws {
	if l == nil {
		l = log.Default()
	}

	u := url.URL{Scheme: "ws", Host: addr, Path: ep}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	} else {
		log.Printf("connected  to %s\n", u.Host)
	}

	ws := ws{c: c, l: l}
	return &ws
}

// lnl listens and logs incoming messages
func (ws *ws) lnl(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, message, err := ws.c.ReadMessage()
			if err != nil {
				log.Printf("could not read alert: %v\n", err)
			}

			var alert alert
			err = json.Unmarshal(message, &alert)
			if err != nil {
				log.Printf("could not unmarshal alert: %v\n", err)
			}
			alog(&alert)
		}
	}
}

// Close closes a ws connection
func (ws *ws) Close() {
	err := ws.c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Printf("could not close connection: %v\n", err)
		return
	}
	log.Println("connection closed")
}

type alert struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	Title string `json:"title"`
	Short string `json:"short"`
	Long  string `json:"long"`
}
