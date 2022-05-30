package scoremanager

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"time"

	"github.com/mittz/role-play-webapp/benchmark/availabilitychecker"
	"github.com/mittz/role-play-webapp/benchmark/benchmarker"
	"github.com/mittz/role-play-webapp/benchmark/jobmanager"
	"github.com/mittz/role-play-webapp/benchmark/usermanager"
	"github.com/mittz/role-play-webapp/benchmark/utils"
	"golang.org/x/sync/semaphore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	PROJECT_BUDGET_MRR_USD   = 1000
	WEIGHT_OF_WORKER         = 1
	BENCHMARK_TIMEOUT_SECOND = 60
	NUM_OF_BENCHMARKER       = 4
)

type ScoreManager struct {
	db  *gorm.DB
	sem *semaphore.Weighted
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

type Ranking struct {
	ID         uint `gorm:"primary_key"`
	LDAP       string
	Score      uint
	ExecutedAt time.Time
}

func NewScoreManager() ScoreManager {
	return ScoreManager{db: initDBConn(), sem: semaphore.NewWeighted(int64(getLimitNumOfManagers()))}
}

func (w ScoreManager) Run() {
	w.sem.Acquire(context.Background(), WEIGHT_OF_WORKER)
	defer w.sem.Release(WEIGHT_OF_WORKER)

	job := jobmanager.DequeueJobRequest()
	defer jobmanager.Delete(job.Userkey)

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

	ldap := usermanager.GetLDAPByUserkey(job.Userkey)
	result := JobHistory{
		Userkey:           job.Userkey,
		LDAP:              ldap,
		ExecutedAt:        time.Now(),
		BenchResultMsg:    "Success",
		PlatformResultMsg: "Success",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*BENCHMARK_TIMEOUT_SECOND)
	defer cancel()
	score := make(chan uint)
	for i := 0; i < NUM_OF_BENCHMARKER; i++ {
		// TODO: Error handling when one of the benchmark functions faced an issue
		go benchmarker.Benchmarker{}.Benchmark(ctx, job.Endpoint, score)
	}

	benchScore := uint(0)
	for i := 0; i < NUM_OF_BENCHMARKER; i++ {
		benchScore += <-score
	}

	result.BenchScore = benchScore

	labels := map[string]string{
		"service_role_webapp": "true",
		"service_role_db":     "true",
	}

	pfRate, err := availabilitychecker.RateArchitecture(job.ProjectID, labels)

	if err != nil {
		result.PlatformResultMsg = err.Error()
	}
	result.PlatformRate = uint(pfRate)

	result.TotalScore = result.BenchScore * result.PlatformRate

	// Store the result in the database server
	if err := w.db.Create(&result).Error; err != nil {
		log.Printf("failed to write the result %v in database: %v", result, err)
	}

	// Update the ranking table
	// Check if the user is present in the ranking table
	// If no, add the user info (ldap), executed_at and result (score) in the ranking table
	// If yes, update the ranking table with the user info, executed_at and result if needed
	var ranking Ranking
	if err := w.db.First(&ranking, "ldap =?", ldap).Error; err != nil {
		log.Printf("failed to read a result of ldap: %s from database", ldap)
	}

	if ranking.LDAP == "" {
		ranking = Ranking{
			LDAP:       ldap,
			Score:      result.TotalScore,
			ExecutedAt: result.ExecutedAt,
		}
		if err := w.db.Create(&ranking).Error; err != nil {
			log.Printf("failed to write the ranking %v in database: %v", ranking, err)
		}
	} else {
		ranking.Score = result.TotalScore
		ranking.ExecutedAt = result.ExecutedAt
		if err := w.db.Save(&ranking).Error; err != nil {
			log.Printf("failed to update the ranking %v in database: %v", ranking, err)
		}
	}

	log.Printf("Userkey: %s - BenchmarkScore: %d, PlatformRate: %d\n", result.Userkey, result.BenchScore, result.PlatformRate)
}

func initDBConn() *gorm.DB {
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s",
		utils.GetEnvDBHostname(),
		utils.GetEnvDBPort(),
		utils.GetEnvDBUsername(),
		utils.GetEnvDBName(),
		utils.GetEnvDBPassword(),
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

	if !conn.Migrator().HasTable(&Ranking{}) {
		if err := conn.Migrator().CreateTable(&Ranking{}); err != nil {
			log.Panicf("Failed to create table: %v", err)
		}
	}

	return conn
}

func getLimitNumOfManagers() int {
	// Limit the max number of workers 2
	return utils.GetMin(runtime.NumCPU(), 2)
}
