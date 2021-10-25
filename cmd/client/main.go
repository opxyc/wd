package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/opxyc/gowd/cmd/client/lgrpc"
	"github.com/opxyc/gowd/wd"
	"google.golang.org/grpc"
)

var (
	cfgFile     = flag.String("c", "config.json", "path to cfg file")
	addr        = flag.String("r", "localhost:40090", "server address in the format IP:PORT")
	logFilePath = flag.String("l", "log/log.txt", "path to log file")
	l           *log.Logger       // logger
	ml          *log.Logger       // multi logger - logs to file and stdout
	client      wd.WatchdogClient // grpc client
	gc          *lgrpc.GC         // grpc client
	hostname    string            // system hostname
)

func main() {
	flag.Parse()

	// read cfg file
	cfg := readFromCfg(*cfgFile)

	// set up logger
	lf, err := os.OpenFile(*logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 06666)
	if err != nil {
		log.Fatalf("could not open %s for logging: %v\n", *logFilePath, err)
	}
	defer lf.Close()
	mw := io.MultiWriter(lf, os.Stdout)
	ml = log.New(mw, "", log.LstdFlags)
	l = log.New(lf, "", log.LstdFlags)

	// // register gRPC client
	// client, err = gRPCClient(*addr)
	// if err != nil {
	// 	ml.Printf("could not start gRPC client: %v", err)
	// }

	// register gRPC client
	gc, err = lgrpc.New(*addr)
	if err != nil {
		ml.Printf("could not start gRPC client: %v", err)
	}

	// get hostname
	hostname = cfg.Hostname
	if cfg.Hostname == "" {
		hostname, _ = os.Hostname()
	}

	ml.Printf("---client (%s) started---\n", hostname)

	// execute tasks
	ctx, cancelFunc := context.WithCancel(context.Background())
	for _, t := range cfg.Tasks {
		go func(t task) {
			execute(ctx, &t)
		}(t)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	signalReceived := <-sigChan
	ml.Printf("received '%v', attempting graceful termination\n", signalReceived)
	cancelFunc()
	tc, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	select {
	case <-tc.Done():
		ml.Println("done")
		lf.Close()
	}
}

// execute executes a given task repetely according to the interval mentioned
func execute(ctx context.Context, t *task) string {
	for {
		select {
		case <-ctx.Done():
			return t.Name
		case <-time.After(time.Second * time.Duration(t.Interval)):
			id := fmt.Sprintf("%v", time.Now().UnixMilli())
			var sb strings.Builder
			var op string
			op = mlog(l, t.Name, nil, "", fmt.Sprintf("starting with id %s", id))
			sb.WriteString(op)

			op, err := run(t)

			if err == nil {
				mlog(l, t.Name, nil, "", "completed successfully")
				continue
			}

			sb.WriteString(op)

			// execute actions if any
			if len(t.Actions) > 0 {
				op, _ := runActions(t, err)
				sb.WriteString(op)
			}
			err = gc.Send(id, hostname, t.Name, t.Msg, sb.String())
			if err != nil {
				mlog(l, t.Name, nil, "", fmt.Sprintf("could not send msg to server: %v", err))
			}
			mlog(l, t.Name, nil, "", "completed with error")
		}
	}
}

// run runs a command and retuns the err
func run(t *task) (op string, err error) {
	cmd := exec.Command(t.Cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		op := mlog(l, t.Name, err, string(out), "")
		return op, err
	}

	return
}

// runActions runs verious actions mentioned in a task
func runActions(t *task, err error) (string, bool) {
	var sb strings.Builder
	mlog(l, t.Name, nil, "", "executing actions")

	var errOccured bool
	// execute actions serially
	for _, actn := range t.Actions {
		cmd := exec.Command(actn.Cmd)
		op, err := cmd.CombinedOutput()
		// if no errors continue
		if err != nil {
			errOccured = true
			sb.WriteString(mlog(l, strings.Join([]string{t.Name, actn.Name}, "."), err, string(op), ""))
			// if it's not mentioned to continue in cfg, do not perform next action
			if !actn.Continue {
				break
			}
		}
	}

	return sb.String(), errOccured
}

// gRPClient creates and returns a gRPC client
func gRPCClient(addr string) (wd.WatchdogClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client = wd.NewWatchdogClient(conn)
	return client, nil
}
