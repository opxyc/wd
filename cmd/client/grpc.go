package main

import (
	"context"

	"github.com/opxyc/wd/wd"
	"google.golang.org/grpc"
)

// GC is a gRPC client handle
type GC struct {
	client wd.WatchdogClient
}

// New returns a gRPC Client handle that can be used to
// start a grpc server and send msgs.
func grpcCon(addr string) (*GC, error) {
	gc := &GC{}

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	gc.client = wd.NewWatchdogClient(conn)
	return gc, nil
}

// send sends a message to gRPC server
func (gc *GC) send(id, hostname, title, short, long string, status int32) error {
	_, err := gc.client.SendErrorMsg(context.Background(), &wd.ErrorMsg{
		Id:     id,
		From:   &wd.From{Hostname: hostname},
		Msg:    &wd.Msg{Title: title, Short: short, Long: long},
		Status: status,
	})
	return err
}
