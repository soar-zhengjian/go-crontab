package main

import (
	"flag"
	"fmt"
	"go-crontab/common"
	"go-crontab/logger"
	"go-crontab/master"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

var (
	confFile string // 配置文件路径
	err      error
)

// 解析命令行参数
func initArgs() {
	// master -config ./master.json -xxx 123 -yyy ddd
	flag.StringVar(&confFile, "config", "./master.json", "指定master.json")
	flag.Parse()
}

// 初始化线程数量
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	// 初始化命令行参数
	initArgs()

	// 初始化线程
	initEnv()

	// 日志配置
	path := "logs"
	mode := os.ModePerm
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, mode)
	}
	file, _ := os.Create(strings.Join([]string{path, "log.txt"}, "/"))
	defer file.Close()
	loger := log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)
	logger.SetDefault(loger)

	// 加载config配置
	if err = master.InitConfig(confFile); err != nil {
		common.FmtErr(err)
		return
	}
	// 输出config 结果
	logger.Infof("%+v\n", master.G_config)

	// 初始化服务发现模块,提供方法给 /worker/list 接口
	if err = master.InitWorkerMgr(); err != nil {
		common.FmtErr(err)
		return
	}

	// 初始化日志管理器, 提供方法给 /job/log接口
	if err = master.InitLogMgr(); err != nil {
		common.FmtErr(err)
		return
	}

	// 初始化任务管理器,提供方法给  /job/save+delete+list+kill接口
	if err = master.InitJobMgr(); err != nil {
		common.FmtErr(err)
		return
	}

	// 启动Api Http服务
	if err = master.InitApiServer(); err != nil {
		common.FmtErr(err)
		return
	}
	// go func() {
	// 	ticker := time.NewTicker(10 * time.Second)
	// 	for t := range ticker.C {
	// 		fmt.Println(t.Format("2006-01-02 15:04:05"), runtime.NumGoroutine())
	// 	}
	// }()

	fmt.Println("master服务开启成功:\t", master.G_config.ApiPort)
	// 正常退出
	for {
		time.Sleep(1 * time.Second)
	}
	return
}
