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
var wg sync.WaitGroup
var jobs chan string
var faileds chan string
var base string

func getPlaylist(url string) string {
	body, err := ioutil.ReadFile(url)
	if err == nil {
		return string(body)
	}
	m3u8name := url
	l := strings.LastIndex(m3u8name, "/")
	if l != -1 {
		m3u8name = m3u8name[l+1:]
	}
	if strings.HasPrefix(url, "http") {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile(m3u8name, body, 0666)
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
		return string(body)
	}
	log.Fatal("no m3u8 content")
	return ""
}
func downloadFile(file string, force bool) (int64, error) {
	if !force {
		info, err := os.Stat(file)
		if err == nil && info.Size() > 1024 {
			fmt.Printf("%s already download size:%d\n", file, info.Size())
			return info.Size(), nil
		}
	}
	url := base + "/" + file
	var body []byte
	fmt.Println("start download ", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if body[0] == '<' {
		return 0, fmt.Errorf("resp error")
	}
	err = ioutil.WriteFile(file, body, 0666)
	if err != nil {
		os.Remove(file)
		fmt.Println(err)
		return 0, err
	}
	fmt.Printf("download success %s size:%d\n", url, len(body))
	return int64(len(body)), nil

}
func getTsFile() error {
	wg.Add(1)
	defer wg.Done()
	var err error
	for file := range jobs {
		_, err := downloadFile(file, false)
		if err != nil {
			fmt.Println("download failed", file)
			faileds <- file
		}
	}
	fmt.Println("download channel quit ", err)
	return err
}
func dealFailed() {
	for failedFile := range faileds {
		fmt.Println("add failed download file ", failedFile)
		jobs <- failedFile
	}
}
func startDownloadWorker() {
	jobs = make(chan string, 10)
	faileds = make(chan string, 1000)
	for w := 1; w <= 5; w++ {
		go getTsFile()
	}
	go dealFailed()
	wg.Wait()
}
func downloadAllTs(m3u8 string, checkSeq bool) {
	startDownloadWorker()
	allTss = make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(m3u8))
	var lastSeq = -1
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "#EXTINF") {
			reg := regexp.MustCompile(`#EXTINF:([\d]+)`)
			strDuration := reg.FindStringSubmatch(text)
			if strDuration == nil {
				fmt.Println("regex can't find ts duration")
			}
			duration, err := strconv.ParseFloat(strDuration[1], 32)
			if err != nil {
				fmt.Println("duration error")
			}
			if scanner.Scan() {
				filename := scanner.Text()
				reg := regexp.MustCompile(`([\d]+)\.ts`)
				thisSeq := reg.FindStringSubmatch(filename)
				if thisSeq == nil {
					fmt.Println("regex can't find ts file")
				}
				thisSeqNum, err := strconv.Atoi(thisSeq[1])
				if err != nil {
					fmt.Println(err)
					break
				}
				if thisSeqNum != lastSeq+1 {
					fmt.Printf("seq error %d-%d", lastSeq, thisSeqNum)
					if checkSeq {
						break
					}
				}
				lastSeq = thisSeqNum
				if v, ok := allTss[filename]; ok {
					fmt.Println("find duplicate ts", v)
					if checkSeq {
						break
					}
				}
				allTss[filename] = true
				fmt.Println(duration, filename)
				if !checkSeq {
					jobs <- filename
				}

			}

		}
	}
	fmt.Println("allts size:", len(allTss))
	close(jobs)
	if scanner.Err() != nil {
		fmt.Printf(" > Failed!: %v\n", scanner.Err())
	}

}

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

/*,
        {
            "codec_type": "audio",
            "stream_index": 0,
            "pts": 877690890,
            "pts_time": "9752.121000",
            "dts": 877690890,
            "dts_time": "9752.121000",
            "duration": 2089,
            "duration_time": "0.023211",
            "size": "457",
            "pos": "3572",
            "flags": "K_",
            "side_data_list": [
                {
                    "side_data_type": "MPEGTS Stream ID"
                }
            ]
		}
*/
func getLastPkt(filename string) string {

	info, err := os.Stat(filename)
	if err != nil || info.Size() < 1024 {
		_, err := downloadFile(filename, true)
		if err != nil {
			return ""
		}
	}
	cmdProbeTs := "ffprobe -show_packets -of json "
	cmdProbeTs += "." + "/" + filename
	list := strings.Split(cmdProbeTs, " ")
	cmd := exec.Command(list[0], list[1:]...)
	// stderrOutPut,err := cmd.StderrPipe()
	// if(err){
	// 	log.Fatal(err)
	// }
	pwdOutput, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(base, "http") {
			fmt.Println("ffprobe error restart download", filename)
			_, err := downloadFile(filename, true)
			if err != nil {
				log.Fatal(err)
				return ""
			}
			return getLastPkt(filename)
		}
		return ""
	}
	//fmt.Println(string(pwdOutput))
	lastPkt := string(pwdOutput[bytes.LastIndex(pwdOutput, []byte("codec_type")):])
	return lastPkt
}
func calcM3u8(m3u8 string) {
	totalDuration := 0.0
	var lastPtsTime float64
	var lastSeq = -1
	var err error
	scanner := bufio.NewScanner(strings.NewReader(m3u8))
	for scanner.Scan() {
		//fmt.Println(scanner.Text())
		text := scanner.Text()
		if strings.HasPrefix(text, "#EXTINF") {
			//#EXTINF:2.000,
			//1566196545993-129.ts
			reg := regexp.MustCompile(`#EXTINF:([\d]+)`)
			strDuration := reg.FindStringSubmatch(text)
			if strDuration == nil {
				fmt.Println("regex can't find ts duration")
				break
			}
			duration, err := strconv.ParseFloat(strDuration[1], 32)
			if err != nil {
				fmt.Println("duration error")
				break
			}
			totalDuration += duration
			//parse file to get file duration
			if scanner.Scan() {
				filename := scanner.Text()
				reg := regexp.MustCompile(`([\d]+)\.ts`)
				thisSeq := reg.FindStringSubmatch(filename)
				if thisSeq == nil {
					fmt.Println("regex can't find ts file")
				}
				thisSeqNum, err := strconv.Atoi(thisSeq[1])
				if err != nil {
					fmt.Println(err)
					break
				}
				if thisSeqNum != lastSeq+1 {
					fmt.Printf("seq error %d-%d", lastSeq, thisSeqNum)
					break
				}
				lastSeq = thisSeqNum
				lastPkt := getLastPkt(filename)
				v, _ := parsePkt(lastPkt)
				pts, err := strconv.Atoi(v["pts"])
				if err != nil {
					fmt.Println("no pts ")
					break
				}

				ptsTime, err := strconv.ParseFloat(v["pts_time"], 32)
				if err != nil {
					fmt.Println("no pts_time ")
					break
				}
				fileDuration := ptsTime - lastPtsTime
				curDiff := fileDuration - duration
				totalDiff := ptsTime - totalDuration
				fmt.Printf("list  file:%s duration:%f,total:%f\n", filename, duration, totalDuration)
				fmt.Printf("probe  duration:%f,pts:%d,ptsTime:%f \n", fileDuration, pts, ptsTime)
				fmt.Printf("diff cur:%f,total:%f\n", curDiff, totalDiff)
				if math.Abs(curDiff) > 1 || math.Abs(totalDiff) > 1 {
					fmt.Printf("curDiff %f totaldiff:%f", curDiff, totalDiff)
					break
				}
				lastPtsTime = ptsTime
			}

		}
	}
	if err != nil {
		log.Fatal(err)
	}
	if scanner.Err() != nil {
		fmt.Printf(" > Failed!: %v\n", scanner.Err())
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("please input as %s -d m3u8 url 1/0(check seq) ", os.Args[0])
	}
	// base = `https://streamcraft-sim.akamaized.net/qa/live/WeLive-QA_21000210_1027_311975/vod/`
	// lastPkt := getLastPkt("1566219192265-4902.ts")
	// v, _ := parsePkt(lastPkt)
	// pts, err := strconv.Atoi(v["pts"])
	// if err != nil {
	// 	fmt.Println("no pts ")
	// }

	// ptsTime, err := strconv.ParseFloat(v["pts_time"], 32)
	// if err != nil {
	// 	fmt.Println("no pts_time ")
	// }
	// fmt.Println(v, pts, ptsTime)
	url := os.Args[2]
	if len(os.Args) > 3 {
		base = os.Args[3]
	} else {
		if strings.Contains(url, "http") {
			base = url[:strings.LastIndex(url, "/")]
		} else {
			l := strings.LastIndex(url, "/")
			base = "."
			if l != -1 {
				base = url[:l]
			}
		}
	}
	m3u8 := getPlaylist(url)
	if strings.Compare(os.Args[1], "-c") == 0 {
		calcM3u8(m3u8)
	} else if strings.Compare(os.Args[1], "-d") == 0 {
		startDownloadWorker()
		checkSeq := false
		if len(os.Args) > 3 && os.Args[3] == "1" {
			checkSeq = true
		}
		downloadAllTs(m3u8, checkSeq)
	} else {
		fmt.Printf("input as %s -c m3u8file -d m3u8 url\n", os.Args[0])
	}
}
