package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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
	// logs received msg in the format:
	// 			ID            Host          Message
	// |ALERT | 1635149439253 srv01         cpu usage on > 10%. take action immediately

	// Note: the detailed info is not logged based on the assumption that it's better to inspect the same
	// from the machine which generated alert given the log ID.
	// also, the detailed info is sent to wdc client which makes more sense.

	// separator := strings.Repeat(" ", len("| ALERT |"))
	// titleRow := fmt.Sprintf("%s %-13s %-16s %s", separator, "ID", "Host", "Message")
	info := fmt.Sprintf("| ALERT | %-13s %-16s %s", msg.Id, msg.From.Hostname, msg.Msg.Short)
	// l.Printf("\n%s\n%s\n", titleRow, info)
	l.Printf("%s\n", info)
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
