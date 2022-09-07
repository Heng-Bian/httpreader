package httpreader

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

var server *httptest.Server
var now = time.Now()

func TestMain(m *testing.M) {
	content := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	reader := bytes.NewReader([]byte(content))

	handler := func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "", now, reader)
	}
	server = httptest.NewServer(http.HandlerFunc(handler))
	os.Exit(m.Run())
}
func reader() (*Reader, error) {
	u, err := url.Parse(server.URL)
	if err != nil {
		return nil, err
	}
	return NewReader(u, WithDiscard(3))
}
func TestZip(t *testing.T) {
	u, _ := url.Parse("https://golang.google.cn/dl/go1.19.1.windows-amd64.zip")
	r, err := NewReader(u)
	if err != nil {
		t.Error(err)
	}
	zipReader, err := zip.NewReader(r, r.Length)
	if err != nil {
		t.Error(err)
	}
	for _, value := range zipReader.File {
		fmt.Printf("%v\n", value.Name)
	}
}
func TestReadFirstOne(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	buf := make([]byte, 1)
	n, err := r.Read(buf)
	if err != nil {
		t.Error(err)
	}
	if n != 1 && buf[0] != 'A' {
		t.Error(buf[0])
	}
}
func TestReadAtLast(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	buf := make([]byte, 1)
	n, err := r.ReadAt(buf, 25)
	if err != io.EOF {
		t.Error(err)
	}
	if n != 1 && buf[0] != 'Z' {
		t.Error(buf[0])
	}
}
