package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/opxyc/gowd/utils"
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

	f := filepath.Join(d, "WatchDog-client", "logs", "log.txt")
	lf, err := utils.LogFile(f)
	if err != nil {
		log.Fatalf("could not set logger: %v\n", err)
	}
	defer lf.Close()

	log.Printf("logging to '%v'\n", filepath.Dir(lf.Name()))
	l = log.New(lf, "", log.LstdFlags)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	// make websocket connection
	ws := websocketCon(*addr, *ep, l)
	// listen and log incoming messages
	go ws.lnl(ctx)

	// wait for interrupt (if any)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	log.Printf("received '%v\n", <-sigchan)
	// close websocket connection
	ws.Close()
}
