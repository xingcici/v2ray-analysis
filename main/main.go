package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/lionsoul2014/ip2region/binding/golang/ip2region"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	file           = flag.String("f", "access.log", "输入文件")
	emaill         = flag.String("e", "", "email")
	user_addr      = flag.String("ua", "user_addr.csv", "输出文件")
	request_addr   = flag.String("ra", "request_addr.csv", "输出文件")
	telegram_token = flag.String("tk", "", "telegram token")
	chat_id        = flag.String("ci", "", "telegram chat id")
	ipp            = flag.Bool("ip", false, "获取使用者ip")
	urll           = flag.Bool("url", false, "获取访问路径")
	c              = flag.Int("c", 20, "多协程分析文件")
	ua             *os.File
	ra             *os.File
	lock           sync.Mutex
	region         *ip2region.Ip2Region
)

const (
	telegram_bot_url_prefix = "https://api.telegram.org/bot"
	telegram_bot_url_suffix = "/sendDocument"
)

type data struct {
	key   string
	value int
}

func main() {
	timeNow := time.Now()
	var timeFormat = timeNow.Format("2006/01/02")
	fmt.Println(timeFormat)
	flag.Parse()
	f, err := os.Open(*file)
	if err != nil {
		panic(err)
	}
	ua, err = os.OpenFile(*user_addr, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	ra, err = os.OpenFile(*request_addr, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	ua.WriteString("\xEF\xBB\xBF") //添加utf-8 BOM头，识别为uft-8编码，防止中文乱码
	ra.WriteString("\xEF\xBB\xBF") //添加utf-8 BOM头，识别为uft-8编码，防止中文乱码
	if err != nil {
		panic(err)
	}
	buf := bufio.NewReader(f)
	ips, urls := []data{}, []data{}
	w := sync.WaitGroup{}

	if !*ipp && !*urll {
		*c = 1
	}
	ch := make(chan struct{}, *c) // 多协程处理
	for {

		line, isPrefix, err := buf.ReadLine()
		if err != nil && err.Error() == "EOF" || isPrefix {
			break
		}
		ch <- struct{}{}
		w.Add(1)
		go func(line string) {
			defer func() {
				<-ch
				w.Done()
			}()
			arr := strings.Split(line, " ")
			if len(arr) == 6 && arr[0] == timeFormat {
				//time := bytes.Join(arr[:2], []byte(" "))
				// todo 完成一个ip对应多个路径
				ip := strings.Split(arr[2], ":")[0]
				ips = toMap(ips, ip)

				uri := strings.Split(arr[4], ":")
				//typee := uri[0]
				url := uri[1]
				//port := uri[2]
				urls = toMap(urls, url)

				//if !*ipp && !*urll {
				//	o.WriteString(line + "\n")
				//	o.WriteString("\n")
				//}

			}

		}(string(line))
	}
	w.Wait()
	// ip

	ua.WriteString("ip,频率,位置\n")
	var ipData strings.Builder
	sort.Slice(ips, func(i, j int) bool {
		return ips[i].value > ips[j].value
	})
	for _, v := range ips {
		ip := strings.Split(v.key, ".")
		if len(ip) == 4 {
			f := true
			for _, p := range ip {
				num, err := strconv.Atoi(p)
				if err != nil || num < 0 || num > 255 {
					f = false
					break
				}
			}
			if !f {
				continue
			}
			ipData.WriteString(v.key)
			ipData.WriteByte(',')
			ipData.WriteString(strconv.Itoa(v.value))
			ipData.WriteByte(',')
			ipData.WriteString(ip2addr(v.key))
			ipData.WriteByte('\n')
		}
	}
	ua.WriteString(ipData.String())
	ua.Close()
	// url

	ra.WriteString("url,频率\n")
	var data strings.Builder
	sort.Slice(urls, func(i, j int) bool {
		return urls[i].value > urls[j].value
	})
	for _, v := range urls {
		data.WriteString(v.key)
		data.WriteByte(',')
		data.WriteString(strconv.Itoa(v.value))
		data.WriteByte('\n')
	}
	ra.WriteString(data.String())

	ra.Close()
}

func toMap(m []data, key string) []data {
	lock.Lock()
	defer lock.Unlock()
	for k, v := range m {
		if v.key == key {
			m[k].value++
			return m
		}
	}
	return append(m, data{
		key:   key,
		value: 1,
	})
}

func ip2addr(ip string) string {
	if region == nil {
		_, err := os.Open("ip2region.db")
		if err != nil {
			if err = download("https://github.com/lionsoul2014/ip2region/raw/master/data/ip2region.db"); err != nil {
				panic(err)
			}
		}
		region, err = ip2region.New("ip2region.db")
		if err != nil {
			panic(err)
		}
	}
	i, err := region.BinarySearch(ip)
	if err != nil {
		log.Println(err)
		return ""
	}
	return i.String()
}

func download(url string) error {
	fmt.Println("downloading...")
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	n := strings.Split(url, "/")
	c, err := os.Create(n[len(n)-1])
	if err != nil {
		return err
	}
	_, err = io.Copy(c, res.Body)
	if err != nil {
		return err
	}
	fmt.Println("download done")
	return nil
}

//func telegramBot(fileAddr string) {
//	buf := new(bytes.Buffer)
//	w := multipart.NewWriter(buf)
//	fw, err := w.CreateFormFile("file", "1d595495-0580-49ec-b96c-cc3346096718")
//	req, err := http.NewRequest("POST", "http://localhost:8080/info", buf)
//	if err != nil {
//		fmt.Println("req err: ", err)
//		return
//	}
//	req.Header.Set("Content-Type", w.FormDataContentType())
//
//	http.DefaultClient.Do(req)
//	//if err != nil {
//	//	fmt.Println("resp err: ", err)
//	//	return
//	//}
//	//defer resp.Body.Close()
//	//
//	//if resp.StatusCode != 200 {
//	//	return errors.New("resp status:" + fmt.Sprint(resp.StatusCode))
//	//}
//	//
//	//return nil
//
//}
