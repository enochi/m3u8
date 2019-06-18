package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var atagRegExp = regexp.MustCompile(`<a[^>]+[(href)|(HREF)]\s*\t*\n*=\s*\t*\n*[(".+")|('.+')][^>]*>[^<]*</a>`) //以Must前缀的方法或函数都是必须保证一定能执行成功的,否则将引发一次panic
func getPlaylist(url string) string {
	if strings.HasPrefix(url, "http") {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
		}
		return string(body)
	} else {
		body, err := ioutil.ReadFile(url)
		if err != nil {
			fmt.Println(err)
		}
		return string(body)
	}

}

var wg sync.WaitGroup

func getTsFile(base string, filename string) error {
	wg.Add(1)
	url := base + "/" + filename
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	err = ioutil.WriteFile(filename, body, 0666)
	if err != nil {
		fmt.Println(err)
	}
	return err
}
func downloadAllTs(base string, m3u8 string) {

	scanner := bufio.NewScanner(strings.NewReader(m3u8))
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		text := scanner.Text()

		if strings.HasPrefix(text, "#EXTINF") {
			text = text[0 : len(text)-1]
			tmp := strings.Split(text, ":")
			duration, _ := strconv.ParseFloat(tmp[1], 32)
			fmt.Println("get duration:", duration)
			if scanner.Scan() {
				filename := scanner.Text()
				fmt.Println(duration, filename)
				go getTsFile(base, filename)
			}

		}

	}
	if scanner.Err() != nil {
		fmt.Printf(" > Failed!: %v\n", scanner.Err())
	}
	wg.Wait()
}
func calcM3u8(file string) {
	base := file[:strings.LastIndex(file, "\\")]
	h, err := os.Open(file)
	if err != nil {
		println(err)
	}
	totalDuration := 0.0
	scanner := bufio.NewScanner(h)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		text := scanner.Text()

		if strings.HasPrefix(text, "#EXTINF") {
			text = text[0 : len(text)-1]
			tmp := strings.Split(text, ":")
			duration, _ := strconv.ParseFloat(tmp[1], 32)
			totalDuration += duration
			fmt.Println("get duration:", duration)
			if scanner.Scan() {
				filename := scanner.Text()
				fmt.Println(duration, filename)
				cmdProbeTs := ` -show_frames -select_streams v -of json `
				cmdProbeTs += base + "\\" + filename
				pwdCmd := exec.Command(`D:\msys64\mingw64\bin\ffmpeg`, cmdProbeTs)
				pwdOutput, err := pwdCmd.Output()
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(string(pwdOutput))

			}

		}
		fmt.Println(totalDuration)

	}
	if scanner.Err() != nil {
		fmt.Printf(" > Failed!: %v\n", scanner.Err())
	}
}

var url = "https://streamcraft-sa.akamaized.net/stream/live/WeLive-Ext-OL_14555120_515_2183954_sd/vod/play_1560302291776.m3u8"

func main() {
	// content := []byte(`
	// # json fragment
	// "id": "dbsuye23sd83d8dasf7",
	// "name": "Larry",
	// "birth_year": 2000
	// `)
	// p := regexp.MustCompile(`(?m)"(?P\w+)":\s+"?(?P[a-zA-Z0-9]+)"?`)
	// var dst []byte
	// tpl := []byte("$key=$value\n")
	// for _, submatches := range p.FindAllSubmatchIndex(content, -1) {
	// 	dst = p.Expand(dst, tpl, content, submatches)
	// }
	// pat := `(?:#EXTINF:)(([+-]?([0-9]*[.])?[0-9]+))`
	// re, _ := regexp.Compile(pat)
	// allTs := re.FindAllString(ret, -1)
	// for _, ts := range allTs {
	// 	fmt.Println(ts)
	// }
	//fmt.Println(string(dst))
	//ret := getPlaylist(url)
	//base := url[:strings.LastIndex(url, "/")]
	//downloadAllTs(base, ret)
	calcM3u8(`E:\v\b\0.m3u8`)
}
