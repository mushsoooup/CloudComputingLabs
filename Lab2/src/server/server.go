package server

import (
	"bufio"
	"main/http"
	"main/logger"
	"main/pool"
	"net"
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

var unavailable = http.FormatResponse([]byte("503 Service Unavailable"), 503, "Content-type: text/plain")

var idleTimeout = 5 * time.Second

// Serve starts the server
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
	// Supports http pipelining
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
	s.releaseCtx(ctx)
	s.releaseReader(reader)
	return err
}

func (s *Server) acquireCtx(c net.Conn) *RequestCtx {
	ctx := s.ctxPool.Get().(*RequestCtx)
	ctx.s = s
	ctx.c = c
	return ctx
}
func (s *Server) releaseCtx(ctx *RequestCtx) {
	ctx.Reset()
	s.ctxPool.Put(ctx)
}

func (s *Server) acquireReader(ctx *RequestCtx) *bufio.Reader {
	r := ctx.s.readerPool.Get().(*bufio.Reader)
	r.Reset(ctx.c)
	return r
}

func (s *Server) releaseReader(r *bufio.Reader) {
	s.readerPool.Put(r)
}

func (ctx *RequestCtx) Reset() {
	ctx.c = nil
	ctx.s = nil
	ctx.Req.Reset()
	ctx.Res.Reset()
}

func (s *Server) RegisterHandler(handler HandlerFunc) {
	s.handler = handler
}
