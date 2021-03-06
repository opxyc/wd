package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opxyc/goutils/logger"
)

var (
	ws *WS         // websocket	handler
	l  *log.Logger // logger
)

func main() {
	gRPCSrvAddr := flag.String("grpc-addr", ":40090", "network address addr on which gRPC server should listen on")
	httpAddr := flag.String("http-addr", ":40080", "network address addr on which http server should listen on")
	dir := flag.String("l", "log", "log directory")
	flag.Parse()

	// set up logger
	var err error
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	const logFileNameFormat = "2006-Jan-02"
	l, err = logger.NewDailyLogger(ctx, *dir, logFileNameFormat, 00, 00, os.Stdout)
	if err != nil {
		log.Fatalf("could not set logger #2: %v\n", err)
	}

	go gRPCServer(*gRPCSrvAddr)
	go websocketServer(*httpAddr, "/ws/connect", l)

	// wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)
	signalReceived := <-sigChan
	l.Printf("Received '%v', attempting graceful termination\n", signalReceived)
	tc, cf := context.WithTimeout(context.Background(), 1*time.Second)
	defer cf()
	select {
	case <-tc.Done():
		l.Println("done")
	}
}
