package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var allTss map[string]bool
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
		err = ioutil.WriteFile("save.m3u8", body, 0666)
		if err != nil {
			fmt.Println(err)
			return ""
		}
		return string(body)
	} else {
		body, err := ioutil.ReadFile(url)
		if err != nil {
			fmt.Println(err)
			return ""
		}
		err = ioutil.WriteFile("save.m3u8", body, 0666)
		if err != nil {
			fmt.Println(err)
			return ""
		}
		return string(body)
	}

}

var wg sync.WaitGroup

func getTsFile(files chan string) error {
	wg.Add(1)
	defer wg.Done()
	var err error
	for file := range files {

		info, err := os.Stat(file)
		if err == nil && info.Size() > 0 {
			fmt.Printf("%s already download size:%d\n", file, info.Size())
			continue
		}
		url := base + "/" + file
		var body []byte
		fmt.Println("start download ", url)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
			goto failed
		}
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			goto failed
		}
		err = ioutil.WriteFile(file, body, 0666)
		if err != nil {
			fmt.Println(err)
			goto failed
		}
		fmt.Println("download success ", url)
		continue
	failed:
		fmt.Println("download failed", url)
		files <- file

	}
	fmt.Println("download channel quit ", err)
	return err
}
func downloadAllTs(m3u8 string) {
	jobs := make(chan string, 10)
	for w := 1; w <= 5; w++ {
		go getTsFile(jobs)
	}
	allTss = make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(m3u8))
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "#EXTINF") {
			text = text[0 : len(text)-1]
			tmp := strings.Split(text, ":")
			duration, _ := strconv.ParseFloat(tmp[1], 32)
			if scanner.Scan() {
				filename := scanner.Text()
				if v, ok := allTss[filename]; ok {
					fmt.Println("find duplicate ts", v)
					break
				}
				allTss[filename] = true
				fmt.Println(duration, filename)
				jobs <- filename
			}

		}
	}
	fmt.Println("allts size:", len(allTss))
	close(jobs)
	if scanner.Err() != nil {
		fmt.Printf(" > Failed!: %v\n", scanner.Err())
	}
	wg.Wait()
}

type Pkt struct {
	media_type   string
	pkt_pts      string
	pkt_pts_time string
}

var pkts []Pkt

func parsePkt(pkt string) (v map[string]string, err error) {
	//fmt.Println("pkt ", pkt)
	v = make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(pkt))
	for scanner.Scan() {
		// "pts_time": "53.999000",
		text := scanner.Text()
		tmp := strings.Split(text, ":")
		if len(tmp) > 1 {
			key := strings.Trim(tmp[0], " ")
			key = strings.Trim(key, "\"")
			v1 := strings.Trim(tmp[1][0:len(tmp[1])-1], " ")
			v1 = strings.Trim(v1, "\"")
			v[key] = v1
		}
	}
	return
}

// func parsePkt1(pkt string) (pts, ptsTime int) {

// }
func getLastPkt(base string, filename string) string {
	cmdProbeTs := "ffprobe -show_packets -of json "
	cmdProbeTs += base + "/" + filename
	list := strings.Split(cmdProbeTs, " ")
	cmd := exec.Command(list[0], list[1:]...)
	// stderrOutPut,err := cmd.StderrPipe()
	// if(err){
	// 	log.Fatal(err)
	// }
	pwdOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(string(pwdOutput))
	lastPkt := string(pwdOutput[bytes.LastIndex(pwdOutput, []byte("{")):])
	return lastPkt
}
func calcM3u8(file string) {
	l := strings.LastIndex(file, "/")
	base := "."
	if l != -1 {
		base = file[:l]
	}
	h, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
	}
	totalDuration := 0.0
	var lastPtsTime float64
	scanner := bufio.NewScanner(h)
	for scanner.Scan() {
		//fmt.Println(scanner.Text())
		text := scanner.Text()
		if strings.HasPrefix(text, "#EXTINF") {
			text = text[0 : len(text)-1]
			tmp := strings.Split(text, ":")
			strDuration := tmp[1]
			l = strings.LastIndex(strDuration, ",")

			if l != -1 {
				strDuration = tmp[1][0:l]
			}
			duration, _ := strconv.ParseFloat(strDuration, 32)
			totalDuration += duration
			//parse file to get file duration
			if scanner.Scan() {
				filename := scanner.Text()
				lastPkt := getLastPkt(base, filename)
				v, _ := parsePkt(lastPkt)
				pts, _ := strconv.Atoi(v["pts"])
				ptsTime, _ := strconv.ParseFloat(v["pts_time"], 32)
				fileDuration := ptsTime - lastPtsTime
				curDiff := fileDuration - duration
				totalDiff := ptsTime - totalDuration
				fmt.Println("list  ", filename, duration, totalDuration)
				fmt.Println("file  ", pts, fileDuration, ptsTime)
				fmt.Println("diff ", curDiff, totalDiff)
				if math.Abs(curDiff) > 1 || math.Abs(totalDiff) > 1 {
					fmt.Errorf("curDiff %f totaldiff:%f", curDiff, totalDiff)
				}
				lastPtsTime = ptsTime
			}

		}
		//fmt.Println(totalDuration)

	}
	if scanner.Err() != nil {
		fmt.Printf(" > Failed!: %v\n", scanner.Err())
	}
}

var base string

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
	if len(os.Args) < 3 {
		log.Fatal("please input m3u8 file")
	}

	if strings.Compare(os.Args[1], "-c") == 0 {
		calcM3u8(os.Args[2])
	} else if strings.Compare(os.Args[1], "-d") == 0 {

		url := os.Args[2]
		ret := getPlaylist(url)
		base = url[:strings.LastIndex(url, "/")]
		downloadAllTs(ret)
	} else {
		fmt.Printf("input as %s -c m3u8file -d m3u8 url\n", os.Args[0])
	}
}
