// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package hellohttp

import (
	"io"
	"log"
	"net/http"

	"github.com/sqreen/go-agent/sdk/middleware/sqhttp"
)

func ListenAndServe(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", http.HandlerFunc(helloHandler))
	if err := http.ListenAndServe(addr, sqhttp.Middleware(mux)); err != nil {
		log.Fatalln(err)
	}
}

func helloHandler(w http.ResponseWriter, _ *http.Request) {
	if _, err := io.WriteString(w, "Hello HTTP!"); err != nil {
		log.Println(err)
	}
}