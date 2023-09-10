package ureq

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GetIndex searches value from []interface{} by index
func GetIndex(v interface{}, index int) interface{} {
	switch res := v.(type) {
	case []interface{}:
		if len(res) > index {
			return res[index]
		}
		return nil
	case *[]interface{}:
		return GetIndex(*res, index)
	default:
		return nil
	}
}

// GetPath searches value from map[string]interface{} by path
func GetPath(v interface{}, branch ...string) interface{} {
	switch res := v.(type) {
	case map[string]interface{}:
		switch len(branch) {
		case 0:
			return nil // should return nil when no branch
		case 1:
			return res[branch[0]]
		default:
			return GetPath(res[branch[0]], branch[1:]...)
		}
	case *map[string]interface{}:
		return GetPath(*res, branch...)
	default:
		return nil
	}
}

// Response represents the response from a HTTP request.
type Response struct {
	*http.Response

	raw     *bytes.Buffer
	content []byte
}

// Raw returns the raw bytes body of the response.
func (r *Response) RawBufferr() (*bytes.Buffer, error) {
	if r.raw == nil {
		r.raw = &bytes.Buffer{}
		if _, e := io.Copy(r.raw, r.Body); e != nil {
			return nil, e
		}

		r.Body.Close()
	}

	return r.raw, nil
}

func (r *Response) Raw() ([]byte, error) {
	buf, e := r.RawBufferr()
	if e != nil {
		return nil, e
	}

	return buf.Bytes(), nil
}

// Content returns the content of the response body, it will handle
// the compression.
func (r *Response) Content() ([]byte, error) {
	if r.content != nil {
		return r.content, nil
	}

	rawBuf, e := r.RawBufferr()
	if e != nil {
		return nil, e
	}

	var reader io.Reader

	switch r.Header.Get(ContentEncoding) {
	case "gzip":
		rr, e := gzip.NewReader(rawBuf)
		if e != nil {
			return nil, e
		}

		defer rr.Close()
		reader = rr
	case "deflate":
		// deflate should be zlib
		// http://www.gzip.org/zlib/zlib_faq.html#faq38
		rr, e := zlib.NewReader(rawBuf)
		if e != nil {
			// try RFC 1951 deflate
			// http: //www.open-open.com/lib/view/open1460866410410.html
			rr = flate.NewReader(rawBuf)
		}

		defer rr.Close()
		reader = rr
	default:
		reader = rawBuf
	}

	r.content, e = io.ReadAll(reader)
	return r.content, e
}

// JSON returns the reponse body with JSON format.
func (r *Response) JSON(res interface{}) error {
	b, e := r.Content()
	if e != nil {
		return e
	}

	if !strings.HasPrefix(r.Header.Get(ContentType), "application/json") {
		if len(b) > 0 {
			return errors.New(string(b))
		}

		return errors.New(r.Status)
	}

	if e := json.Unmarshal(b, res); e != nil {
		return e
	}

	if !r.OK() {
		return ErrStatusNotOk
	}

	return nil
}

// Text returns the reponse body with text format.
func (r *Response) Text() (string, error) {
	b, e := r.Content()
	if e != nil {
		return "", e
	}

	if !r.OK() {
		return string(b), ErrStatusNotOk
	}

	return string(b), nil
}

// URL returns url of the final request.
func (r *Response) URL() (*url.URL, error) {
	u := r.Request.URL

	if r.StatusCode == http.StatusMovedPermanently ||
		r.StatusCode == http.StatusFound ||
		r.StatusCode == http.StatusSeeOther ||
		r.StatusCode == http.StatusTemporaryRedirect {
		location, err := r.Location()
		if err != nil {
			return nil, err
		}

		u = u.ResolveReference(location)
	}

	return u, nil
}

// Reason returns the status text of the response status code.
func (r *Response) Reason() string {
	return http.StatusText(r.StatusCode)
}

// OK returns whether the reponse status code is less than 400.
func (r *Response) OK() bool {
	return r.StatusCode < 400
}
