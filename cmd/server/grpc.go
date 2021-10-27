package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/opxyc/wd/wd"
	"google.golang.org/grpc"
)

// gRPCServer creates a gRPC server and server
func gRPCServer(addr string) {
	srv := grpc.NewServer()
	var pb pbSrv
	wd.RegisterWatchdogServer(srv, pb)
	lsnr, err := net.Listen("tcp", addr)
	if err != nil {
		l.Fatalf("could not listen on %s: %v\n", addr, err)
	}

	l.Printf("gRPC listening on %s\n", addr)
	l.Fatal(srv.Serve(lsnr))
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
	info := fmt.Sprintf("| ALERT | %-38s %-16s %s", msg.Id, msg.From.Hostname, msg.Msg.Short)
	// l.Printf("\n%s\n%s\n", titleRow, info)
	l.Printf("%s\n", info)
	// send the received alert/msg to all ws connections
	pushmsg(msg)
	return &wd.Void{}, nil
}

// pushmsg broadcasts msg to websocket connections
func pushmsg(msg *wd.ErrorMsg) {
	m := &msgFormat{
		ID:     msg.Id,
		From:   msg.From.Hostname,
		Title:  msg.Msg.Title,
		Short:  msg.Msg.Short,
		Long:   msg.Msg.Long,
		Status: msg.Status,
	}
	b, err := json.Marshal(m)
	if err != nil {
		l.Printf("failed to marshal msg: %v", err)
		return
	}

	ws.Broadcast(b)
}

type msgFormat struct {
	ID     string `json:"id"`
	From   string `json:"from"`
	Title  string `json:"title"`
	Short  string `json:"short"`
	Long   string `json:"long"`
	Status int32  `json:"status"`
}
