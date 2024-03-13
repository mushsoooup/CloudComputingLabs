package http

import (
	"log"
	"testing"
)

func TestFormatResponse(t *testing.T) {
	log.Printf("%s\n", FormatResponse([]byte("hello, http!"), 200, "Content-type: text/plain"))
}

func TestParams(t *testing.T) {
	r := Request{}
	r.SetPath([]byte("/abc/cde?a=c&b=d"))
	if r.GetPath() != "/abc/cde" {
		log.Printf("wrong path /abc/cde -> %v\n", r.GetPath())
	}
	if r.GetParam("a") != "c" {
		log.Printf("wrong param c -> %v\n", r.GetParam("a"))
	}
	if r.GetParam("b") != "d" {
		log.Printf("wrong param d -> %v\n", r.GetParam("b"))
	}
}
