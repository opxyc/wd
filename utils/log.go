package utils

import (
	"log"
	"os"
	"path/filepath"
)

// LogFile check if log directory exists or not.t
// If not, it will ry to create it and open the file f and returns it's descriptor.
func LogFile(f string) (*os.File, error) {
	// check if log file path's parent dir exists
	logDir := filepath.Dir(f)
	_, err := os.Stat(logDir)
	if err != nil {
		log.Printf("could not stat log dir: %v\n", err)
		err = os.MkdirAll(logDir, 0755)
		if err != nil {
			return nil, err
		}
		log.Printf("created log dir '%v'\n", logDir)
	}

	lf, err := os.OpenFile(f, os.O_CREATE|os.O_APPEND|os.O_RDWR, 06666)
	if err != nil {
		log.Fatalf("could not open %s for logging: %v\n", f, err)
	}

	return lf, nil
}
