package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// hello world, the web server
func HelloServer(w http.ResponseWriter, req *http.Request) {
	contents, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", string(contents))
	io.WriteString(w, `{"code": 0, "data":0 }`)
}

func main() {
	http.HandleFunc("/", HelloServer)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
