package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	asset "cloud.google.com/go/asset/apiv1"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/semaphore"
	"google.golang.org/api/iterator"
	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"
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
	BENCHMARK_TIMEOUT_SECOND  = 2
	WEIGHT_OF_WORKER          = 1
	MAX_PRODUCT_QUANTITY      = 10
	SCORE_GET_PRODUCTS        = 5
	SCORE_POST_CHECKOUT       = 2
	SCORE_GET_PRODUCT         = 1
	SCORE_GET_CHECKOUTS       = 4
	RATE_NO_RESOURCE          = 0
	RATE_ZONAL                = 1
	RATE_REGIONAL             = 2
	RATE_MULTI_REGIONAL       = 3
)

var (
	queue       chan Request
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
	ID                uint `gorm:"primary_key"`
	Userkey           string
	LDAP              string
	BenchScore        uint
	BenchResultMsg    string
	PlatformRate      uint
	PlatformResultMsg string
	TotalScore        uint
	ExecutedAt        time.Time
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
		return 0
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("%v\n", err)
		return 0
	}

	// Check hashsum of image
	imagePaths := doc.Find("div.content-container").Find("img.card-img-top.products-img").Map(func(_ int, s *goquery.Selection) string {
		val, _ := s.Attr("src")

		if !strings.HasPrefix(val, "http") {
			return fmt.Sprintf("%s://%s%s", baseURL.Scheme, baseURL.Host, val)
		}

		return val
	})

	productID := rand.Intn(getNumOfProducts()-1) + 1 // Exclude 0
	if len(imagePaths) <= productID {
		return 0
	}

	imagePath := imagePaths[productID]
	respImage, err := http.Get(imagePath)
	if err != nil {
		log.Printf("%v\n", err)
		return 0
	}
	defer respImage.Body.Close()

	h := md5.New()
	if _, err := io.Copy(h, respImage.Body); err != nil {
		log.Printf("%v\n", err)
		return 0
	}

	// TODO: Check stylesheets
	if resp.StatusCode == http.StatusOK && fmt.Sprintf("%x", h.Sum(nil)) == imageHashes[path.Base(imagePath)] {
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
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0
	}

	orderInfo := doc.Find("div.content-container").Find("p.card-text").Text()
	imagePath, ok := doc.Find("div.content-container").Find("img.checkout-img").Attr("src")
	if !ok {
		return 0
	}

	if !strings.HasPrefix(imagePath, "http") {
		imagePath = fmt.Sprintf("%s://%s%s", baseURL.Scheme, baseURL.Host, imagePath)
	}

	respImage, err := http.Get(imagePath)
	if err != nil {
		log.Printf("%v\n", err)
		return 0
	}
	defer respImage.Body.Close()

	h := md5.New()
	if _, err := io.Copy(h, respImage.Body); err != nil {
		log.Printf("%v\n", err)
		return 0
	}

	if resp.StatusCode == http.StatusAccepted &&
		strings.Contains(orderInfo, fmt.Sprintf("%d x", productQuantity)) &&
		fmt.Sprintf("%x", h.Sum(nil)) == imageHashes[path.Base(imagePath)] {
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
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("%v\n", err)
		return 0
	}

	imagePath, ok := doc.Find("div.content-container").Find("img.product-img").Attr("src")
	if !ok {
		return 0
	}

	if !strings.HasPrefix(imagePath, "http") {
		imagePath = fmt.Sprintf("%s://%s%s", baseURL.Scheme, baseURL.Host, imagePath)
	}

	respImage, err := http.Get(imagePath)
	if err != nil {
		log.Printf("%v\n", err)
		return 0
	}
	defer respImage.Body.Close()

	h := md5.New()
	if _, err := io.Copy(h, respImage.Body); err != nil {
		log.Printf("%v\n", err)
		return 0
	}

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
	defer resp.Body.Close()

	// Check if the order which is just created exists
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0
	}

	order := doc.Find("table").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		return s.Find("td.product_id").Text() != fmt.Sprint(productID) ||
			s.Find("td.product_quantity").Text() != fmt.Sprint(productQuantity)
	})

	imagePath, ok := order.Find("td.product_image").Find("img").Attr("src")
	if !ok {
		return 0
	}

	if !strings.HasPrefix(imagePath, "http") {
		imagePath = fmt.Sprintf("%s://%s%s", baseURL.Scheme, baseURL.Host, imagePath)
	}

	respImage, err := http.Get(imagePath)
	if err != nil {
		log.Printf("%v\n", err)
		return 0
	}
	defer respImage.Body.Close()

	h := md5.New()
	if _, err := io.Copy(h, respImage.Body); err != nil {
		log.Printf("%v\n", err)
		return 0
	}

	if resp.StatusCode == http.StatusOK && order != nil && fmt.Sprintf("%x", h.Sum(nil)) == imageHashes[path.Base(imagePath)] {
		return SCORE_GET_CHECKOUTS
	}

	return 0
}

func benchmark(ctx context.Context, endpoint string) (uint, error) {
	score := uint(0)
	for {
		select {
		case <-ctx.Done():
			return score, nil
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
				return score, fmt.Errorf("unable to receive expected results from the endpoint (%s)", endpoint)
			}
		}
	}
}

type AvailabilityChecker struct {
	projectID string
}

type LabelLocation struct {
	label    string
	location string
}

type LabelRegion struct {
	label  string
	region string
}

type ResourceInfo struct {
	labels               map[string]string
	assetType            string
	location             string
	additionalAttributes map[string]interface{}
}

func NewAvailabilityChecker() *AvailabilityChecker {
	return new(AvailabilityChecker)
}

func (ac *AvailabilityChecker) SetProjectID(projectID string) {
	ac.projectID = projectID
}

func (ac *AvailabilityChecker) GetAllResourceInfo() ([]ResourceInfo, error) {
	// $ gcloud asset search-all-resources \
	// --scope projects/[PROJECT_ID] \
	scope := fmt.Sprintf("projects/%s", ac.projectID)
	ctx := context.Background()
	client, err := asset.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	req := &assetpb.SearchAllResourcesRequest{
		Scope: scope,
	}

	it := client.SearchAllResources(ctx, req)
	var resources []ResourceInfo
	for {
		resource, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		ri := ResourceInfo{
			assetType:            resource.AssetType,
			location:             resource.Location,
			labels:               resource.Labels,
			additionalAttributes: resource.AdditionalAttributes.AsMap(),
		}
		resources = append(resources, ri)
	}

	return resources, nil
}

func (ac *AvailabilityChecker) CheckRuleViolation(resourceInfo []ResourceInfo,
	validAssetTypes map[string]interface{}, invalidAssetTypes map[string]interface{}) error {
	// TODO: Check if the resources are not exceeded the limits
	for _, ri := range resourceInfo {
		if _, ok := invalidAssetTypes[ri.assetType]; ok {
			return fmt.Errorf("%s can't be used in this contest", ri.assetType)
		}
	}

	return nil
}

func getMin(x, y int) int {
	if x < y {
		return x
	}

	return y
}

func (ac *AvailabilityChecker) GetMinAvailabilityScore(info []ResourceInfo, assetTypes map[string]interface{}) uint {
	roles := map[string]interface{}{
		"service_role_webapp": struct{}{},
		"service_role_db":     struct{}{},
	}

	labels := make(map[string]interface{})

	// locByLabels[tag][location]: interface{}
	// e.g. locByLabels[service_role_webapp][us-central1-c]: struct{}{}
	locationLabels := make(map[LabelLocation]interface{})
	for _, r := range info {
		if _, ok := assetTypes[r.assetType]; !ok {
			continue
		}

		for key, val := range r.labels {
			if _, ok := roles[key]; ok && val == "true" {
				labels[key] = struct{}{}
				locationLabels[LabelLocation{label: key, location: r.location}] = struct{}{}
			}
		}
	}

	// Either service_role_webapp or service_role_db is not present
	if len(labels) < 2 {
		return RATE_NO_RESOURCE
	}

	labelRegion := make(map[LabelRegion]interface{})
	for ll := range locationLabels {
		l := strings.Split(ll.location, "-")
		region := strings.Join(l[0:2], "-")
		labelRegion[LabelRegion{label: ll.label, region: region}] = struct{}{}
	}

	countRegionsByLabel := make(map[string]int) // countRegionsByLabel["service_role_webapp"]: 2 (regions)
	for lr := range labelRegion {
		countRegionsByLabel[lr.label]++
	}
	minNumOfRegions := math.MaxInt32 // Set enough large number for the number of regions
	for _, num := range countRegionsByLabel {
		minNumOfRegions = getMin(minNumOfRegions, num)
	}
	if minNumOfRegions >= 2 {
		return RATE_MULTI_REGIONAL
	}

	countLocationsByLabel := make(map[string]int) // countRegionsByLabel["service_role_webapp"]: 2 (locations)
	for ll := range locationLabels {
		countLocationsByLabel[ll.label]++
	}
	minNumOfLocations := math.MaxInt32 // Set enough large number for the number of locations
	for _, num := range countLocationsByLabel {
		minNumOfLocations = getMin(minNumOfLocations, num)
	}
	if minNumOfLocations >= 2 {
		return RATE_REGIONAL
	}

	return RATE_ZONAL
}

func scoreArchitecture(projectID string) (uint, error) {
	log.Printf("Scoring started - ProjectID: %s\n", projectID)
	ac := NewAvailabilityChecker()
	ac.SetProjectID(projectID)

	allInfo, err := ac.GetAllResourceInfo()
	if err != nil {
		return 0, fmt.Errorf("failed to get all resource info via Availability Checker: %v", err)
	}

	validAssetTypes := map[string]interface{}{
		"compute.googleapis.com/Instance":             struct{}{},
		"container.googleapis.com/NodePool":           struct{}{},
		"appengine.googleapis.com/Service":            struct{}{},
		"run.googleapis.com/Service":                  struct{}{},
		"cloudfunctions.googleapis.com/CloudFunction": struct{}{},
		"sqladmin.googleapis.com/Instance":            struct{}{},
		"spanner.googleapis.com/Instance":             struct{}{},
		"bigtableadmin.googleapis.com/Instance":       struct{}{},
	}

	invalidAssetTypes := map[string]interface{}{
		"redis.googleapis.com/Instance": struct{}{},
	}

	if err := ac.CheckRuleViolation(allInfo, validAssetTypes, invalidAssetTypes); err != nil {
		// Rule violation: 0 (Using invalid machine types or unlabeled resources)
		return 0, fmt.Errorf("rule violation: %v", err)
	}

	score := ac.GetMinAvailabilityScore(allInfo, validAssetTypes)

	log.Printf("Scoring finished - ProjectID: %s\n", projectID)
	return score, nil
}

func (w Worker) RunScoring() {
	w.sem.Acquire(context.Background(), WEIGHT_OF_WORKER)
	defer w.sem.Release(WEIGHT_OF_WORKER)

	job := <-queue
	defer jobInQueue.Delete(job.Userkey)

	// Initialize data in user app
	u, err := url.Parse(job.Endpoint)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	u.Path = path.Join(u.Path, "/admin/init")
	resp, err := http.Post(u.String(), "", nil)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	resp.Body.Close()

	result := JobHistory{
		Userkey:           job.Userkey,
		LDAP:              users[job.Userkey].LDAP,
		ExecutedAt:        time.Now(),
		BenchResultMsg:    "Success",
		PlatformResultMsg: "Success",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*BENCHMARK_TIMEOUT_SECOND)
	defer cancel()
	benchScore, err := benchmark(ctx, job.Endpoint)
	if err != nil {
		result.BenchResultMsg = err.Error()
	}
	result.BenchScore = benchScore

	pfRate, err := scoreArchitecture(job.ProjectID)
	if err != nil {
		result.PlatformResultMsg = err.Error()
	}
	result.PlatformRate = pfRate

	result.TotalScore = benchScore * pfRate

	// Store the result in the database server
	if err := w.conn.Create(&result).Error; err != nil {
		log.Printf("failed to write the result %v in database: %v", result, err)
	}

	log.Printf("Userkey: %s - BenchmarkScore: %d, PlatformRate: %d\n", result.Userkey, result.BenchScore, result.PlatformRate)
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

	queue <- Request{Userkey: userkey, Endpoint: endpoint, ProjectID: projectID}
	jobInQueue.Store(userkey, time.Now())

	c.HTML(http.StatusAccepted, "benchmark.html", gin.H{
		"endpoint": endpoint,
		"userkey":  userkey,
	})
}

func timeoutPostBenchmark(c *gin.Context) {
	c.HTML(http.StatusRequestTimeout, "timeout.html", nil)
}

func getRequestForm(c *gin.Context) {
	// Sort the users by appended date
	type Job struct {
		LDAP      string
		StartedAt time.Time
	}

	var jobs []Job
	jobInQueue.Range(func(key, value interface{}) bool {
		jobs = append(jobs, Job{LDAP: users[key.(string)].LDAP, StartedAt: value.(time.Time)})
		return true
	})

	sort.Slice(jobs, func(i, j int) bool { return jobs[i].StartedAt.Before(jobs[j].StartedAt) })

	c.HTML(http.StatusOK, "index.html", gin.H{
		"jobs":  jobs,
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
	r.StaticFile("/favicon.ico", "favicon.ico")

	r.GET("/", getRequestForm)
	r.POST("/benchmark", timeout.New(
		timeout.WithTimeout(time.Second*REQUEST_TIMEOUT_SECOND),
		timeout.WithHandler(postBenchmark),
		timeout.WithResponse(timeoutPostBenchmark),
	))

	r.Run(fmt.Sprintf(":%d", getEnvPort()))
}
