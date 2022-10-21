package httpreader

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// A Reader implements the io.Reader, io.ReaderAt, io.Seeker
// interfaces by reading from a http response body.
type Reader struct {
	//the current reading offset
	off int64
	//the size of the associated URL resource
	Length int64
	//HeadBytes is the first bytes and max size is 512
	HeadBytes []byte

	URL    *url.URL
	client *http.Client
	resp   *http.Response
	//count of http requests
	Count int
	//http request header
	Header  http.Header
	ifRange string
	discard []byte
}
type Option func(option *Reader)

// Specify the http client
func WithClient(client *http.Client) Option {
	return func(r *Reader) {
		r.client = client
	}
}

// Specify the http request header
func WithHeader(header http.Header) Option {
	return func(r *Reader) {
		r.Header = header
	}
}

// Specify the max discard size.
// Reader try to reuse the http response body according to the parameter
func WithDiscard(maxDiscard int) Option {
	return func(r *Reader) {
		r.discard = make([]byte, maxDiscard)
	}
}

// ReadAt reads len(p) bytes from the ranged-over source.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b). At end of file, that error is io.EOF.
func (r *Reader) ReadAt(p []byte, off int64) (int, error) {
	_, err := r.Seek(off, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return r.Read(p)
}

// Read reads len(p) bytes from ranged-over source.
// It returns the number of bytes read and the error, if any.
// EOF is signaled by a zero count with err set to io.EOF.
func (r *Reader) Read(p []byte) (int, error) {
	if r.off >= r.Length {
		return 0, io.EOF
	}
	if r.resp == nil {
		err := r.request()
		if err != nil {
			return 0, err
		}
	}
	n, err := r.resp.Body.Read(p)
	r.off = r.off + int64(n)
	return n, err
}

// Seek sets the offset for the next Read to offset, interpreted
// according to whence: 0 means relative to the origin of the file, 1 means relative
// to the current offset, and 2 means relative to the end. It returns the new offset
// and an error, if any.
func (r *Reader) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case 0: // set
	case 1: // cur
		off = r.off + off
	case 2: // end
		off = r.Length + off
	}

	if off > r.Length {
		return 0, errors.New("seek beyond end of file")
	}

	if off < 0 {
		return 0, errors.New("seek before beginning of file")
	}

	length := off - r.off
	if length <= int64(len(r.discard)) && length >= 0 {
		//try to reuse the http response body
		n, err := r.Read(r.discard[:length])
		if n != int(length) {
			return r.off, errors.New("discard bytes error")
		}
		if err != nil && err != io.EOF {
			return 0, err
		}
	} else {
		r.off = off
		err := r.request()
		if err != nil {
			return 0, err
		}
	}
	return r.off, nil
}

// Close the associated http response body.
// It is the caller's responsibility to close the Reader
func (r *Reader) Close() error {
	if r.resp != nil {
		return r.resp.Body.Close()
	}
	return nil
}

func (r *Reader) request() error {
	if r.resp != nil {
		r.resp.Body.Close()
	}
	req := &http.Request{
		Method: "GET",
		URL:    r.URL,
		Header: http.Header{
			"Range":    []string{fmt.Sprintf("bytes=%d-", r.off)},
			"If-Range": []string{r.ifRange},
		},
	}
	resp, err := r.client.Do(req)
	r.Count++
	if err != nil {
		return err
	}
	if resp.StatusCode != 206 {
		return errors.New("not partical content or resource changed")
	}
	r.resp = resp
	return nil
}

func (r *Reader) init() error {
	req, err := http.NewRequest(http.MethodGet, r.URL.String(), nil)
	if err != nil {
		return err
	}
	if len(r.Header) > 0 {
		req.Header = r.Header
	}
	//first 512 bytes
	req.Header.Add("Range", "bytes=0-511")
	resp, err := r.client.Do(req)
	r.Count++
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data := make([]byte, 512)
	n, err := resp.Body.Read(data)
	if err != nil && err != io.EOF {
		return err
	}
	r.HeadBytes = data[:n]
	if !statusIsAcceptable(resp.StatusCode) {
		return fmt.Errorf("unexpected response (status %d)", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Accept-Ranges"), "bytes") {
		return errors.New(r.URL.String() + " does not support byte-ranged requests.")
	}

	validator, err := validatorFromResponse(resp)
	if err != nil {
		return errors.New(r.URL.String() + " did not offer a strong-enough validator for subsequent requests")
	}
	str := resp.Header.Get("Content-Range")
	if strings.Contains(str, "/") {
		length, err := strconv.ParseInt(strings.Split(str, "/")[1], 10, 64)
		if err != nil {
			return errors.New(r.URL.String() + "invalid response header Content-Range " + str)
		}
		r.Length = length
		r.ifRange = validator
		return nil
	} else {
		return errors.New(r.URL.String() + "invalid response header Content-Range " + str)
	}
}

func statusIsAcceptable(status int) bool {
	return status >= 200 && status < 300
}

func validatorFromResponse(resp *http.Response) (string, error) {
	etag := resp.Header.Get("ETag")
	if etag != "" && etag[0] == '"' {
		return etag, nil
	}

	modtime := resp.Header.Get("Last-Modified")
	if modtime != "" {
		return modtime, nil
	}

	return "", errors.New("no applicable validator in response")
}

// NewReader returns a newly-initialized Reader,
// which also try to fetch the first 512 bytes
// It returns the new reader and an error, if any.
func NewReader(u *url.URL, opts ...Option) (*Reader, error) {
	reader := &Reader{
		URL:     u,
		client:  http.DefaultClient,
		discard: make([]byte, 1024*4),
		Count:   0,
	}
	for _, o := range opts {
		o(reader)
	}
	return reader, reader.init()
}
