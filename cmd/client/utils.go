package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type cfg struct {
	Hostname string `json:"hostname"`
	Tasks    []task
}

type task struct {
	Name     string   `json:"name"`
	Interval int64    `json:"repeatInterval"`
	Cmd      string   `json:"cmd"`
	Msg      string   `json:"msg"`
	Actions  []action `json:"actionsToBeTaken"`
}

type action struct {
	Name     string `json:"name"`
	Cmd      string `json:"cmd"`
	Continue bool   `json:"continueOnFailure"`
}

// reads configuration file
func readFromCfg(path string) *cfg {
	f, err := os.Open("config.json")
	if err != nil {
		sl.Println("could not open config.json:", err)
		os.Exit(1)
	}
	enc := json.NewDecoder(f)
	cfg := cfg{}
	err = enc.Decode(&cfg)
	if err != nil {
		log.Println("could not decode config file:", err)
	}
	return &cfg
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
