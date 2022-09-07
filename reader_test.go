package httpreader

import (
	"archive/zip"
	"fmt"
	"net/url"
	"testing"
)

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
