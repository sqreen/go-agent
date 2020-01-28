package main

import (
	"io"
	"log"
	"net/http"

	"github.com/sqreen/go-agent/sdk/middleware/sqhttp"
)

func main() {
	http.Handle("/", sqhttp.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello HTTP!")
	})))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalln(err)
	}
}
