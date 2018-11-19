package master

import (
	"net/http"
	"net"
	"time"
	"go-crontab/common"
	"encoding/json"
)

//任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

var (
	//单例对象
	G_apiServer *ApiServer
)
//保存任务接口
//POST job={"name":"job1","command":"echo hello", "cronExpr":"* * * * *"}
func handleJobSave(resp http.ResponseWriter, req *http.Request)  {
	var (
		err error
		postJob string
		job common.Job
		oldJob *common.Job
	)

	//1,解析POST表单
	if err = req.ParseForm(); err != nil{
		common.ResponseErr(resp,-1,err.Error(),nil)
		return
	}

	//2,取表单中的job字段
	postJob = req.PostForm.Get("job")

	//3,反序列化job
	if err = json.Unmarshal([]byte(postJob),&job); err != nil{
		common.ResponseErr(resp,-1,err.Error(),nil)
		return
	}

	//4,保存到etcd
	if oldJob, err = G_jobMgr.SaveJob(&job); err != nil{
		common.ResponseErr(resp,-1,err.Error(),nil)
		return
	}

	//正常应答
	common.ResponseErr(resp,0,"success",oldJob)
	return

}

//删除任务接口
//POST /job/delete name=job1
func handleJobDelete(resp http.ResponseWriter, req *http.Request)  {
	var (
		err error
		name string
		oldJob *common.Job
	)
	//POST: a=1&b=2&c=3
	if err = req.ParseForm(); err != nil{
		common.ResponseErr(resp,-1,err.Error(),nil)
		return
	}

	//删除的任务名
	name = req.PostForm.Get("name")

	//去删除任务
	if oldJob,err = G_jobMgr.DeletedJob(name); err != nil{
		common.ResponseErr(resp,-1,err.Error(),nil)
		return
	}

	//正常应答
	common.ResponseErr(resp,0,"success",oldJob)
	return
}

//列举所有crontab任务
func handleJobList(resp http.ResponseWriter, req *http.Request)  {
	var(
		jobList []*common.Job
		err error
	)

	//获取任务列表
	if jobList, err = G_jobMgr.ListJobs(); err != nil{
		common.ResponseErr(resp,-1,err.Error(),nil)
		return
	}

	//正常应答
	common.ResponseErr(resp,0,"success",jobList)
	return
}

//强制杀死某个任务

//初始化服务
func InitApiServer() (err error) {
	var (
		mux *http.ServeMux
		listener net.Listener
		err error
		httpServer *http.Server
	)
	//配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save",handleJobSave)

	//启动TCP监听
	if listener,err = net.Listen("tcp",":8070");err != nil{
		return
	}

	//创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Handler:mux,
	}

	//赋值单例
	G_apiServer = &ApiServer{
		httpServer:httpServer,
	}

	//启动了服务端
	go httpServer.Serve(listener)
	return
}