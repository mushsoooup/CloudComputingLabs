package router

import (
	"io"
	"io/fs"
	"main/server"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type routeEntry struct {
	handler     server.HandlerFunc
	allowMethod string
}

type Router struct {
	routeMap map[string]*routeEntry
}

func (r *Router) Serve(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}
	server := server.Server{}
	server.RegisterHandler(r.handler)
	server.Serve(ln)
}

// Please load static so error pages function
func (r *Router) handler(ctx *server.RequestCtx) error {
	entry, ok := r.routeMap[ctx.Req.GetPath()]
	if !ok {
		entry = r.routeMap["/404.html"]
	} else if entry.allowMethod != ctx.Req.GetMethod() {
		entry = r.routeMap["/501.html"]
	}
	return entry.handler(ctx)
}

// Register handler
func (r *Router) Register(path string, method string, handler server.HandlerFunc) {
	if r.routeMap == nil {
		r.routeMap = make(map[string]*routeEntry)
	}
	r.routeMap[path] = &routeEntry{
		handler:     handler,
		allowMethod: method,
	}
}

// Load static folder and resgister files to specific path
func (r *Router) LoadStatic(static, path string) error {
	err := filepath.Walk(static, func(file string, info fs.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		idx := strings.LastIndex(info.Name(), ".")
		if idx == -1 {
			return nil
		}
		extension := info.Name()[idx+1:]
		var contentType string
		switch extension {
		case "html":
			fallthrough
		case "css":
			contentType = "text/" + extension
		case "js":
			contentType = "text/javascript"
		case "json":
			contentType = "application/json"
		default:
			return nil
		}
		r.Register(path+info.Name(), "GET", r.serveFile(file, contentType, info.Name()))
		return nil
	})
	return err
}

// Serve static file
func (r *Router) serveFile(path string, contentType string, name string) server.HandlerFunc {
	return func(ctx *server.RequestCtx) error {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		ctx.Res.SetContentType(contentType)
		content, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		var status int
		switch name {
		case "404.html":
			status = 404
		case "501.html":
			status = 501
		case "503.html":
			status = 503
		default:
			status = 200
		}
		ctx.Res.SetData(content, status)
		return nil
	}
}
