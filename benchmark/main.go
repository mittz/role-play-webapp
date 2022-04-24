package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/semaphore"
)

const (
	MESSAGE_INVALID_ENDPOINT = "invalid endpoint"
	MESSAGE_INVALID_USERKEY  = "invalid userkey"
	MESSAGE_ALREADY_INQUEUE  = "already in the queue"
	TIMEOUT_SECOND           = 10
	WEIGHT_OF_WORKERS        = 1
	PORT                     = 8080 // TODO: This should be provided through CLI or File
	NUM_OF_PARTICIPANTS      = 2    // TODO: This should be provided through CLI or File
	LIMIT_OF_WORKERS         = 1    // TODO: This should be provided through CLI or File
)

var queue chan Request
var results chan JobHistory
var jobInQueue sync.Map
var worker Worker

type Request struct {
	Userkey   string
	Endpoint  string
	ProjectID string
}

type User struct {
	Userkey   string
	LDAP      string
	Team      string
	Region    string
	SubRegion string
	Role      string
}

type JobHistory struct {
	ID         string
	Userkey    string
	Score      uint
	ErrMsg     string
	ExecutedAt time.Time
}

type Worker struct {
	sem *semaphore.Weighted
}

func init() {
	queue = make(chan Request, NUM_OF_PARTICIPANTS)
	results = make(chan JobHistory, NUM_OF_PARTICIPANTS)
	jobInQueue = sync.Map{}
	worker = Worker{sem: semaphore.NewWeighted(LIMIT_OF_WORKERS)}
}

func isValidEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")
}

func isValidUserKey(userkey string) bool {
	return true
}

func isInQueue(userkey string) bool {
	if flag, ok := jobInQueue.Load(userkey); ok {
		return flag.(bool)
	}

	return false
}

func benchmark(endpoint string) (uint, error) {
	log.Printf("[Benchmark][Start] Endpoint: %s\n", endpoint)
	time.Sleep(time.Second * 10) // do benchmark
	log.Printf("[Benchmark][End] Endpoint: %s\n", endpoint)
	return 10, nil
}

func scoreArchitecture(projectID string) (uint, error) {
	log.Printf("[Scoring][Start] ProjectID: %s\n", projectID)
	time.Sleep(time.Second * 3) // do scoring
	log.Printf("[Scoring][End] ProjectID: %s\n", projectID)
	return 2, nil
}

func (w Worker) RunScoring() {
	w.sem.Acquire(context.Background(), WEIGHT_OF_WORKERS)

	job := <-queue
	benchScore, err := benchmark(job.Endpoint)
	var errMsgs []string
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}

	pfScore, err := scoreArchitecture(job.ProjectID)
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}

	result := JobHistory{
		Userkey:    job.Userkey,
		Score:      benchScore * pfScore,
		ErrMsg:     strings.Join(errMsgs, ";"),
		ExecutedAt: time.Now(),
	}

	results <- result

	w.sem.Release(WEIGHT_OF_WORKERS)
}

func (w Worker) WriteResult() {
	result := <-results
	jobInQueue.Store(result.Userkey, false)
}

func postBenchmark(c *gin.Context) {
	userkey := c.PostForm("userkey")
	endpoint := c.PostForm("endpoint")
	projectID := c.PostForm("project_id")

	if !isValidEndpoint(endpoint) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": MESSAGE_INVALID_ENDPOINT,
		})
		return
	}

	if !isValidUserKey(userkey) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": MESSAGE_INVALID_USERKEY,
		})
		return
	}

	if isInQueue(userkey) {
		c.JSON(http.StatusNotAcceptable, gin.H{
			"message": MESSAGE_ALREADY_INQUEUE,
		})
		return
	}

	go worker.RunScoring()
	go worker.WriteResult()

	queue <- Request{Userkey: userkey, Endpoint: endpoint, ProjectID: projectID}
	jobInQueue.Store(userkey, true) // The value will be updated with false once the job is done.

	c.JSON(http.StatusAccepted, gin.H{
		"endpoint": endpoint,
		"userkey":  userkey,
	})
}

func timeoutPostBenchmark(c *gin.Context) {
	c.String(http.StatusRequestTimeout, "Currently we are getting a lot of requests. Please try again later.")
}

func main() {
	r := gin.Default()

	r.POST("/benchmark", timeout.New(
		timeout.WithTimeout(time.Second*TIMEOUT_SECOND),
		timeout.WithHandler(postBenchmark),
		timeout.WithResponse(timeoutPostBenchmark),
	))

	r.Run(fmt.Sprintf(":%d", PORT))
}
