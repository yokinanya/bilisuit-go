package utils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

var NowTime int64
var startTimer = true

var mutex = sync.Mutex{}

type NetTimer struct {
	Message []byte
	client  *tls.Conn
}

type BiliTimeResponse struct {
	Data json.RawMessage `json:"data"`
}

// init
// 初始化
func (receiver *NetTimer) init() *NetTimer {
	var MessageList = []string{
		"GET /x/report/click/now HTTP/1.1\r\nhost: api.bilibili.com", "Connection: keep-alive",
		"User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:105.0) Gecko/20100101 Firefox/105.0",
	}
	receiver.Message = []byte(strings.Join(MessageList, "\r\n") + "\r\n\r\n")
	return receiver
}

// updateClient
// 更新连接
func (receiver *NetTimer) updateClient() error {
	client, err := tls.Dial("tcp", "api.bilibili.com:443", &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "api.bilibili.com",
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS12,
		ClientAuth:         tls.RequireAndVerifyClientCert,
	})
	if err != nil {
		return err
	}
	receiver.client = client
	return nil
}

func parseBiliTime(body string) (int64, error) {
	var response BiliTimeResponse
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		return 0, err
	}

	var now int64
	if err := json.Unmarshal(response.Data, &now); err == nil {
		return now, nil
	}

	var data = make(map[string]int64)
	if err := json.Unmarshal(response.Data, &data); err != nil {
		return 0, err
	}
	return data["now"], nil
}

// GetBiliTime
// 获取b站时间
func (receiver *NetTimer) getBiliTime() int64 {
	if receiver.client == nil {
		if err := receiver.updateClient(); err != nil {
			fmt.Printf("连接B站时间服务器失败: %v\n", err)
			return 0
		}
	}

	_, err := receiver.client.Write(receiver.Message)
	if err != nil {
		fmt.Printf("请求B站时间失败: %v\n", err)
		_ = receiver.client.Close()
		receiver.client = nil
		return 0
	}

	var buf = make([]byte, 1024)
	var length, readErr = receiver.client.Read(buf)
	if readErr != nil || length == 0 {
		if readErr != nil {
			fmt.Printf("读取B站时间失败: %v\n", readErr)
		}
		_ = receiver.client.Close()
		receiver.client = nil
		return 0
	}

	var rec = string(buf[:length])
	var SplitBody = strings.Split(rec, "\r\n\r\n")
	var Body = SplitBody[len(SplitBody)-1]
	var biliTime, parseErr = parseBiliTime(Body)
	if parseErr != nil {
		fmt.Printf("解析B站时间失败: %v\n", parseErr)
		return 0
	}
	return biliTime
}

// UpdateServerTime
// 更新b站服务器时间
func (receiver *NetTimer) UpdateServerTime() {
	if err := receiver.updateClient(); err != nil {
		fmt.Printf("连接B站时间服务器失败: %v\n", err)
	}
	for startTimer {
		var biliTime = receiver.getBiliTime()
		if biliTime > 0 {
			mutex.Lock()
			if biliTime > NowTime {
				NowTime = biliTime
			}
			mutex.Unlock()
		}
		time.Sleep(50 * time.Millisecond)
	}
	if receiver.client != nil {
		_ = receiver.client.Close()
	}
}

// WaitLocalBiliTimer
// 计时器人口
// saleTime: 开售时间
// jump: 跳出时间
func WaitLocalBiliTimer(saleTime, jump int64) {
	var localTime, JumpTime float64
	localTime = float64(time.Now().UnixNano()) / 1e9
	JumpTime = float64(saleTime) - float64(jump)
	for JumpTime > localTime {
		fmt.Printf("\r%f", localTime)
		time.Sleep(20 * time.Millisecond)
		localTime = float64(time.Now().UnixNano()) / 1e9
	}
}

func WaitServerBiliTimer(saleTime, number int64) {
	for i := 0; i < int(number); i++ {
		var timer = new(NetTimer).init()
		go timer.UpdateServerTime()
	}
	for NowTime < saleTime {
		fmt.Printf("\r%v", NowTime)
		time.Sleep(10 * time.Millisecond)
	}
	startTimer = false
}
