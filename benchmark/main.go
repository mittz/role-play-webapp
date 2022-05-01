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
	USERS_DATA_FILENAME       = "users.json"
	IMAGEHASHES_DATA_FILENAME = "image_hashes.json"
	MESSAGE_INVALID_ENDPOINT  = "invalid endpoint"
	MESSAGE_INVALID_USERKEY   = "invalid userkey"
	MESSAGE_ALREADY_INQUEUE   = "already in the queue"
	REQUEST_TIMEOUT_SECOND    = 10
	BENCHMARK_TIMEOUT_SECOND  = 10
	WEIGHT_OF_WORKER          = 1
	REGISTERED_PRODUCTS_NUM   = 100
	MAX_PRODUCT_QUANTITY      = 10
	SCORE_GET_PRODUCTS        = 5
	SCORE_POST_CHECKOUT       = 2
	SCORE_GET_PRODUCT         = 1
	SCORE_GET_CHECKOUTS       = 4
	PRODUCTS_NUM_PER_PAGE     = 100
	PORT                      = 8081 // TODO: This should be provided through CLI or File
	NUM_OF_USERS              = 6    // TODO: This should be provided through CLI or File
	LIMIT_OF_WORKERS          = 2    // TODO: This should be provided through CLI or File
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

type ImageHash struct {
	Name string
	Hash string
}

type UsersBlob struct {
	Users []User `json:"users"`
}

type ImageHashesBlob struct {
	ImageHashes []ImageHash
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
	jsonFromFile, err := os.ReadFile(IMAGEHASHES_DATA_FILENAME)
	if err != nil {
		log.Fatal(err)
	}

	var jsonData ImageHashesBlob
	err = json.Unmarshal(jsonFromFile, &jsonData)
	if err != nil {
		log.Fatal(err)
	}

	imageHashes = make(map[string]string)
	for _, imageHash := range jsonData.ImageHashes {
		imageHashes[imageHash.Name] = imageHash.Hash
	}

	return imageHashes
}

func initUsers() map[string]User {
	jsonFromFile, err := os.ReadFile(USERS_DATA_FILENAME)
	if err != nil {
		log.Fatal(err)
	}

	var jsonData UsersBlob
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
	return strings.HasPrefix(endpoint, "http://") ||
		strings.HasPrefix(endpoint, "https://") ||
		strings.Contains(endpoint, "localhost") ||
		strings.Contains(endpoint, "127.0.0.1")
}

func isValidUserKey(userkey string) bool {
	_, exist := users[userkey]
	return exist
}

func isInQueue(userkey string) bool {
	_, exist := jobInQueue.Load(userkey)
	return exist
}

func benchGetProducts(baseURL url.URL) uint {
	getProductsURL := baseURL
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
				val = fmt.Sprintf("%s://%s%s", baseURL.Scheme, baseURL.Host, val)
			}

			imagePaths = append(imagePaths, val)
		}
	})

	productID := rand.Intn(PRODUCTS_NUM_PER_PAGE)
	if len(imagePaths) <= productID {
		return 0
	}
	imagePath := imagePaths[productID] // TODO: Consider index out of range when the page is empty
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
		return SCORE_GET_PRODUCTS
	}

	return 0
}

func benchPostCheckout(baseURL url.URL, productID int, productQuantity int) uint {
	data := url.Values{
		"product_id":       {fmt.Sprintf("%d", productID)},
		"product_quantity": {fmt.Sprintf("%d", productQuantity)},
	}
	postCheckout := baseURL
	postCheckout.Path = path.Join(postCheckout.Path, "/checkout")
	resp, err := http.PostForm(postCheckout.String(), data)
	resp.Body.Close()
	if err == nil && resp.StatusCode == http.StatusAccepted { // and content is the same as expected
		return SCORE_POST_CHECKOUT
	}

	return 0
}

func benchGetProduct(baseURL url.URL) uint {
	getProductURL := baseURL
	productID := rand.Intn(PRODUCTS_NUM_PER_PAGE)
	getProductURL.Path = path.Join(getProductURL.Path, "/product", fmt.Sprintf("%d", productID))
	resp, err := http.Get(getProductURL.String())
	resp.Body.Close()
	if err == nil && resp.StatusCode == http.StatusOK { // and content is the same as expected
		return SCORE_GET_PRODUCT
	}

	return 0
}

func benchGetCheckouts(baseURL url.URL, productID int, productQuantity int) uint {
	getCheckoutsURL := baseURL
	getCheckoutsURL.Path = path.Join(getCheckoutsURL.Path, "/checkouts")
	resp, err := http.Get(getCheckoutsURL.String())
	resp.Body.Close()

	// TODO: Check if the order which is just created exists
	if err == nil && resp.StatusCode == http.StatusOK { // and content is the same as expected
		return SCORE_GET_CHECKOUTS
	}

	return 0
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

			productID := rand.Intn(REGISTERED_PRODUCTS_NUM-1) + 1    // Exclude 0
			productQuantity := rand.Intn(MAX_PRODUCT_QUANTITY-1) + 1 // Exclude 0

			score += benchGetProducts(*baseURL)
			score += benchPostCheckout(*baseURL, productID, productQuantity)
			score += benchGetProduct(*baseURL)
			score += benchGetCheckouts(*baseURL, productID, productQuantity)

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

	// Initialize data in user app
	u, err := url.Parse(job.Endpoint)
	if err != nil {
		log.Printf("%v", err)
	}
	u.Path = path.Join(u.Path, "/admin/init")
	resp, err := http.Post(u.String(), "", nil)
	if err != nil {
		log.Printf("%v", err)
	}
	resp.Body.Close()

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
	// TODO: Store the result in the database server
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

func getRequestForm(c *gin.Context) {
	var ldaps []string
	jobInQueue.Range(func(key, value interface{}) bool {
		ldaps = append(ldaps, users[key.(string)].LDAP)
		return true
	})

	// TODO: Link to the results dashboard (Data Studio)
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Welcome to Scoring Server",
		"ldaps": ldaps,
	})
}

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", getRequestForm)
	r.POST("/benchmark", timeout.New(
		timeout.WithTimeout(time.Second*REQUEST_TIMEOUT_SECOND),
		timeout.WithHandler(postBenchmark),
		timeout.WithResponse(timeoutPostBenchmark),
	))

	r.Run(fmt.Sprintf(":%d", PORT))
}
