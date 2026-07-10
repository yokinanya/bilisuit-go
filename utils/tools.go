package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func IsImmediateBuy() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--now" || arg == "--immediate" {
			return true
		}
	}
	return false
}

func GetSettingFilePath() string {
	var FilePath string
	if len(os.Args) <= 1 {
		fmt.Printf("请选择一个配置文件\n")
		os.Exit(1)
	}

	for _, arg := range os.Args[1:] {
		if arg == "--now" || arg == "--immediate" {
			continue
		}
		FilePath = arg
	}

	if FilePath == "" {
		fmt.Printf("请选择一个配置文件\n")
		os.Exit(1)
	}

	_, err := os.Lstat(FilePath)
	if err != nil {
		fmt.Printf("[%v]不存在\n", FilePath)
		os.Exit(1)
	}
	fmt.Printf("配置文件:[%v]\n", FilePath)
	return FilePath
}

type SettingContent struct {
	StartTime int64  `json:"start_time"`
	DelayTime int64  `json:"delay_time"`
	ItemId    string `json:"item_id"`
}

type SettingFile struct {
	Setting  SettingContent    `json:"setting"`
	FormData string            `json:"form_data"`
	Headers  map[string]string `json:"headers"`
}

func ReaderSetting(filePath string, immediateBuy bool) (map[string]string, int64, int64, string) {
	var SettingData, _ = os.ReadFile(filePath)
	var settingContent = SettingFile{}

	_ = json.Unmarshal(SettingData, &settingContent)

	var headers = settingContent.Headers
	var formData = settingContent.FormData
	var startTime = settingContent.Setting.StartTime

	if startTime <= time.Now().Unix() {
		if !immediateBuy {
			fmt.Printf("%v\n", "启动时间小于当前时间")
			fmt.Printf("如需忽略启动时间并立即购买，请添加参数: --now\n")
			os.Exit(2)
		}
		fmt.Printf("启动时间小于当前时间，已启用立即购买模式\n")
	}

	var delayTime = settingContent.Setting.DelayTime

	fmt.Printf("装扮id:[%v]\n", settingContent.Setting.ItemId)
	fmt.Printf("启动时间:[%v]\n", startTime)
	fmt.Printf("延时:[%vms]\n", delayTime)

	return headers, startTime, delayTime, formData
}
