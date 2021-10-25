package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
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

var (
	wsH         *ws.WS
	logFilePath = flag.String("l", "log/log.txt", "path to log file")
	l           *log.Logger // logger
	ml          *log.Logger // multi logger - logs to file and stdout
)

func main() {
	flag.Parse()

	// set up logger
	lf, err := os.OpenFile(*logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 06666)
	if err != nil {
		log.Fatalf("could not open %s for logging: %v\n", *logFilePath, err)
	}
	defer lf.Close()
	mw := io.MultiWriter(lf, os.Stdout)
	ml = log.New(mw, "", log.LstdFlags)
	l = log.New(lf, "", log.LstdFlags)

	go func() {
		// start gRPC server
		srv := grpc.NewServer()
		var pb pbSrv
		wd.RegisterWatchdogServer(srv, pb)
		lsnr, err := net.Listen("tcp", gRPCSrvAddr)
		if err != nil {
			ml.Fatalf("could not listen on %s: %v\n", gRPCSrvAddr, err)
		}

		ml.Printf("gRPC listening on %s\n", gRPCSrvAddr)
		ml.Fatal(srv.Serve(lsnr))
	}()

	go func() {
		// start ws server
		wsH = ws.New(httpAddr, "/ws/connect", nil)
		ml.Printf("http listening on %s\n", httpAddr)
		wsH.Start()
	}()

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

type pbSrv struct{}

func (pbSrv) SendErrorMsg(ctx context.Context, msg *wd.ErrorMsg) (*wd.Void, error) {
	l.Printf("%-10s %-14s %-20s %s\n", msg.Id, msg.From, msg.Msg.Title, msg.Msg.Short)
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
		l.Printf("failed to marshal msg: %v", err)
		return
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
