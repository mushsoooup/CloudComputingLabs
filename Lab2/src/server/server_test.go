package server

import (
	"bufio"
	"main/http"
	"net"
	"testing"
	"time"
)

func TestRequestParse(t *testing.T) {
	s := Server{}
	s.RegisterHandler(func(rc *RequestCtx) error {
		method := rc.Req.GetMethod()
		path := rc.Req.GetPath()
		host := rc.Req.GetHeader("Host")
		hello := rc.Req.GetHeader("Hello")
		if method != "GET" {
			t.Fatalf("wrong method %v", method)
		}
		if path != "/index.html" {
			t.Fatalf("wrong path %v", path)
		}
		if host != "127.0.0.1:65500" {
			t.Fatalf("wrong host %v", host)
		}
		if hello != "World!" {
			t.Fatalf("wrong hello %v", hello)
		}
		rc.c.Write(http.FormatResponse([]byte("this is the reply"), 200, "Content-type: text/plain"))
		return nil
	})
	ln, err := net.Listen("tcp", ":65500")
	if err != nil {
		t.Fatalf("error listening port 65500 %v", err)
	}
	go s.Serve(ln)
	client, err := net.Dial("tcp", ":65500")
	if err != nil {
		t.Fatalf("error dial port 65500 %v", err)
	}
	_, err = client.Write([]byte("GET /index.html HTTP/1.1\r\nHost: 127.0.0.1:65500\r\nHello: World!\r\n\r\n"))
	if err != nil {
		t.Fatalf("error write to port 65500 %v", err)
	}
	data := make([]byte, 0)
	client.Read(data)
	r := bufio.NewReader(client)
	scanner := bufio.NewScanner(r)
	scanner.Split(scanCRLF)
	for scanner.Scan() {
		t.Log(scanner.Text())
	}
	time.Sleep(1 * time.Second)
}
