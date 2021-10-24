package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opxyc/gowd/cmd/server/ws"
	"github.com/opxyc/gowd/wd"
	"google.golang.org/grpc"
)

const gRPCSrvAddr = ":40090"
const httpAddr = ":40080"

var wsH *ws.WS

func main() {
	flag.Parse()

	go func() {
		// start gRPC server
		srv := grpc.NewServer()
		var pb pbSrv
		wd.RegisterWatchdogServer(srv, pb)
		l, err := net.Listen("tcp", gRPCSrvAddr)
		if err != nil {
			log.Fatalf("could not listen on %s: %v\n", gRPCSrvAddr, err)
		}

		log.Printf("gRPC listening on %s\n", gRPCSrvAddr)
		log.Fatal(srv.Serve(l))
	}()

	go func() {
		// start ws server
		wsH = ws.New()
		wsH.Start(httpAddr)
	}()

	// wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)
	signalReceived := <-sigChan
	log.Printf("Received '%v', attempting graceful termination\n", signalReceived)
	tc, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	select {
	case <-tc.Done():
		log.Println("done")
	}
}

type pbSrv struct{}

func (pbSrv) SendErrorMsg(ctx context.Context, msg *wd.ErrorMsg) (*wd.Void, error) {
	fmt.Printf("%-10s %-14s %-20s %s\n", msg.Id, msg.From, msg.Msg.Title, msg.Msg.Short)
	// send the received alert/msg to all ws connections
	pushmsg(msg)
	return &wd.Void{}, nil
}

func pushmsg(msg *wd.ErrorMsg) {
	m := &msgFormat{
		ID:    msg.Id,
		From:  msg.From.Hostname,
		Title: msg.Msg.Title,
		Short: msg.Msg.Short,
		Long:  msg.Msg.Long,
	}
	b, err := json.Marshal(m)
	if err != nil {
		log.Printf("failed to marshal msg: %v", err)
	}

	wsH.Broadcast(b)
}

type msgFormat struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	Title string `json:"title"`
	Short string `json:"short"`
	Long  string `json:"long"`
}
