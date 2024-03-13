package main

import (
	"main/router"
)

func main() {
	r := router.Router{}
	r.LoadStatic("/config/CloudComputingLabs/Lab2/static")
	r.Serve(":65500")
}
