// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/sqreen/go-agent/examples/hellohttp"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalln("unexpected empty port number from environment variable `PORT`")
	}
	hellohttp.ListenAndServe(fmt.Sprintf(":%s", port))
}
