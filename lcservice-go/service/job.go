package service

import "github.com/refractionPOINT/lc-service/lcservice-go/common"

type Job = common.Job
type JobEntry = common.JobEntry
type JobAttachment = common.JobAttachment

func NewJob(jobID ...string) *Job {
	return common.NewJob(jobID...)
}

func NewHexDumpAttachment(caption string, data []byte) JobAttachment {
	return common.NewHexDumpAttachment(caption, data)
}
