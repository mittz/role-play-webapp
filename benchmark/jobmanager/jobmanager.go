package jobmanager

import (
	"sort"
	"sync"
	"time"

	"github.com/mittz/role-play-webapp/benchmark/usermanager"
)

var (
	requestsQueue chan JobRequest
	jobsMap       sync.Map
)

// TODO: Consider if we can replace Job with JobRequest
type Job struct {
	LDAP      string
	StartedAt time.Time
}

type JobRequest struct {
	Userkey   string
	Endpoint  string
	ProjectID string
	LDAP      string
	StartedAt time.Time
}

func init() {
	jobsMap = sync.Map{}
	requestsQueue = make(chan JobRequest, usermanager.GetNumOfUsers())
}

func EnqueueJobRequest(request JobRequest) {
	requestsQueue <- request
}

func DequeueJobRequest() JobRequest {
	return <-requestsQueue
}

func GetJobs() []Job {
	var jobs []Job

	jobsMap.Range(func(key, value interface{}) bool {
		jobs = append(jobs, Job{LDAP: usermanager.GetLDAPByUserkey(key.(string)), StartedAt: value.(time.Time)})
		return true
	})

	// Sort the users by appended date
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].StartedAt.Before(jobs[j].StartedAt) })

	return jobs
}

func ExistUserInQueue(userkey string) bool {
	_, exist := jobsMap.Load(userkey)

	return exist
}

func Store(userkey string, t time.Time) {
	jobsMap.Store(userkey, t)
}

func Delete(userkey string) {
	jobsMap.Delete(userkey)
}
