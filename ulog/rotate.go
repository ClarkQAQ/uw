package ulog

// fork from https://github.com/stathat/rotate

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxDefault  = 1024 * 1024 * 8
	keepDefault = 10
)

// RootPerm defines the permissions that Writer will use if it
// needs to create the root directory.
var RootPerm = os.FileMode(0o755)

// FilePerm defines the permissions that Writer will use for all
// the files it creates.
var FilePerm = os.FileMode(0o666)

// Writer implements the io.Writer interface and writes to the
// "current" file in the root directory.  When current's size
// exceeds max, it is renamed and a new file is created.
type RotateWriter struct {
	root    string
	prefix  string
	current *os.File
	size    int
	max     int
	keep    int
	sync.Mutex
}

// New creates a new Writer.  The files will be created in the
// root directory.  root will be created if necessary.  The
// filenames will start with prefix.
func NewRotateWriter(root string, prefix string) (*RotateWriter, error) {
	l := &RotateWriter{root: root, prefix: prefix, max: maxDefault, keep: keepDefault}
	if err := l.setup(); err != nil {
		return nil, err
	}
	return l, nil
}

// SetMax sets the maximum size for a file in bytes.
func (r *RotateWriter) SetMax(size int) {
	r.max = size
}

// SetKeep sets the number of archived files to keep.
func (r *RotateWriter) SetKeep(n int) {
	r.keep = n
}

// Write writes p to the current file, then checks to see if
// rotation is necessary.
func (r *RotateWriter) Write(p []byte) (n int, err error) {
	r.Lock()
	defer r.Unlock()
	n, err = r.current.Write(p)
	if err != nil {
		return n, err
	}
	r.size += n
	if r.size >= r.max {
		if err := r.rotate(); err != nil {
			return n, err
		}
	}
	return n, nil
}

// Close closes the current file.  Writer is unusable after this
// is called.
func (r *RotateWriter) Close() error {
	r.Lock()
	defer r.Unlock()
	if err := r.current.Close(); err != nil {
		return err
	}
	r.current = nil
	return nil
}

// setup creates the root directory if necessary, then opens the
// current file.
func (r *RotateWriter) setup() error {
	fi, err := os.Stat(r.root)
	if err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(r.root, RootPerm)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if !fi.IsDir() {
		return errors.New("root must be a directory")
	}

	// root exists, and it is a directory

	return r.openCurrent()
}

func (r *RotateWriter) openCurrent() error {
	cp := path.Join(r.root, "current")
	var err error
	r.current, err = os.OpenFile(cp, os.O_RDWR|os.O_CREATE|os.O_APPEND, FilePerm)
	if err != nil {
		return err
	}
	r.size = 0
	return nil
}

func (r *RotateWriter) rotate() error {
	if err := r.current.Close(); err != nil {
		return err
	}
	filename := fmt.Sprintf("%s_%d", r.prefix, time.Now().UnixNano())
	if err := os.Rename(path.Join(r.root, "current"), path.Join(r.root, filename)); err != nil {
		return err
	}
	if err := r.clean(); err != nil {
		return err
	}
	return r.openCurrent()
}

func (r *RotateWriter) clean() error {
	d, err := os.Open(r.root)
	if err != nil {
		return err
	}
	names, err := d.Readdirnames(1024)
	if err != nil {
		return err
	}
	var archNames []string
	for _, n := range names {
		if strings.HasPrefix(n, r.prefix+"_") {
			archNames = append(archNames, n)
		}
	}
	if len(archNames) <= r.keep {
		return nil
	}

	sort.Strings(archNames)
	toDel := archNames[0 : len(archNames)-r.keep]
	for _, n := range toDel {
		if err := os.Remove(path.Join(r.root, n)); err != nil {
			return err
		}
	}
	return nil
}
