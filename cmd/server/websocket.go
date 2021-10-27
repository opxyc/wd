package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// WS is websocket handle
type WS struct {
	// address of ws
	addr string
	// the endpoint on which nodes should hit to make connection
	ep string
	// map of conn info (of connected nodes)
	cons map[string]websocket.Conn
	l    *log.Logger
}

// Start starts the websocket server and listens on ws.ep
func (ws *WS) Start() error {
	http.Handle(ws.ep, connectHandler(ws, connect))
	return http.ListenAndServe(ws.addr, nil)
}

// Broadcast broadcasts a given msg to all the connections in ws
func (ws *WS) Broadcast(msg []byte) {
	failedCons := []string{}
	for _, c := range ws.cons {
		err := c.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			ws.l.Printf("failed to send msg to socket %s: %v\n", c.RemoteAddr().String(), err)
			failedCons = append(failedCons, c.RemoteAddr().String())
		}
	}

	for _, rAddr := range failedCons {
		delete(ws.cons, rAddr)
		ws.l.Printf("removed connection %s from %s%s\n", rAddr, ws.addr, ws.ep)
	}
}

// New creates a new websocket handle
func New(addr, ep string, l *log.Logger) *WS {
	return &WS{
		addr: addr,
		ep:   ep,
		cons: make(map[string]websocket.Conn, 1000),
		l:    l,
	}
}

// websocketServer creates a websocket server and listens for incoming
// connections. It also assigns a handle to the global var ws
// which can be used to broadcast messages to ws connections.
func websocketServer(addr, ep string, l *log.Logger) {
	ws = New(addr, ep, l)
	log.Printf("http listening on %v\n", ws.addr)
	ws.l.Fatal(ws.Start())
}

// connectHandler connects a node to given ws
func connectHandler(ws *WS, f func(ws *WS, rw http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		err := f(ws, rw, r)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			ws.l.Printf("error handling %v:%v", r.RequestURI, err)
		}
	}
}

func connect(ws *WS, w http.ResponseWriter, r *http.Request) error {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return nil
	}
	defer c.Close()

	ws.cons[r.RemoteAddr] = *c
	ws.l.Printf("new connection %s added on %+v%v :: total: %d\n", r.RemoteAddr, ws.addr, ws.ep, len(ws.cons))

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
	}
	return nil
}
