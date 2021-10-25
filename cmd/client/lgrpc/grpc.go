package lgrpc

import (
	"context"

	"github.com/opxyc/gowd/wd"
	"google.golang.org/grpc"
)

// GC is a gRPC Client Handle
type GC struct {
	Client wd.WatchdogClient
}

// New returns a gRPC Client handle that can be used to
// start a grpc server and send msgs.
func New(addr string) (*GC, error) {
	gc := &GC{}

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	gc.Client = wd.NewWatchdogClient(conn)
	return gc, nil
}

// Send sends a message to gRPC server
func (gc *GC) Send(id, hostname, title, short, long string) error {
	_, err := gc.Client.SendErrorMsg(context.Background(), &wd.ErrorMsg{
		Id:   id,
		From: &wd.From{Hostname: hostname},
		Msg:  &wd.Msg{Title: title, Short: short, Long: long},
	})
	return err
}
