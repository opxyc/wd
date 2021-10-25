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

	"github.com/opxyc/gowd/utils"
	"github.com/opxyc/gowd/wd"
	"google.golang.org/grpc"
)

var (
	cfgF     = flag.String("c", "config.json", "path to cfg file")
	addr     = flag.String("r", "localhost:40090", "server address in the format IP:PORT")
	sf       = flag.String("sl", "log/self-logs.txt", "client specific log file")
	tf       = flag.String("tl", "log/task-logs.txt", "task execution log file")
	sl       *log.Logger       // self logger - for logging client specific stuff
	tl       *log.Logger       // task execution logger
	ml       *log.Logger       // multi logger - logs to file and stdout
	client   wd.WatchdogClient // grpc client
	gc       *GC               // grpc client
	hostname string            // system hostname
)

func main() {
	flag.Parse()

	// set up loggers
	// ---------------------------
	// self logger
	slf, err := utils.LogFile(*sf)
	if err != nil {
		log.Fatalf("could not set logger: %v\n", err)
	}
	defer slf.Close()

	tlf, err := utils.LogFile(*tf)
	if err != nil {
		log.Fatalf("could not set logger: %v\n", err)
	}
	defer slf.Close()
	mw := io.MultiWriter(slf, os.Stdout)
	ml = log.New(mw, "", log.LstdFlags)
	sl = log.New(slf, "", log.LstdFlags)
	tl = log.New(tlf, "", log.LstdFlags)
	// ------------------------------

	// register gRPC client
	gc, err = grpcCon(*addr)
	if err != nil {
		ml.Fatalf("could not start gRPC client: %v", err)
	}

	// read cfg file
	cfg := readFromCfg(*cfgF)
	hostname = cfg.Hostname
	if cfg.Hostname == "" {
		hostname, err = os.Hostname()
	}

	ml.Printf("client (%s) started\n", hostname)

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
			mlog(tl, t.Name, nil, "", fmt.Sprintf("starting with ID %v", id))
			sb.WriteString(op)

			errOp, err := run(t)

			if err == nil {
				mlog(tl, t.Name, nil, "", "completed successfully")
				continue
			}

			sb.WriteString(errOp)

			// execute actions if any
			if len(t.Actions) > 0 {
				errOp, errorOccured := runActions(t, err)
				// errOp != nil means anyone of the actions failed
				sb.WriteString(*errOp)
				if errorOccured {
					sb.WriteString("Actions failed to complete. Status: Need manual INT\n")
				} else {
					sb.WriteString("Actions completed successfully. Status: OK\n")
				}
			}
			err = gc.send(id, hostname, t.Name, t.Msg, sb.String())
			if err != nil {
				ml.Printf("could not send msg to server: %v", err)
			}
			mlog(tl, t.Name, nil, "", "completed with error")
		}
	}
}

// run runs a command and retuns the err and output in errOp
func run(t *task) (errOp string, err error) {
	cmd := exec.Command(t.Cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		errOp := mlog(tl, t.Name, err, string(out), "")
		return errOp, err
	}

	return
}

// runActions runs verious actions mentioned in a task.
// if any of the actions failed to complete, it will return the error and output combined
func runActions(t *task, err error) (*string, bool) {
	var sb strings.Builder
	mlog(tl, t.Name, nil, "", "running actions")

	var errorOccured bool
	// execute actions serially
	for _, actn := range t.Actions {
		cmd := exec.Command(actn.Cmd)
		op, err := cmd.CombinedOutput()
		// if no errors continue
		if err != nil {
			errorOccured = true
			sb.WriteString(mlog(tl, strings.Join([]string{t.Name, actn.Name}, "."), err, string(op), ""))
			// if it's not mentioned to continue in cfg, do not perform next action
			if !actn.Continue {
				break
			}
		}
	}

	errOp := sb.String()
	return &errOp, errorOccured
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
