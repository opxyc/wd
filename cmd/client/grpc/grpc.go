package grpc

import (
	"context"

	"github.com/opxyc/gowd/wd"
	"google.golang.org/grpc"
)

// GC is a gRPC Client Handle
type GC struct {
	addr   string
	client wd.WatchdogClient
}

// New returns a gRPC Client handle that can be used to
// start a grpc server and send msgs.
func New(addr string) *GC {
	gc := GC{
		addr: addr,
	}
	return &gc
}

// Start starts and serves ws server
func (gc *GC) Start() error {
	conn, err := grpc.Dial(gc.addr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	gc.client = wd.NewWatchdogClient(conn)
	return nil
}

// Send sends a message to gRPC server
func (gc *GC) Send(id, hostname, title, short, long string) error {
	_, err := gc.client.SendErrorMsg(context.Background(), &wd.ErrorMsg{
		Id:   id,
		From: &wd.From{Hostname: hostname},
		Msg:  &wd.Msg{Title: title, Short: short, Long: long},
	})
	return err
}
