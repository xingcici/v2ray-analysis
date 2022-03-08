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
)

var (
	file   = flag.String("f", "/var/log/v2ray/access.log", "输入文件")
	emaill = flag.String("e", "", "email")
	out    = flag.String("o", "out.csv", "输出文件")
	ipp    = flag.Bool("ip", false, "获取使用者ip")
	urll   = flag.Bool("url", false, "获取访问路径")
	c      = flag.Int("c", 20, "多协程分析文件")
	o      *os.File
	lock   sync.Mutex
	region *ip2region.Ip2Region
)

type data struct {
	key   string
	value int
}

func main() {
	flag.Parse()
	f, err := os.Open(*file)
	if err != nil {
		panic(err)
	}
	o, err = os.OpenFile(*out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	o.WriteString("\xEF\xBB\xBF") //添加utf-8 BOM头，识别为uft-8编码，防止中文乱码
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
			if len(arr) == 6 {
				//time := bytes.Join(arr[:2], []byte(" "))
				// todo 完成一个ip对应多个路径
				var ippp = *ipp
				fmt.Println(ippp)

				ip := strings.Split(arr[2], ":")[0]
				ips = toMap(ips, ip)

				uri := strings.Split(arr[4], ":")
				//typee := uri[0]
				url := uri[1]
				//port := uri[2]
				urls = toMap(urls, url)

				if !*ipp && !*urll {
					o.WriteString(line + "\n")
					o.WriteString("\n")
				}

			}

		}(string(line))
	}
	w.Wait()
	// ip

	o.WriteString("ip,频率,位置\n")
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
	o.WriteString(ipData.String())

	// url

	o.WriteString("url,频率\n")
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
	o.WriteString(data.String())

	o.Close()
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
