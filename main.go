package main

import (
	"bilisuit/utils"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BuildMessage
// 生成报文
func BuildMessage(headers map[string]string, formData string) ([]byte, []byte) {
	var message = "POST /xlive/revenue/v2/order/createOrder HTTP/1.1\r\n"
	for s := range headers {
		message += fmt.Sprintf("%v: %v\r\n", s, headers[s])
	}
	var MessageByte = []byte(message + "\r\n" + formData)
	return MessageByte[:len(MessageByte)-1], MessageByte[len(MessageByte)-1:]
}

// H1CreateTlsConnection
// 创建连接
func H1CreateTlsConnection(BuyHost string) *tls.Conn {
	var adder = fmt.Sprintf("%v:443", BuyHost)
	var client, err = tls.Dial("tcp", adder, &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         BuyHost,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS12,
		ClientAuth:         tls.RequireAndVerifyClientCert,
	})
	if err != nil {
		fmt.Printf("连接购买服务器失败: %v\n", err)
		return nil
	}
	return client
}

// H1SendMessage
// 发送请求
func H1SendMessage(client *tls.Conn, body []byte) bool {
	if client == nil {
		return false
	}
	_, err := client.Write(body)
	if err != nil {
		fmt.Printf("发送请求失败: %v\n", err)
		return false
	}
	return true
}

// H1ReceiveResponse
// 接收响应
func H1ReceiveResponse(client *tls.Conn, BufLen int64) []byte {
	if client == nil {
		return nil
	}
	var result = make([]byte, BufLen)
	var length, err = client.Read(result)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return nil
	}
	return result[:length]
}

func IsPurchaseSuccess(res []byte) bool {
	if len(res) == 0 {
		return false
	}

	var body = string(res)
	var splitBody = strings.Split(body, "\r\n\r\n")
	body = splitBody[len(splitBody)-1]

	var jsonData = make(map[string]interface{})
	if err := json.Unmarshal([]byte(body), &jsonData); err != nil {
		fmt.Printf("解析购买响应失败: %v\n", err)
		return false
	}

	code, ok := jsonData["code"].(float64)
	if !ok {
		fmt.Printf("购买响应中没有code字段，按失败处理\n")
		return false
	}

	if code != 0 {
		var msg = ""
		if message, ok := jsonData["message"].(string); ok {
			msg = message
		} else if message, ok := jsonData["msg"].(string); ok {
			msg = message
		}
		fmt.Printf("购买失败响应: code=%v message=%v\n", code, msg)
	}

	return code == 0
}

func FinishPurchase(client *tls.Conn, messageBody []byte) ([]byte, int64, bool) {
	var s = time.Now().UnixNano() / 1e6

	if !H1SendMessage(client, messageBody) {
		return nil, 0, false
	}
	var res = H1ReceiveResponse(client, 1024)

	var e = time.Now().UnixNano() / 1e6
	return res, e - s, IsPurchaseSuccess(res)
}

func BuyOnce(headers map[string]string, messageHeader, messageBody []byte, sleepTime time.Duration) ([]byte, int64, bool) {
	var client = H1CreateTlsConnection(headers["host"])
	if client == nil {
		return nil, 0, false
	}
	defer client.Close()

	if !H1SendMessage(client, messageHeader) {
		return nil, 0, false
	}

	time.Sleep(sleepTime)
	return FinishPurchase(client, messageBody)
}

func main() {
	var immediateBuy = utils.IsImmediateBuy()
	var filePath = utils.GetSettingFilePath()
	var headers, startTime, delayTime, formData = utils.ReaderSetting(filePath, immediateBuy)
	var SleepTimeNumber = (float64(delayTime) / 1000) * float64(time.Second)

	var MessageHeader, MessageBody = BuildMessage(headers, formData)

	if !immediateBuy {
		utils.WaitLocalBiliTimer(startTime, 3)
	}

	var client = H1CreateTlsConnection(headers["host"])
	if client == nil {
		return
	}
	defer client.Close()

	if !H1SendMessage(client, MessageHeader) {
		return
	}

	if !immediateBuy {
		utils.WaitServerBiliTimer(startTime, 1)
	}

	time.Sleep(time.Duration(SleepTimeNumber))

	var res, cost, success = FinishPurchase(client, MessageBody)
	fmt.Printf("\n%v\n", string(res))
	fmt.Printf("耗时%vms\n", cost)
	if success {
		fmt.Printf("购买成功\n")
		return
	}

	const maxRetry = 15
	const retryInterval = 3 * time.Second

	for i := 1; i <= maxRetry; i++ {
		fmt.Printf("\n购买失败，%v秒后进行第%v/%v次重试\n", int(retryInterval/time.Second), i, maxRetry)
		time.Sleep(retryInterval)

		res, cost, success = BuyOnce(headers, MessageHeader, MessageBody, time.Duration(SleepTimeNumber))
		fmt.Printf("\n%v\n", string(res))
		fmt.Printf("耗时%vms\n", cost)

		if success {
			fmt.Printf("购买成功\n")
			return
		}
	}

	fmt.Printf("购买失败，已重试%v次\n", maxRetry)
}
