package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

const (
	USERS_DATA_FILENAME      = "users.json"
	MESSAGE_INVALID_ENDPOINT = "invalid endpoint"
	MESSAGE_INVALID_USERKEY  = "invalid userkey"
	MESSAGE_ALREADY_INQUEUE  = "already in the queue"
	REQUEST_TIMEOUT_SECOND   = 10
	BENCHMARK_TIMEOUT_SECOND = 2
	WEIGHT_OF_WORKER         = 1
	REGISTERED_PRODUCTS_NUM  = 200
	MAX_PRODUCT_QUANTITY     = 10
	SCORE_GET_PRODUCTS       = 5
	SCORE_POST_CHECKOUT      = 2
	SCORE_GET_PRODUCT        = 1
	SCORE_GET_CHECKOUTS      = 4
	PRODUCTS_NUM_PER_PAGE    = 2
	PORT                     = 8081 // TODO: This should be provided through CLI or File
	NUM_OF_USERS             = 2    // TODO: This should be provided through CLI or File
	LIMIT_OF_WORKERS         = 1    // TODO: This should be provided through CLI or File
)

var queue chan Request
var results chan JobHistory
var jobInQueue sync.Map
var worker Worker
var imageHashes map[string]string
var users map[string]User

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

type Blob struct {
	Users []User `json:"users"`
}

func init() {
	queue = make(chan Request, NUM_OF_USERS)
	results = make(chan JobHistory, NUM_OF_USERS)
	jobInQueue = sync.Map{}
	imageHashes = initImageHashes()
	worker = Worker{sem: semaphore.NewWeighted(LIMIT_OF_WORKERS)}
	users = initUsers()
}

func initImageHashes() map[string]string {
	imageHashes = make(map[string]string)
	imageHashes["hunters-race-Vk3QiwyrAUA-unsplash.jpg"] = "94b74ff9d2dec482cb76c8e595d14522"
	imageHashes["louis-mornaud-ADvixEYm5qE-unsplash.jpg"] = "3fb74bb187744fe3a1004255761f34b0"

	return imageHashes
}

func initUsers() map[string]User {
	jsonFromFile, err := os.ReadFile(USERS_DATA_FILENAME)
	if err != nil {
		log.Fatal(err)
	}

	var jsonData Blob
	err = json.Unmarshal(jsonFromFile, &jsonData)
	if err != nil {
		log.Fatal(err)
	}

	users = make(map[string]User)
	for _, user := range jsonData.Users {
		users[user.Userkey] = user
	}

	return users
}

func isValidEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")
}

func isValidUserKey(userkey string) bool {
	_, exist := users[userkey]
	return exist
}

func isInQueue(userkey string) bool {
	_, exist := jobInQueue.Load(userkey)
	return exist
}

func benchmark(ctx context.Context, endpoint string) (uint, error) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("Benchmark started", zap.String("endpoint", endpoint))

	score := uint(0)
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		default: // do benchmark
			rand.Seed(time.Now().UnixNano())
			baseURL, err := url.Parse(endpoint)
			if err != nil {
				log.Printf("%v\n", err)
			}

			// GET: /products 5
			getProductsURL := *baseURL
			getProductsURL.Path = path.Join(getProductsURL.Path, "/products")
			resp, err := http.Get(getProductsURL.String())
			if err != nil {
				log.Printf("%v\n", err)
			}
			doc, err := goquery.NewDocumentFromReader(resp.Body)
			if err != nil {
				panic(err)
			}

			// Check hashsum of image
			selection := doc.Find("tr")
			var imagePaths []string
			selection.Find("img").Each(func(index int, s *goquery.Selection) {
				if val, ok := s.Attr("src"); ok {
					if !strings.HasPrefix(val, "http") {
						val = endpoint + val
					}

					imagePaths = append(imagePaths, val)
				}
			})
			imagePath := imagePaths[rand.Intn(PRODUCTS_NUM_PER_PAGE)]
			respImage, err := http.Get(imagePath)
			if err != nil {
				log.Printf("%v\n", err)
			}
			h := md5.New()
			if _, err := io.Copy(h, respImage.Body); err != nil {
				log.Printf("%v\n", err)
			}
			respImage.Body.Close()

			// TODO: Check stylesheets
			resp.Body.Close()
			if err == nil && resp.StatusCode == http.StatusOK && fmt.Sprintf("%x", h.Sum(nil)) == imageHashes[path.Base(imagePath)] {
				score += SCORE_GET_PRODUCTS
			}

			// POST: /checkout 2
			productID := rand.Intn(REGISTERED_PRODUCTS_NUM)
			productQuantity := rand.Intn(MAX_PRODUCT_QUANTITY)
			data := url.Values{
				"product_id":       {fmt.Sprintf("%d", productID)},
				"product_quantity": {fmt.Sprintf("%d", productQuantity)},
			}
			postCheckout := *baseURL
			postCheckout.Path = path.Join(postCheckout.Path, "/checkout")
			resp, err = http.PostForm(postCheckout.String(), data)
			if err == nil && resp.StatusCode == http.StatusAccepted { // and content is the same as expected
				score += SCORE_POST_CHECKOUT
			}
			resp.Body.Close()

			// GET: /product/:product_id 1
			getProductURL := *baseURL
			getProductURL.Path = path.Join(getProductURL.Path, "/product", fmt.Sprintf("%d", productID))
			resp, err = http.Get(getProductURL.String())
			if err == nil && resp.StatusCode == http.StatusOK { // and content is the same as expected
				score += SCORE_GET_PRODUCT
			}
			resp.Body.Close()

			// GET: /checkouts 4
			getCheckoutsURL := *baseURL
			getCheckoutsURL.Path = path.Join(getCheckoutsURL.Path, "/checkouts")
			resp, err = http.Get(getCheckoutsURL.String())
			if err == nil && resp.StatusCode == http.StatusOK { // and content is the same as expected
				score += SCORE_GET_CHECKOUTS
			}
			resp.Body.Close()

			if score == 0 {
				break LOOP
			}
		}
	}

	logger.Info("Benchmark finished", zap.String("endpoint", endpoint))
	return score, nil
}

func scoreArchitecture(projectID string) (uint, error) {
	log.Printf("[Scoring][Start] ProjectID: %s\n", projectID)
	time.Sleep(time.Second * 3) // do scoring
	log.Printf("[Scoring][End] ProjectID: %s\n", projectID)
	return 2, nil
}

func (w Worker) RunScoring() {
	w.sem.Acquire(context.Background(), WEIGHT_OF_WORKER)

	job := <-queue
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*BENCHMARK_TIMEOUT_SECOND)
	defer cancel()
	benchScore, err := benchmark(ctx, job.Endpoint)
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

	w.sem.Release(WEIGHT_OF_WORKER)
}

func (w Worker) WriteResult() {
	result := <-results
	log.Printf("Score: %d\n", result.Score)
	jobInQueue.Delete(result.Userkey)
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
	jobInQueue.Store(userkey, struct{}{})

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

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to Scoring Server")
	})

	r.POST("/benchmark", timeout.New(
		timeout.WithTimeout(time.Second*REQUEST_TIMEOUT_SECOND),
		timeout.WithHandler(postBenchmark),
		timeout.WithResponse(timeoutPostBenchmark),
	))

	r.Run(fmt.Sprintf(":%d", PORT))
}
