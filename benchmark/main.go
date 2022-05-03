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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/semaphore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
	MAX_PRODUCT_QUANTITY      = 10
	SCORE_GET_PRODUCTS        = 5
	SCORE_POST_CHECKOUT       = 2
	SCORE_GET_PRODUCT         = 1
	SCORE_GET_CHECKOUTS       = 4
)

var (
	queue       chan Request
	results     chan JobHistory
	jobInQueue  sync.Map
	worker      Worker
	imageHashes map[string]string
	users       map[string]User
)

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
	ID         uint `gorm:"primary_key"`
	Userkey    string
	LDAP       string
	BenchScore uint
	PFScore    uint
	TotalScore uint
	ExecutedAt time.Time
}

type Worker struct {
	sem  *semaphore.Weighted
	conn *gorm.DB
}

type ImageHash struct {
	Name string
	Hash string
}

type UsersBlob struct {
	Users []User `json:"users"`
}

type ImageHashesBlob struct {
	ImageHashes []ImageHash `json:"image_hashes"`
}

func init() {
	queue = make(chan Request, getNumOfUsers())
	results = make(chan JobHistory, getNumOfUsers())
	jobInQueue = sync.Map{}
	imageHashes = initImageHashes()
	worker = Worker{sem: semaphore.NewWeighted(int64(getLimitOfWorkers())), conn: initDBConn()}
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

func getEnvDBHostname() string {
	return getEnv("DB_HOSTNAME", "score-database")
}

func getEnvDBPort() int {
	val := getEnv("DB_PORT", "5432")
	port, err := strconv.Atoi(val)
	if err != nil {
		log.Fatalf("DB_PORT should be integer, but %s: %v", val, err)
	}

	return port
}

func getEnvDBUsername() string {
	return getEnv("DB_USERNAME", "score")
}

func getEnvDBPassword() string {
	return getEnv("DB_PASSWORD", "score")
}

func getEnvDBName() string {
	return getEnv("DB_NAME", "score")
}

func getEnvDataStudioURL() string {
	return getEnv("DS_URL", "")
}

func initDBConn() *gorm.DB {
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s",
		getEnvDBHostname(),
		getEnvDBPort(),
		getEnvDBUsername(),
		getEnvDBName(),
		getEnvDBPassword(),
	)

	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Panicf("Failed to open database connection: %v", err)
	}

	if !conn.Migrator().HasTable(&JobHistory{}) {
		if err := conn.Migrator().CreateTable(&JobHistory{}); err != nil {
			log.Panicf("Failed to create table: %v", err)
		}
	}

	return conn
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

func getNumOfUsers() int {
	return len(users)
}

func getNumOfProducts() int {
	return len(imageHashes)
}

func getLimitOfWorkers() int {
	// Limit the max number of workers 10
	l := getNumOfProducts()/5 + 1
	if l >= 10 {
		return 10
	}

	return l
}

func isValidEndpoint(endpoint string) bool {
	return (strings.HasPrefix(endpoint, "http://") ||
		strings.HasPrefix(endpoint, "https://")) &&
		(!strings.Contains(endpoint, "localhost") &&
			!strings.Contains(endpoint, "127.0.0.1"))
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
	selection.Find("img").Each(func(_ int, s *goquery.Selection) {
		if val, ok := s.Attr("src"); ok {
			if !strings.HasPrefix(val, "http") {
				val = fmt.Sprintf("%s://%s%s", baseURL.Scheme, baseURL.Host, val)
			}

			imagePaths = append(imagePaths, val)
		}
	})

	productID := rand.Intn(getNumOfProducts()-1) + 1 // Exclude 0
	if len(imagePaths) <= productID {
		return 0
	}
	imagePath := imagePaths[productID]
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
	if err != nil {
		return 0
	}

	orderExists := false
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0
	}

	doc.Find("table").Each(func(_ int, tableHTML *goquery.Selection) {
		tableHTML.Find("tr").Each(func(_ int, rowHTML *goquery.Selection) {
			id := rowHTML.Find("td.product_id").Text()
			quantity := rowHTML.Find("td.product_quantity").Text()

			if id == fmt.Sprint(productID) && quantity == fmt.Sprint(productQuantity) {
				orderExists = true
				return
			}
		})
	})
	resp.Body.Close()
	if resp.StatusCode == http.StatusAccepted && orderExists { // and content is the same as expected
		return SCORE_POST_CHECKOUT
	}

	return 0
}

func benchGetProduct(baseURL url.URL) uint {
	getProductURL := baseURL
	productID := rand.Intn(getNumOfProducts()-1) + 1 // Exclude 0
	getProductURL.Path = path.Join(getProductURL.Path, "/product", fmt.Sprintf("%d", productID))
	resp, err := http.Get(getProductURL.String())
	if err != nil {
		log.Printf("%v\n", err)
		return 0
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("%v\n", err)
		return 0
	}

	var imagePath string
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		if val, ok := s.Attr("src"); ok {
			if !strings.HasPrefix(val, "http") {
				val = fmt.Sprintf("%s://%s%s", baseURL.Scheme, baseURL.Host, val)
			}

			imagePath = val
		}
	})

	respImage, err := http.Get(imagePath)
	if err != nil {
		log.Printf("%v\n", err)
		return 0
	}
	h := md5.New()
	if _, err := io.Copy(h, respImage.Body); err != nil {
		log.Printf("%v\n", err)
		return 0
	}
	respImage.Body.Close()

	resp.Body.Close()

	if resp.StatusCode == http.StatusOK && fmt.Sprintf("%x", h.Sum(nil)) == imageHashes[path.Base(imagePath)] {
		return SCORE_GET_PRODUCT
	}

	return 0
}

func benchGetCheckouts(baseURL url.URL, productID int, productQuantity int) uint {
	getCheckoutsURL := baseURL
	getCheckoutsURL.Path = path.Join(getCheckoutsURL.Path, "/checkouts")
	resp, err := http.Get(getCheckoutsURL.String())
	if err != nil {
		return 0
	}

	// Check if the order which is just created exists
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0
	}

	orderExists := false
	doc.Find("table").Each(func(_ int, tableHTML *goquery.Selection) {
		tableHTML.Find("tr").Each(func(_ int, rowHTML *goquery.Selection) {
			id := rowHTML.Find("td.product_id").Text()
			quantity := rowHTML.Find("td.product_quantity").Text()

			if id == fmt.Sprint(productID) && quantity == fmt.Sprint(productQuantity) {
				orderExists = true
				return
			}
		})
	})
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK && orderExists { // and content is the same as expected
		return SCORE_GET_CHECKOUTS
	}

	return 0
}

func benchmark(ctx context.Context, endpoint string) (uint, error) {
	log.Printf("Benchmark started - Endpoint: %s\n", endpoint)

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

			productID := rand.Intn(getNumOfProducts()-1) + 1         // Exclude 0
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

	log.Printf("Benchmark finished - Endpoint: %s\n", endpoint)
	return score, nil
}

func scoreArchitecture(projectID string) (uint, error) {
	log.Printf("Scoring started - ProjectID: %s\n", projectID)
	time.Sleep(time.Second * 3) // TODO: Implement this
	log.Printf("Scoring finished - ProjectID: %s\n", projectID)
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
	resp.Body.Close() // TODO: Fix the scenario when the init path is invalid and this is called

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*BENCHMARK_TIMEOUT_SECOND)
	defer cancel()
	benchScore, err := benchmark(ctx, job.Endpoint)
	if err != nil {
		log.Printf("%v", err.Error())
	}

	pfScore, err := scoreArchitecture(job.ProjectID)
	if err != nil {
		log.Printf("%v", err.Error())
	}

	result := JobHistory{
		Userkey:    job.Userkey,
		BenchScore: benchScore,
		PFScore:    pfScore,
		TotalScore: benchScore * pfScore,
		ExecutedAt: time.Now(),
	}

	results <- result

	w.sem.Release(WEIGHT_OF_WORKER)
}

func (w Worker) WriteResult() {
	result := <-results
	result.LDAP = users[result.Userkey].LDAP
	// Store the result in the database server
	if err := w.conn.Create(&result).Error; err != nil {
		log.Printf("failed to write the result %v in database: %v", result, err)
	}

	log.Printf("Userkey: %s - BenchScore: %d, PFScore: %d\n", result.Userkey, result.BenchScore, result.PFScore)
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

	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Welcome to Scoring Server",
		"ldaps": ldaps,
		"dsurl": getEnvDataStudioURL(),
	})
}

func getEnv(key, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultVal
}

func getEnvPort() int {
	val := getEnv("GIN_PORT", "8080")
	port, err := strconv.Atoi(val)
	if err != nil {
		log.Fatalf("GIN_PORT: %s should be integer: %v", val, err)
	}

	return port
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

	r.Run(fmt.Sprintf(":%d", getEnvPort()))
}
