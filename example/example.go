package main

import (
	"fmt"

	"github.com/xkeyideal/log4go"
)

func asyncLogUserd() {
	logger := log4go.NewLogger(100, "async_test_service") //设置异步channel的大小，log属于的服务名
	err := logger.Open("./logs", "async_test", "Debug")   //分别表示设置log文件的存储路径，文件名，log的级别
	if err != nil {
		fmt.Println(err)
	}

	logger.SetLevel("Debug")   //设置log级别
	logger.SetMaxDays(7)       //设置log文件最大保存的天数
	logger.SetMaxLines(10000)  //设置单个log文件最大的行数，超过该行数会自动存至另一个文件
	logger.SetMaxSize(1 << 28) //256M 设置单个log文件最大的空间，超过该空间会自动存至另一个文件

	logger.Debug("Debug: %s %s %s", "1", "2", "3")
	logger.Info("Info: %d %d %s", 1, 4, "5")
	logger.Notice("Notice")
	logger.Warn("Warn")
	logger.Fatal("Fatal")

	logger.Close()
}

func main() {
	//asyncLogUserd()
}
