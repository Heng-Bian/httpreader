package httpreader

import (
	"archive/zip"
	"bytes"
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
var content = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func TestMain(m *testing.M) {

	reader := bytes.NewReader([]byte(content))

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/withHeader":
			if r.Header["X-Test-Header"][0] == "401" {
				w.WriteHeader(401)
			} else {
				http.ServeContent(w, r, "", now, reader)
			}
		default:
			http.ServeContent(w, r, "", now, reader)
		}

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
	defer r.Close()
	zipReader, err := zip.NewReader(r, r.Length)
	if err != nil {
		t.Error(err)
	}
	t.Log(zipReader.File[0].Name)

}
func TestReadFirstOne(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
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
	defer r.Close()
	buf := make([]byte, 1)
	n, err := r.ReadAt(buf, 25)
	if err != io.EOF {
		t.Error(err)
	}
	if n != 1 && buf[0] != 'Z' {
		t.Error(buf[0])
	}
}

func TestReadEOF(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	buf := make([]byte, 2)
	n, err := r.ReadAt(buf, 24)
	if n != 2 || buf[0] != 'Y' || buf[1] != 'Z' {
		t.Error("read error")
	}
	if err != io.EOF {
		t.Error("not EOF")
	}
}

func TestReadFull(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	buf := make([]byte, 64)
	n, err := r.Read(buf)
	if n != 26 || buf[0] != 'A' || buf[25] != 'Z' {
		t.Error("read error")
	}
	if err != io.EOF {
		t.Error("not EOF")
	}
}

func TestReadUntilEOF(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	buf := make([]byte, 3)
	data := make([]byte, 64)
	index := 0
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err != io.EOF {
				t.Error(err)
			}
			copy(data[index:], buf[:n])
			index = index + n
			break
		}
		copy(data[index:], buf[:n])
		index = index + n
	}
	if string(data[:index]) != content {
		t.Error(string(data[:index]))
	}
}

func TestCopy(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	data := make([]byte, 0, 64)
	w := bytes.NewBuffer(data)
	n, err := io.Copy(w, r)
	if err != nil || n != 26 {
		t.Error(err)
	}
	if w.String() != content {
		t.Error(w.String())
	}
}

func TestSkipRead(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	data := make([]byte, 1)
	r.Read(data)
	r.Seek(5, io.SeekCurrent)
	r.Read(data)
	if data[0] != 'G' {
		t.Error(data[0])
	}
}

func TestSkipDiscardRead(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	data := make([]byte, 1)
	r.Read(data)
	r.Seek(2, io.SeekCurrent)
	r.Read(data)
	if data[0] != 'D' {
		t.Error(data[0])
	}
}

func TestSeekOutOfBoundary(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	_, err = r.Seek(-1, io.SeekStart)
	if err == nil {
		t.Error("")
	}
	_, err = r.Seek(27, io.SeekStart)
	if err == nil {
		t.Error("")
	}
}

func TestResourceChanged(t *testing.T) {
	r, err := reader()
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	data := make([]byte, 1)
	r.ifRange = "changed"
	_, err = r.Read(data)
	if err == nil {
		t.Error("an error shoule be returned")
	} else {
		t.Log(err)
	}
}

func TestWithHeader(t *testing.T) {
	u, err := url.Parse(server.URL + "/withHeader")
	if err != nil {
		t.Error(err)
	}
	header := make(http.Header, 1)
	header.Add("X-Test-Header", "401")
	_, err = NewReader(u, WithDiscard(3), WithHeader(header))
	if err == nil {
		t.Error("an error shoule be returned")
	} else {
		t.Log(err)
	}
}
