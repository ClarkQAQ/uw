package ulog

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"
)

type RotateWriter struct {
	root        string
	fileFormat  string
	currentName string
	current     io.WriteCloser
	intervalDay int
	compress    bool
	rw          sync.RWMutex
	exited      chan struct{}
}

func NewRotateWriter(dir, fileFormat string, intervalDay int, compress ...bool) (*RotateWriter, error) {
	if intervalDay < 1 {
		return nil, fmt.Errorf("intervalDay must be >= 1")
	}

	l := &RotateWriter{
		root:        dir,
		fileFormat:  fileFormat,
		intervalDay: intervalDay,
		exited:      make(chan struct{}),
	}

	if len(compress) > 0 {
		l.compress = compress[0]
	}

	if e := l.setup(); e != nil {
		return nil, e
	}

	return l, nil
}

func (r *RotateWriter) Write(p []byte) (int, error) {
	r.rw.RLock()
	defer r.rw.RUnlock()

	if r.current == nil {
		return 0, errors.New("rotate writer is closed")
	}

	return r.current.Write(p)
}

func (r *RotateWriter) WriteString(s string) (int, error) {
	return r.Write([]byte(s))
}

func (r *RotateWriter) Close() error {
	r.rw.Lock()
	defer r.rw.Unlock()

	if err := r.current.Close(); err != nil {
		return err
	}
	r.current = nil
	r.exited <- struct{}{}
	defer close(r.exited)

	return nil
}

func (r *RotateWriter) setup() error {
	fi, e := os.Stat(r.root)
	if e != nil && os.IsNotExist(e) {
		if e := os.MkdirAll(r.root, os.ModePerm); e != nil {
			return e
		}
	} else if e != nil {
		return e
	} else if !fi.IsDir() {
		return errors.New("root must be a directory")
	}

	go func(r *RotateWriter) {
		t := time.NewTimer(r.getNextIntervalDuration())
		defer t.Stop()

		for {
			select {
			case <-r.exited:
				return
			case <-t.C:
				time.Sleep(time.Second)

				if e := r.openCurrent(); e != nil {
					os.Stderr.WriteString("log rotate: " + e.Error() + "\n")
				}
			}

			t.Reset(r.getNextIntervalDuration())
		}
	}(r)

	return r.openCurrent()
}

func (r *RotateWriter) getNextIntervalDuration() time.Duration {
	nt := time.Now()
	nt = time.Date(nt.Year(), nt.Month(), nt.Day(), 0, 0, 0, 0, nt.Location())
	nt = nt.AddDate(0, 0, r.intervalDay)

	return time.Until(nt)
}

func (r *RotateWriter) openCurrent() (e error) {
	r.rw.Lock()
	defer r.rw.Unlock()

	nextName := time.Now().Format(r.fileFormat)
	if nextName == r.currentName {
		return nil
	}

	if r.current != nil {
		if e := r.current.Close(); e != nil {
			return e
		}

		if r.currentName != "" && r.compress {
			if e := r.compressCurrent(); e != nil {
				os.Stderr.WriteString("log rotate: " + e.Error() + "\n")
			} else {
				os.Remove(path.Join(r.root, r.currentName))
			}
		}
	}

	r.currentName = nextName
	r.current, e = os.OpenFile(path.Join(r.root, r.currentName),
		os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_SYNC, os.ModePerm)
	return
}

func (r *RotateWriter) OpenCurrent() error {
	return r.openCurrent()
}

func (r *RotateWriter) compressCurrent() error {
	f, e := os.Open(path.Join(r.root, r.currentName))
	if e != nil {
		return e
	}

	defer f.Close()

	w, e := os.OpenFile(path.Join(r.root, r.currentName+".gz"),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if e != nil {
		return e
	}
	defer w.Close()

	gz, e := gzip.NewWriterLevel(w, gzip.BestCompression)
	if e != nil {
		return e
	}
	defer gz.Close()

	gz.Comment = fmt.Sprintf("rotate compressed at %s", time.Now().Format(time.RFC3339))

	if _, e := io.Copy(gz, f); e != nil {
		return e
	}

	if e := gz.Flush(); e != nil {
		return e
	}

	return nil
}
