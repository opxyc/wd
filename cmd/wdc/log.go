package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	// to keep track of whether header row was printed to console
	printed bool
	mu      sync.Mutex
)

// printHeader prints log header to the console if not already done
func printHeader() {
	if !printed {
		mu.Lock()
		fmt.Printf("%s\n", alogHeader())
		printed = true
		mu.Unlock()
	}
}

// alogHeader returns alert log header
func alogHeader() string {
	// header format:
	// 			 TIME		ID            Host          Message
	separator := strings.Repeat(" ", len("| ALERT |"))
	titleRow := fmt.Sprintf("%s %-13v %-13s %-16s %s", separator, "TIME", "ID", "Host", "Message")
	return titleRow
}

// alog logs alert to console and file
func alog(msg *alert) {
	// logs received msg in the format:
	// 			 TIME		ID            Host          Message
	// | ALERT | 13:00:10	1635149439253 srv01         cpu usage on > 10%. take action immediately

	// Note: the detailed info is not logged to console. it it stored to log file only.
	printHeader()
	info := fmt.Sprintf("| ALERT | %-13v %-13s %-16s %s", time.Now().Format("15:04:05"), msg.ID, msg.From, msg.Short)
	fmt.Printf("%s\n", info)

	// log to file
	l.Println(alogHeader())
	l.Printf("%s\n", info)
	l.Printf("| DETAILED INFO:\n%s\n", msg.Long)
}
