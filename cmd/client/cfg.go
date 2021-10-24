package main

import (
	"encoding/json"
	"log"
	"os"
)

type tasks []task

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

func readFromCfg(path string) *tasks {
	f, err := os.Open("config.json")
	if err != nil {
		l.Println("could not open config.json:", err)
		os.Exit(1)
	}
	enc := json.NewDecoder(f)
	cfg := tasks{}
	err = enc.Decode(&cfg)
	if err != nil {
		log.Println("could not decode config file:", err)
	}
	return &cfg
}
