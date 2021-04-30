package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/MrHuangJser/switchgameserver/helper"
)

var port = flag.String("port", "3001", "server port")

func main() {
	flag.Parse()
	http.HandleFunc("/", serverHandler)
	fmt.Printf("Server initialized at %v", *port)
	var error = http.ListenAndServe(fmt.Sprintf(":%v", *port), nil)
	fmt.Printf("%v", error.Error())
}

func serverHandler(resWriter http.ResponseWriter, req *http.Request) {
	err := helper.GetFilesIndex()
	if err != nil {
		resWriter.Write([]byte(err.Error()))
	} else {
		if err != nil {
			resWriter.WriteHeader(500)
			resWriter.Write([]byte(err.Error()))
		} else {
			f, err := os.Open("./hbg.json")
			defer f.Close()
			if err != nil {
				resWriter.WriteHeader(500)
				resWriter.Write([]byte(err.Error()))
			} else {
				io.Copy(resWriter, f)
			}
		}
	}
}
