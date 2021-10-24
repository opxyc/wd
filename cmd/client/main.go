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

	"github.com/opxyc/gowd/wd"
	"google.golang.org/grpc"
)

const (
	logFilePath = "log/log.txt"
)

var (
	cfgFile  = flag.String("c", "congif.json", "path to cfg file")
	lf       *os.File          // log file
	l        *log.Logger       // logger
	ml       *log.Logger       // multi logger - logs to file and stdout
	client   wd.WatchdogClient // grpc client
	hostname string            // system hostname
	addr     = flag.String("r", "localhost:40090", "server address in the format IP:PORT")
)

func init() {
	// get hostname
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = fmt.Sprintf("%v", time.Now().Unix())
		l.Printf("faile to get hostname. using random id %s\n", hostname)
	}

	// set up logger
	lf, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 06666)
	if err != nil {
		log.Fatalf("could not open %s for logging: %v\n", logFilePath, err)
	}
	mw := io.MultiWriter(lf, os.Stdout)
	ml = log.New(mw, "", log.LstdFlags)
	l = log.New(lf, "", log.LstdFlags)

	// set up gRPC client
	client, err = gRPCClient(*addr)
	if err != nil {
		ml.Printf("could not start gRPC client: %v", err)
	}

	ml.Printf("\n----------------------------\nclient (%s) started\n----------------------------", hostname)
}

func main() {
	flag.Parse()

	// close logger when done
	defer lf.Close()

	// read cfg file
	tasks := readFromCfg(*cfgFile)

	// execute the tasks
	for _, t := range *tasks {
		go func(t task) {
			execute(&t)
		}(t)
	}

	// wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)
	signalReceived := <-sigChan
	ml.Printf("Received '%v', attempting graceful termination\n", signalReceived)
	// close logfile
	lf.Close()
	// wait for few seconds so that if some os.exec is running, let it complete...
	tc, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	select {
	case <-tc.Done():
		ml.Println("done")
	}
}

// execute executes a given task repetely according to the interval mentioned
func execute(t *task) {
	for {
		select {
		case <-time.After(time.Second * time.Duration(t.Interval)):
			id := fmt.Sprintf("%v", time.Now().Unix())
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
				op, errorOccured := runActions(t, err)
				sb.WriteString(op)
				if errorOccured {
					send(id, t.Name, t.Msg, sb.String())
				} else {
					send(id, t.Name, t.Msg, sb.String())
				}
			} else {
				send(id, t.Name, t.Msg, "")
			}
			mlog(l, t.Name, nil, "", "completed with error")
		}
	}
}

// run runs a command and retuns the err
func run(t *task) (string, error) {
	cmd := exec.Command(t.Cmd)
	op, err := cmd.CombinedOutput()
	if err != nil {
		out := mlog(l, t.Name, err, string(op), "")
		return out, err
	}

	return "", nil
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

// mlog will log given tName, err, op and info to the logger l and
// return what ever is logged
func mlog(l *log.Logger, tName string, err error, op string, info string) string {
	var sb strings.Builder
	// format:
	// TIMEBLABLA (tname.err) some error
	// TIMEBLABLA (tname.out) output received both stdout and stderr since we are
	// using CombinedOutput
	// TIMEBLABLA (tname.inf) some msg
	if err != nil {
		l.Printf("(%s.err) %s\n", tName, err)
		sb.WriteString(fmt.Sprintf("(%s.err) %s\n", tName, err))
	}
	if op != "" {
		l.Printf("(%s.out) %s", tName, op)
		sb.WriteString(fmt.Sprintf("(%s.out) %s", tName, op))
	}
	if info != "" {
		l.Printf("(%s.inf) %s\n", tName, info)
		sb.WriteString(fmt.Sprintf("(%s.inf) %s\n", tName, info))
	}

	return sb.String()
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

// send sends gRPC message to server
// need title, short and long msgs
func send(id, title, short, long string) error {
	_, err := client.SendErrorMsg(context.Background(), &wd.ErrorMsg{
		Id:   id,
		From: &wd.From{Hostname: hostname},
		Msg:  &wd.Msg{Title: title, Short: short, Long: long},
	})
	return err
}
