package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opxyc/gowd/utils"
)

const gRPCSrvAddr = ":40090"
const httpAddr = ":40080"

var (
	ws *WS // websocket	handler
	f  = flag.String("l", "log/log.txt", "path to log file")
	l  *log.Logger // logger
	ml *log.Logger // multi logger - logs to file and stdout
)

func main() {
	flag.Parse()

	// set up logger
	lf, err := utils.LogFile(*f)
	if err != nil {
		log.Fatalf("could not set logger: %v\n", err)
	}
	defer lf.Close()

	mw := io.MultiWriter(lf, os.Stdout)
	ml = log.New(mw, "", log.LstdFlags)
	l = log.New(lf, "", log.LstdFlags)

	go gRPCServer()
	go websocketServer(httpAddr, "/ws/connect", nil)

	// wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)
	signalReceived := <-sigChan
	ml.Printf("Received '%v', attempting graceful termination\n", signalReceived)
	tc, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	select {
	case <-tc.Done():
		ml.Println("done")
	}
}
