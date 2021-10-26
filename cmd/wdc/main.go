package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/opxyc/gowd/utils/logger"
)

var l *log.Logger

func main() {
	addr := flag.String("r", "localhost:40080", "http service address")
	ep := flag.String("ep", "/ws/connect", "http service address")
	flag.Parse()

	// set up logger that logs into user's $HOME/WatchDog-client/logs
	d, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("could not get your home directory: %v\n", err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	dir := filepath.Join(d, "WatchDog-client", "logs")
	l, err = logger.NewDailyLogger(ctx, dir, 00, 00)
	if err != nil {
		log.Fatalf("could not set logger: %v\n", err)
	}

	// make websocket connection
	ws := websocketCon(*addr, *ep)
	// listen and log incoming messages
	go ws.lnl(ctx)

	// wait for interrupt (if any)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	log.Printf("received '%v\n", <-sigchan)
	log.Println("saving logs..")
	// cancel context which will close ws and log file
	cancelFunc()
	time.Sleep(time.Millisecond * 300)
	log.Println("done")
}
