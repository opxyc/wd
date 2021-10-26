package logger

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// NewDailyLogger creates a new Logger that logs to a file with name YYYY-MM-DD
// in the given directory dir. It swiches to a new file everyday at given hour h
// and minute m. The outs variable denotes other destinations to which log data
// should be written apart from the file. It stops logging when ctx is closed.
func NewDailyLogger(ctx context.Context, dir string, h int, m int, outs ...io.Writer) (*log.Logger, error) {
	err := prepareDir(dir)
	if err != nil {
		return nil, err
	}

	// try to open a file in dir
	f, err := openFile(dir)
	if err != nil {
		return nil, err
	}

	// create a new logger
	l := log.New(mw(f, outs...), "", log.LstdFlags)

	// calculate repeat interval
	t := time.Now()
	trgtTime := time.Date(t.Year(), t.Month(), t.Day(), h, m, 0, 0, t.Location())
	d := trgtTime.Sub(t)
	if d < 0 {
		trgtTime = trgtTime.Add(24 * time.Hour)
		d = trgtTime.Sub(t)
	}

	// and repeat
	go repeat(ctx, d, dir, l, outs...)

	return l, nil
}

func repeat(ctx context.Context, d time.Duration, dir string, l *log.Logger, outs ...io.Writer) {
	var prevf, f *os.File
	prevf = f
	for {
		select {
		case <-ctx.Done():
			prevf.Close()
			// err := prevf.Close()
			// if err != nil {
			// 	fmt.Printf("could not close prev file '%v': %v\n", prevf.Name(), err)
			// }
			// fmt.Printf("closed prev file '%v'\n", prevf.Name())

			f.Close()
			// err = f.Close()
			// if err != nil {
			// 	fmt.Printf("could not close curr log file '%v': %v\n", prevf.Name(), err)
			// }
			// fmt.Printf("closed curr log file '%v'\n", prevf.Name())

			return
		case <-time.After(d):
			// set duration to 24 hours to that next file will be created
			// After(duration d)
			d = 24 * time.Hour
			// open another file
			f, err := openFile(dir)
			if err != nil {
				return
			}
			l.SetOutput(mw(f, outs...))
			// close previous log file
			prevf.Close()
			prevf = f
		}
	}
}

// openFile opens a file with name = today's date in the given dir
func openFile(dir string) (*os.File, error) {
	const format = "2006-Jan-02"
	t := time.Now()

	// check if a file with today's date exists
	// if yes, return it
	fname := filepath.Join(dir, t.Format(format))

	f, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_RDWR, 06666)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// prepareDir tries to create dir if not present and
// assigns permissions to create files in it
func prepareDir(dir string) error {
	// check if dir exists
	_, err := os.Stat(dir)
	if err != nil {
		// try creating directory
		err = os.MkdirAll(dir, 06666)
		if err != nil {
			return err
		}
	}

	// set permission
	err = os.Chmod(dir, 0755)
	if err != nil {
		return err
	}

	return nil
}

// mw combines the given writers to a multiwriter
func mw(f io.Writer, ws ...io.Writer) io.Writer {
	newOuts := []io.Writer{f}
	for _, w := range ws {
		newOuts = append(newOuts, w)
	}
	return io.MultiWriter(newOuts...)
}
