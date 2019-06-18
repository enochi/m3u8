package m3u8

import "github.com/beego/bee/utils"

// dirStruct describes the application's directory structure
type dirStruct struct {
	WatchAll    bool `json:"watch_all" yaml:"watch_all"`
	Controllers string
	Models      string
	Others      []string // Other directories
}

var SQLDriver utils.DocValue

// var atagRegExp = regexp.MustCompile(`<a[^>]+[(href)|(HREF)]\s*\t*\n*=\s*\t*\n*[(".+")|('.+')][^>]*>[^<]*</a>`) //以Must前缀的方法或函数都是必须保证一定能执行成功的,否则将引发一次panic
// func getPlaylist(url string) string {
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	defer resp.Body.Close()
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		// handle error
// 	}
// 	return string(body)
// }
// func getTsFile(baseUrl string, filename string) (b []byte, err error) {
// 	url := baseUrl + "/" + filename
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	defer resp.Body.Close()
// 	body, err := ioutil.ReadAll(resp.Body)
// 	return nil, ioutil.WriteFile(filename, body, 0666)
// }

// var url = "https://streamcraft-sa.akamaized.net/stream/live/WeLive-Ext-OL_14555120_515_2183954_sd/vod/play_1560302291776.m3u8"
