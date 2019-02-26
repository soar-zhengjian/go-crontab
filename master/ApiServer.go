package master

import (
	"encoding/json"
	"go-crontab/common"
	"go-crontab/logger"
	"net"
	"net/http"
	"strconv"
	"time"
)

// 任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

var (
	// 单例对象
	G_apiServer *ApiServer
)

// 初始化服务
func InitApiServer() (err error) {
	var (
		mux           *http.ServeMux
		listener      net.Listener
		httpServer    *http.Server
		staticDir     http.Dir     // 静态文件根目录
		staticHandler http.Handler // 静态文件的HTTP回调
	)

	// 配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)
	mux.HandleFunc("/job/log", handleJobLog)
	mux.HandleFunc("/worker/list", handleWorkerList)

	//  /index.html

	// 静态文件目录
	staticDir = http.Dir(G_config.WebRoot)
	staticHandler = http.FileServer(staticDir)
	mux.Handle("/", http.StripPrefix("/", staticHandler)) // ./webroot/index.html

	// 启动TCP监听
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_config.ApiPort)); err != nil {
		return
	}

	// 创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	// 赋值单例
	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}

	// 启动服务端
	go httpServer.Serve(listener)
	return
}

// 保存任务接口
// POST job={"name":"job1","command":"echo hello", "cronExpr":"* * * * *"}
func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	var (
		err     error
		postJob string
		job     common.Job
		oldJob  *common.Job
	)

	// 1,解析POST表单
	if err = req.ParseForm(); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}

	// 2,取表单中的job字段
	postJob = req.PostForm.Get("job")

	// 3,反序列化job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}

	// 4,保存到etcd
	if oldJob, err = G_jobMgr.SaveJob(&job); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}

	logger.Infof("[/job/save]成功:%s\t%+v\n", time.Now().Format("2006-01-02 15:04:05"), job)
	// 5,正常应答
	common.ResponseErr(resp, 0, "success", oldJob)
	return

}

// 删除任务接口
// POST /job/delete name=job1
func handleJobDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err    error
		name   string
		oldJob *common.Job
	)

	// 1,解析POST表单 POST: a=1&b=2&c=3
	if err = req.ParseForm(); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}

	// 2,删除的任务名
	name = req.PostForm.Get("name")

	// 3,去删除任务
	if oldJob, err = G_jobMgr.DeleteJob(name); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}

	logger.Infof("[/job/delete]成功:%s\t%+v\n", time.Now().Format("2006-01-02 15:04:05"), name)
	// 4,正常应答
	common.ResponseErr(resp, 0, "success", oldJob)
	return
}

// 列举所有crontab任务
func handleJobList(resp http.ResponseWriter, req *http.Request) {
	var (
		jobList []*common.Job
		err     error
	)

	// 获取任务列表
	if jobList, err = G_jobMgr.ListJobs(); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}

	logger.Infof("[/job/list]成功:%s\t%d\n", time.Now().Format("2006-01-02 15:04:05"), len(jobList))
	// 正常应答
	common.ResponseErr(resp, 0, "success", jobList)
	return
}

// 强制杀死某个任务
// POST /job/kill name=job1
func handleJobKill(resp http.ResponseWriter, req *http.Request) {
	var (
		err  error
		name string
	)

	// 1,解析POST表单
	if err = req.ParseForm(); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}

	// 2,要杀死的任务名
	name = req.PostForm.Get("name")

	// 3,杀死任务
	if err = G_jobMgr.KillJob(name); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}
	logger.Infof("[/job/kill]成功:%s\t%d\n", time.Now().Format("2006-01-02 15:04:05"), name)
	// 正常应答
	common.ResponseErr(resp, 0, "success", nil)
	return
}

// 查询任务日志
func handleJobLog(resp http.ResponseWriter, req *http.Request) {
	var (
		err        error
		name       string // 任务名字
		skipParam  string // 从第几条开始
		limitParam string // 返回多少条
		skip       int
		limit      int
		logArr     []*common.JobLog
	)

	// 1,解析Get参数
	if err = req.ParseForm(); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}

	// 2,获取请求参数 /job/log?name=job10&skip=0&limit=10
	name = req.Form.Get("name")
	skipParam = req.Form.Get("skip")
	limitParam = req.Form.Get("limit")
	if skip, err = strconv.Atoi(skipParam); err != nil {
		skip = 0
	}
	if limit, err = strconv.Atoi(limitParam); err != nil {
		limit = 20
	}

	if logArr, err = G_logMgr.ListLog(name, skip, limit); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}

	logger.Infof("[/job/log]成功:%s\t%s\n", time.Now().Format("2006-01-02 15:04:05"), name)
	// 正常应答
	common.ResponseErr(resp, 0, "success", logArr)
	return
}

// 获取健康worker节点列表
func handleWorkerList(resp http.ResponseWriter, req *http.Request) {
	var (
		workerArr []string
		err       error
	)

	if workerArr, err = G_workerMgr.ListWorkers(); err != nil {
		common.ResponseErr(resp, -1, err.Error(), nil)
		return
	}

	logger.Infof("[/worker/list]成功:%s\t%d\n", time.Now().Format("2006-01-02 15:04:05"), len(workerArr))
	// 正常应答
	common.ResponseErr(resp, 0, "success", workerArr)
	return
}
