package main

import (
	"flag"
	"fmt"
	"go-crontab/common"
	"go-crontab/logger"
	"go-crontab/worker"
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
	// worker -config ./worker.json
	// worker -h
	flag.StringVar(&confFile, "config", "./worker.json", "worker.json")
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

	// 加载配置
	if err = worker.InitConfig(confFile); err != nil {
		common.FmtErr(err)
		return
	}
	// 输出config 结果
	logger.Infof("%+v\n", worker.G_config)

	// 服务注册，注册本机IP到etcd，并不断续租
	if err = worker.InitRegister(); err != nil {
		common.FmtErr(err)
		return
	}

	// 启动日志协程, 被动接收信息，处理，保存
	if err = worker.InitLogSink(); err != nil {
		common.FmtErr(err)
		return
	}

	// 启动执行器
	if err = worker.InitExecutor(); err != nil {
		common.FmtErr(err)
		return
	}

	// 启动调度器
	if err = worker.InitScheduler(); err != nil {
		common.FmtErr(err)
		return
	}

	// 初始化任务管理器
	if err = worker.InitJobMgr(); err != nil {
		common.FmtErr(err)
		return
	}

	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "worker服务开启成功\t")

	// go func() {
	// 	ticker := time.NewTicker(10 * time.Second)
	// 	for t := range ticker.C {
	// 		fmt.Println(t.Format("2006-01-02 15:04:05"), runtime.NumGoroutine())
	// 	}
	// }()

	// 正常退出
	for {
		time.Sleep(1 * time.Second)
	}
	return
}
