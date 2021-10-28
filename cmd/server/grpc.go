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
	// log msg to file..
	info := fmt.Sprintf("%-23s %-16s %s", msg.Id, msg.From.Hostname, msg.Msg.Short)
	l.Printf("%s\n", info)

	// send the received alert/msg to all ws connections
	pushmsg(msg)
	return &wd.Void{}, nil
}

// pushmsg broadcasts msg to websocket connections
func pushmsg(msg *wd.ErrorMsg) {
	m := &msgFormat{
		Time:     msg.Msg.Time,
		ID:       msg.Id,
		From:     msg.From.Hostname,
		TaskName: msg.From.TaskName,
		Short:    msg.Msg.Short,
		Long:     msg.Msg.Long,
		Status:   msg.Status,
	}
	b, err := json.Marshal(m)
	if err != nil {
		l.Printf("failed to marshal msg: %v", err)
		return
	}

	ws.Broadcast(b)
}

type msgFormat struct {
	Time     string `json:"time"`
	ID       string `json:"id"`
	From     string `json:"from"`
	TaskName string `json:"taskName"`
	Short    string `json:"short"`  // short message - msg field in client config.json
	Long     string `json:"long"`   // long message - combined output of `cmd`
	Status   int32  `json:"status"` // 0 if success, 1 if failed
}
