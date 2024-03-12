package http

import (
	"strconv"
	"strings"
)

func StatusMessage(status int64) []byte {
	switch status {
	case 200:
		return []byte(" OK")
	case 404:
		return []byte(" Not Found")
	case 501:
		return []byte(" Not Implemented")
	case 503:
		return []byte(" Service Unavailable")
	default:
		return []byte(" Unknown")
	}
}

// Helper function to send back response
func FormatResponse(data []byte, status int64, headers ...string) []byte {
	result := make([]byte, 0, len(data))
	result = append(result, []byte("HTTP/1.1 ")...)
	result = strconv.AppendInt(result, status, 10)
	result = append(result, StatusMessage(status)...)
	result = append(result, []byte("\r\nServer: GO SERVER\r\n")...)
	for _, header := range headers {
		result = append(result, []byte(header)...)
		result = append(result, []byte("\r\n")...)
	}
	result = append(result, []byte("Content-Length: ")...)
	result = strconv.AppendInt(result, int64(len(data)), 10)
	result = append(result, []byte("\r\n\r\n")...)
	result = append(result, data...)
	return result
}

type Request struct {
	method  string
	path    string
	params  map[string]string
	headers map[string]string
	data    []byte
}

func (r *Request) Reset() {
	r.method = ""
	r.path = ""
	r.params = make(map[string]string)
	r.headers = make(map[string]string)
	r.data = make([]byte, 0)
}

func (r *Request) SetMethod(method string) {
	r.method = method
}
func (r *Request) SetPath(path string) {
	idx := strings.Index(path, "?")
	if idx == -1 {
		r.path = path
		return
	}
	r.path = path[:idx]
	params := strings.Split(path[idx+1:], "&")
	if r.params == nil {
		r.params = make(map[string]string)
	}
	for _, param := range params {
		idx := strings.Index(param, "=")
		if idx == -1 {
			continue
		}
		r.params[param[:idx]] = param[idx+1:]
	}
}
func (r *Request) SetData(data []byte) {
	r.data = data
}
func (r *Request) AddHeader(key, val string) {
	if r.headers == nil {
		r.headers = make(map[string]string)
		r.headers[key] = val
	}
}
func (r *Request) GetMethod() string {
	return r.method
}
func (r *Request) GetPath() string {
	return r.path
}
func (r *Request) GetParam(key string) string {
	return r.params[key]
}
func (r *Request) GetData() []byte {
	return r.data
}
func (r *Request) GetHeader(key string) string {
	return r.headers[key]
}

type Response struct {
	headers     map[string]string
	data        []byte
	status      int64
	contentType string
}

func (r *Response) Reset() {
	r.headers = make(map[string]string)
	r.data = make([]byte, 0)
	r.status = 0
	r.contentType = ""
}
func (r *Response) SetContentType(contentType string) {
	r.contentType = contentType
}

func (r *Response) AddHeader(key, val string) {
	if r.headers == nil {
		r.headers = make(map[string]string)
	}
	r.headers[key] = val
}

func (r *Response) SetData(data []byte, status int) {
	r.status = int64(status)
	r.data = data
}

func (r *Response) Prepare() []byte {
	if r.status == 0 {
		r.status = 200
	}
	if r.contentType == "" {
		r.contentType = "text/plain"
	}
	r.AddHeader("Content-Length", strconv.FormatInt(int64(len(r.data)), 10))
	result := make([]byte, 0, len(r.data))
	result = append(result, []byte("HTTP/1.1 ")...)
	result = strconv.AppendInt(result, r.status, 10)
	result = append(result, StatusMessage(r.status)...)
	result = append(result, []byte("\r\nServer: GO SERVER\r\n")...)
	for key, val := range r.headers {
		result = append(result, []byte(key+": "+val+"\r\n")...)
	}
	result = append(result, []byte("\r\n")...)
	result = append(result, r.data...)
	return result
}
