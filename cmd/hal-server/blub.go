package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func hello(w http.ResponseWriter, req *http.Request) {
	buf, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Println(err)
	}

	log.Println(string(buf))
}

func main() {
	http.HandleFunc("/fence", hello)
	http.ListenAndServe(":8080", nil)

}
