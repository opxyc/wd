package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lithammer/shortuuid"
	"github.com/opxyc/goutils/logger"
	"github.com/opxyc/wd/wd"
	"google.golang.org/grpc"
)

var (
	cfgF     = flag.String("c", "config.json", "path to cfg file")
	addr     = flag.String("r", "localhost:40090", "server address in the format IP:PORT")
	sDir     = flag.String("sl", "log/self", "client specific log directory")
	tDir     = flag.String("tl", "log/task", "task execution log directory")
	sl       *log.Logger       // self logger - for logging client specific stuff
	tl       *log.Logger       // task execution logger
	client   wd.WatchdogClient // grpc client
	gc       *GC               // grpc client
	hostname string            // system hostname
)

func main() {
	flag.Parse()

	// set up loggers
	// ---------------------------
	var err error
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	const logFileNameFormat = "2006-Jan-02"
	sl, err = logger.NewDailyLogger(ctx, *sDir, logFileNameFormat, 00, 00, os.Stdout)
	if err != nil {
		log.Fatalf("could not set logger #1: %v\n", err)
	}

	tl, err = logger.NewDailyLogger(ctx, *tDir, logFileNameFormat, 00, 00)
	if err != nil {
		log.Fatalf("could not set logger #2: %v\n", err)
	}
	// ------------------------------

	// register gRPC client
	gc, err = grpcCon(*addr)
	if err != nil {
		sl.Fatalf("could not start gRPC client: %v", err)
	}

	// read cfg file
	cfg := readFromCfg(*cfgF)
	hostname = cfg.Hostname
	if cfg.Hostname == "" {
		hostname, err = os.Hostname()
	}

	sl.Printf("client (%s) started\n", hostname)

	// execute tasks
	for _, t := range cfg.Tasks {
		go func(t task) {
			execute(ctx, &t)
		}(t)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	signalReceived := <-sigChan
	sl.Printf("received '%v', attempting graceful shutdown\n", signalReceived)
	cancelFunc()
	time.Sleep(time.Millisecond * 500)
	log.Println("done")
}

// execute executes a given task repetely according to the interval mentioned
func execute(ctx context.Context, t *task) string {
	for {
		select {
		case <-ctx.Done():
			return t.Name
		case <-time.After(time.Second * time.Duration(t.Interval)):
			id := shortuuid.New()
			var sb strings.Builder
			var op string
			mlog(tl, t.Name, nil, "", fmt.Sprintf("starting with ID %v", id))
			sb.WriteString(op)

			errOp, err := run(t)

			if err == nil {
				mlog(tl, t.Name, nil, "", "completed successfully")
				continue
			}

			// error has occured...
			// set status to 1
			var status int32 = 1
			sb.WriteString(errOp)

			// execute actions if any
			if len(t.Actions) > 0 {
				errOp, errorOccured := runActions(t, err)
				// errOp != nil means anyone of the actions failed
				sb.WriteString(*errOp)
				if !errorOccured {
					// actions were taken and situation is handled.
					// so set status to 0
					status = 0
				}
			}
			err = gc.send(id, hostname, t.Name, t.Name, t.Msg, sb.String(), status)
			if err != nil {
				sl.Printf("could not send msg to server: %v", err)
			}
			mlog(tl, t.Name, nil, "", fmt.Sprintf("completed with status %v", status))
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
