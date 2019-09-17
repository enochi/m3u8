package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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

type EtcdPut struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Lease string `json:"lease,omitempty"`
}

var count = 0

func etcdLoad(w http.ResponseWriter, req *http.Request) {
	key := "/zengym/testload" + strconv.Itoa(count)
	count++
	value := "111111111111111111111111111111"
	key = base64.StdEncoding.EncodeToString([]byte(key))
	value = base64.StdEncoding.EncodeToString([]byte(value))
	data := EtcdPut{key, value, ""}
	b, err := json.Marshal(data)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println("send data", string(b))
	resp, err := http.Post("http://10.14.3.229:23791/v3alpha/kv/put", "application/x-www-form-urlencoded", strings.NewReader(string(b)))
	if err != nil {
		fmt.Println("post error ", err)
		w.WriteHeader(400)
		io.WriteString(w, err.Error())
	}
	defer resp.Body.Close()
	//io.Reader

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("read response error ", err)
		w.WriteHeader(500)
		io.WriteString(w, err.Error())
	}
	w.Write(body)
	fmt.Println(body)
}
func main() {
	http.HandleFunc("/", HelloServer)
	http.HandleFunc("/etcdload", etcdLoad)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	fmt.Println("server end")
}
