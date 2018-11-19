package worker

import (
	"go-crontab/common"
	"fmt"
)

//任务调度
type Scheduler struct {
	jobEventChan chan *common.JobEvent  //etcd任务事件队列
	jobPlanTable map[string]*common.JobSchedulePlan //任务调度计划表
	jobExecutingTable map[string]*common.JobExecuteInfo //任务执行表
	jobResultChan chan *common.JobExecuteResult //任务结果队列
}


var (
	G_scheduler *Scheduler
)

//处理任务事件
func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent)  {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting bool
		jobExisted bool
		err error
	)

	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:
		//保存任务事件
		if jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil{
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOB_EVENT_DELETE:
		//删除任务事件
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted{
			delete(scheduler.jobPlanTable,jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL:
		//强杀任务事件
		//取消掉Command执行，判断任务是否在执行中
		if jobExecuteInfo,jobExecuting = scheduler.jobExecutingTable[jobEvent.Job.Name]; jobExecuting{
			//触发command杀死shell子进程，任务得到退出
			jobExecuteInfo.CancelFunc()
		}
	}
}

//尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan)  {
	//调度和执行是2件事情
	var (
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting bool
	)

	//执行的任务可能运行很久，1分钟会调度60次，但是只能执行1次，防止并发!

	//如果任务正在执行，跳过本次调度
	if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.Name]; jobExecuting{
		fmt.Println("尚未退出，跳过本次执行:",jobPlan.Job.Name)
		return
	}

	//构建执行状态信息
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)

	//保存执行状态
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

	//执行任务
	fmt.Println("执行任务:",jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)
}
