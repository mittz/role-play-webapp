package scoreportal

import (
	"fmt"
	"net/http"
	"os/user"
	"strings"
	"time"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"github.com/mittz/role-play-webapp/benchmark/jobmanager"
	"github.com/mittz/role-play-webapp/benchmark/scoremanager"
	"github.com/mittz/role-play-webapp/benchmark/usermanager"
	"github.com/mittz/role-play-webapp/benchmark/utils"
)

type ScorePortal struct {
	users map[string]user.User
}

const (
	REQUEST_TIMEOUT_SECOND   = 10
	MESSAGE_INVALID_ENDPOINT = "invalid endpoint"
	MESSAGE_INVALID_USERKEY  = "invalid userkey"
	MESSAGE_ALREADY_INQUEUE  = "already in the queue"
)

var (
	sm scoremanager.ScoreManager
)

func init() {
	sm = scoremanager.NewScoreManager()
}

func isValidEndpoint(endpoint string) bool {
	return (strings.HasPrefix(endpoint, "http://") ||
		strings.HasPrefix(endpoint, "https://")) &&
		(!strings.Contains(endpoint, "localhost") &&
			!strings.Contains(endpoint, "127.0.0.1"))
}

func isValidUserKey(userkey string) bool {
	return usermanager.ExistUser(userkey)
}

func isUserInQueue(userkey string) bool {
	return jobmanager.ExistUserInQueue(userkey)
}

func getRequestForm(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"jobs":  jobmanager.GetJobs(),
		"dsurl": utils.GetEnvDataStudioURL(),
	})
}

func timeoutPostBenchmark(c *gin.Context) {
	c.HTML(http.StatusRequestTimeout, "timeout.html", nil)
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

	if isUserInQueue(userkey) {
		c.JSON(http.StatusNotAcceptable, gin.H{
			"message": MESSAGE_ALREADY_INQUEUE,
		})
		return
	}

	go sm.Run()

	jobmanager.EnqueueJobRequest(jobmanager.JobRequest{Userkey: userkey, Endpoint: endpoint, ProjectID: projectID})
	jobmanager.Store(userkey, time.Now())

	c.HTML(http.StatusAccepted, "benchmark.html", gin.H{
		"endpoint": endpoint,
		"userkey":  userkey,
	})
}

func Run() {
	r := gin.Default()
	r.LoadHTMLGlob("scoreportal/templates/*")
	r.StaticFile("/favicon.ico", "favicon.ico")

	r.GET("/", getRequestForm)
	r.POST("/benchmark", timeout.New(
		timeout.WithTimeout(time.Second*REQUEST_TIMEOUT_SECOND),
		timeout.WithHandler(postBenchmark),
		timeout.WithResponse(timeoutPostBenchmark),
	))

	r.Run(fmt.Sprintf(":%d", utils.GetEnvPortalPort()))
}
