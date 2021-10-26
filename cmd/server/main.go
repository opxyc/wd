package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opxyc/gowd/utils/logger"
)

const gRPCSrvAddr = ":40090"
const httpAddr = ":40080"

var (
	ws  *WS // websocket	handler
	dir = flag.String("l", "log", "log directory")
	l   *log.Logger // logger
)

func main() {
	flag.Parse()

	// set up logger
	var err error
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	l, err = logger.NewDailyLogger(ctx, *dir, 00, 00, os.Stdout)
	if err != nil {
		log.Fatalf("could not set logger #2: %v\n", err)
	}

	go gRPCServer()
	go websocketServer(httpAddr, "/ws/connect", nil)

	// wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)
	signalReceived := <-sigChan
	l.Printf("Received '%v', attempting graceful termination\n", signalReceived)
	tc, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	select {
	case <-tc.Done():
		l.Println("done")
	}
}
