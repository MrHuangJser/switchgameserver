package main

import (
	"crypto/subtle"
	"flag"
	"fmt"
	"net/http"

	"github.com/MrHuangJser/switchgameserver/helper"
)

var port = flag.String("port", "3001", "server port")
var userName = flag.String("u", "MrHuangJser", "user name")
var password = flag.String("p", "FuckYou123.", "password")

func main() {
	flag.Parse()
	http.HandleFunc("/", serverHandler)
	fmt.Printf("Server initialized at %v", *port)
	var error = http.ListenAndServe(fmt.Sprintf(":%v", *port), nil)
	fmt.Printf("%v", error.Error())
}

func serverHandler(resWriter http.ResponseWriter, req *http.Request) {
	user, passwd, ok := req.BasicAuth()

	if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(*userName)) != 1 || subtle.ConstantTimeCompare([]byte(passwd), []byte(*password)) != 1 {
		resWriter.Header().Set("WWW-Authenticate", `Basic realm="error"`)
		resWriter.WriteHeader(401)
		resWriter.Write([]byte("Unauthorised.\n"))
	} else {
		content, err := helper.GetFilesIndex()
		if err != nil {
			resWriter.Write([]byte(err.Error()))
		} else {
			if err != nil {
				resWriter.WriteHeader(500)
				resWriter.Write([]byte(err.Error()))
			} else {
				resWriter.Write(content)
			}
		}
	}
}
