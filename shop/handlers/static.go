package handlers

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

var (
	defaultFileTimestamp = time.Now()
)

// static file is in memory bytes content with fake os.FileInfo,
// it's implements http.File interface.
type StaticFile struct {
	*bytes.Reader
	io.Closer

	Path      string
	Len       int64
	Timestamp time.Time
}

func NewStaticFile(name string, content []byte, timestamp time.Time) *StaticFile {
	if timestamp.IsZero() {
		timestamp = defaultFileTimestamp
	}
	return &StaticFile{
		Reader:    bytes.NewReader(content),
		Closer:    ioutil.NopCloser(nil),
		Path:      name,
		Timestamp: timestamp,
		Len:       int64(len(content)),
	}
}

func (f *StaticFile) Name() string {
	_, name := filepath.Split(f.Path)
	return name
}

func (f *StaticFile) Mode() os.FileMode {
	return os.FileMode(0644)
}

func (f *StaticFile) ModTime() time.Time {
	return f.Timestamp
}

func (f *StaticFile) Size() int64 {
	return f.Len
}

func (f *StaticFile) IsDir() bool {
	return false
}

func (f *StaticFile) Sys() interface{} {
	return nil
}

func (f *StaticFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("not a directory")
}

func (f *StaticFile) Stat() (os.FileInfo, error) {
	return f, nil
}

// http.FileSystem implementation for bind-fs like function, the timestamp of http.File
// is set to timestamp of the http server started. We never allow directory listing
type StaticFs struct {
	asset  func(string) ([]byte, error)
	prefix string
}

func (fs *StaticFs) Open(name string) (http.File, error) {
	name = path.Join(fs.prefix, filepath.FromSlash(path.Clean("/"+name)))
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	if b, err := fs.asset(name); err == nil {
		return NewStaticFile(name, b, defaultFileTimestamp), nil
	}

	return nil, errors.New("File not found or not allowed")
}
