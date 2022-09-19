httpreader
============
![GitHub](https://img.shields.io/github/license/Heng-Bian/httpreader)
![GitHub](https://img.shields.io/badge/build-pass-green)
![GitHub](https://img.shields.io/badge/coverage-81.9%25-green)
![GitHub](https://img.shields.io/badge/tests-12%2F12%20tests%20passed-green)

Go package httpreader implements io.ReaderAt, io.Reader, and io.Seeker depending on HTTP Range Requests.

Package httpreader is the most efficient. Unlike others, it makes HTTP Requests only when needed.  
Httpreader reuses `http.Response.Body` if the reading is sequential.  
Httpreader makes a HTTP Request when the reading offset is changed Non-sequential such as calling Seeker method to skip large data.  

There is no need to buffer the httpreader because the underlying layer of httpreader is `http.Response.Body`.  
The number of HTTP Requests only depends on the degree of sequential reading.  

It can be used for example with "archive/zip" package in Go standard
library. Together they can be used to access remote (HTTP accessible)
ZIP archives without needing to download the whole archive file.

HTTP Range Requests (see [RFC 7233](https://tools.ietf.org/html/rfc7233))
are used to retrieve the requested byte range.

Example
-------

The following example outputs a file list of a remote zip archive without
downloading the whole archive:

```Go
package main

import (
	"archive/zip"
	"fmt"
	"github.com/Heng-Bian/httpreader"
	"net/url"
)

func main() {
	u, _ := url.Parse("https://golang.google.cn/dl/go1.19.1.windows-amd64.zip")
	r, err := httpreader.NewReader(u)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	defer r.Close()
	zipReader, err := zip.NewReader(r, r.Length)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	for _, f := range zipReader.File {
		fmt.Println(f.Name)
	}
}
```

License
-------

MIT