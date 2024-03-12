package server

import (
	"bufio"
	"errors"
	"io"
	"main/http"
	"main/logger"
	"main/pool"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HandlerFunc func(*RequestCtx) error

type Server struct {
	p          pool.Pool
	ctxPool    sync.Pool
	readerPool sync.Pool
	handler    HandlerFunc
}

type RequestCtx struct {
	c   net.Conn
	s   *Server
	Req http.Request
	Res http.Response
}

var unavailable = http.FormatResponse([]byte("503 Service Unavailable"), 503, "Content-type: text/plain")

var idleTimeout = 5 * time.Second

func (s *Server) Serve(ln net.Listener) {
	s.ctxPool.New = func() any {
		return &RequestCtx{}
	}
	s.readerPool.New = func() any {
		return &bufio.Reader{}
	}

	s.p.Start(s.serveConn, 256*1024)
	for {
		c, err := ln.Accept()
		if err != nil {
			logger.Debug("error establishing connection %v", err)
			continue
		}
		if !s.p.Serve(c) {
			logger.Debug("Drop connection %v", err)
			c.SetWriteDeadline(time.Now().Add(50 * time.Millisecond))
			c.Write(unavailable)
			c.Close()
		}
	}
}

func (s *Server) serveConn(c net.Conn) (err error) {
	ctx := s.acquireCtx(c)
	reader := s.acquireReader(ctx)
	for {
		if err = c.SetReadDeadline(time.Now().Add(idleTimeout)); err != nil {
			break
		}
		_, err = reader.Peek(1)
		if err != nil {
			break
		}
		err = ctx.parseData(reader)
		if err != nil {
			break
		}
		err = s.handler(ctx)
		if err != nil {
			c.Write(unavailable)
		} else {
			c.SetWriteDeadline(time.Now().Add(idleTimeout))
			_, err = c.Write(ctx.Res.Prepare())
		}
		if err != nil {
			break
		}
		ctx.Req.Reset()
		ctx.Res.Reset()
	}
	return err
}

func (s *Server) acquireCtx(c net.Conn) *RequestCtx {
	ctx := s.ctxPool.Get().(*RequestCtx)
	ctx.s = s
	ctx.c = c
	return ctx
}

func (s *Server) acquireReader(ctx *RequestCtx) *bufio.Reader {
	r := ctx.s.readerPool.Get().(*bufio.Reader)
	r.Reset(ctx.c)
	return r
}

func (ctx *RequestCtx) parseData(r *bufio.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Split(scanCRLF)
	// Parse first line
	if scanner.Scan() {
		line := scanner.Text()
		strs := strings.Split(line, " ")
		if len(strs) != 3 || strs[2] != "HTTP/1.1" {
			return errors.New("error parsing request")
		}
		ctx.Req.SetMethod(strs[0])
		ctx.Req.SetPath(strs[1])
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	// Parse headers
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		idx := strings.Index(line, ": ")
		if idx == -1 {
			return errors.New("error parsing request")
		}
		ctx.Req.AddHeader(line[0:idx], line[idx+2:])
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	// Check content length (if there is)
	length, err := strconv.Atoi(ctx.Req.GetHeader("Content-Length"))
	if err != nil {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(r, int64(length)))
	if err != nil {
		return errors.New("error parsing request")
	}
	ctx.Req.SetData(data)
	return nil
}

func scanCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := strings.Index(string(data), "\r\n"); i >= 0 {
		return i + 2, data[0:i], nil
	}

	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func (s *Server) RegisterHandler(handler HandlerFunc) {
	s.handler = handler
}
