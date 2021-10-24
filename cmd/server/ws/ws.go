package ws

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	wsCons   = make(map[string]websocket.Conn, 1000) // ws connections
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// New returns a WS handle that can be used to
// start a ws server and broadcast msgs.
func New() *WS {
	return &WS{}
}

// WS is WebSocket handle
type WS struct{}

// Start starts and serves ws server
func (*WS) Start(addr string) {
	http.HandleFunc("/ws/connect", connect)
	log.Printf("http listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// RemoveCon removes the connection info with id from wsCons
func (*WS) RemoveCon(id string) {
	delete(wsCons, id)
}

// Broadcast broadcasts given msg to all the connections
func (*WS) Broadcast(msg []byte) {
	failedCons := []string{}
	for _, c := range wsCons {
		err := c.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Printf("failed to send msg to socket %s: %v\n", c.RemoteAddr().String(), err)
			failedCons = append(failedCons, c.RemoteAddr().String())
		}
	}

	for _, rAddr := range failedCons {
		delete(wsCons, rAddr)
	}
}

func connect(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	wsCons[c.RemoteAddr().String()] = *c
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, _, err := c.ReadMessage()
		fmt.Println(mt, c.RemoteAddr())
		if err != nil {
			log.Println("read:", err)
			break
		}
	}
}
