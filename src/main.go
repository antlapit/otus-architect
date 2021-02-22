package main

import (
	"antlapit/otus-architect/users"
	"antlapit/otus-architect/utils"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	. "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type DatabaseConfig struct {
	host     string
	port     string
	user     string
	password string
	name     string
}

func main() {
	serviceMode := os.Getenv("SERVICE_MODE")

	config := &DatabaseConfig{
		host:     os.Getenv("DB_HOST"),
		port:     os.Getenv("DB_PORT"),
		user:     os.Getenv("DB_USER"),
		password: os.Getenv("DB_PASSWORD"),
		name:     os.Getenv("DB_NAME"),
	}

	db, driver := initDatabase(config)

	if serviceMode == "INIT" {
		migrateDb(driver)
	} else {
		var repository = users.Repository{DB: db}

		engine := gin.Default()
		initTechResources(engine, config)
		initUsersApi(engine, &repository)
		engine.Run(":8000")
	}
}

func migrateDb(driver database.Driver) {
	m, err := NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		log.Fatal(err)
	}

	err = m.Up()
	if err != nil {
		switch err.Error() {
		case ErrNoChange.Error(), ErrNilVersion.Error(), ErrLockTimeout.Error():
			return
		}
		log.Fatal(err)
	}
}

func initDatabase(config *DatabaseConfig) (*sql.DB, database.Driver) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.host, config.port, config.user, config.password, config.name)
	fmt.Println(psqlInfo)

	var db *sql.DB
	var err error
	var driver database.Driver
	db, err = sql.Open("postgres", psqlInfo)
	for {
		driver, err = postgres.WithInstance(db, &postgres.Config{})
		if err == nil {
			break
		} else {
			time.Sleep(2 * time.Second)
		}
	}
	return db, driver
}

func initTechResources(engine *gin.Engine, config *DatabaseConfig) {
	engine.GET("/health", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"status": "OK",
		})
	})
	engine.GET("/db", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"host":     config.host,
			"port":     config.port,
			"user":     config.user,
			"password": config.password,
			"name":     config.name,
		})
	})

	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

var (
	metricRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "otus_architect_requests_total",
		Help: "The total number of processed requests",
	}, []string{"url", "method", "status"})

	metricLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "otus_architect_requests_latency",
		Help:    "Latency of processed requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"url", "method"})
)

func initUsersApi(engine *gin.Engine, repository *users.Repository) {
	rootUserRoute := engine.Group("/user")
	rootUserRoute.Use(metrics, errorHandler)

	rootUserRoute.POST("", userDataExtractor, func(context *gin.Context) {
		createUser(context, repository)
	})

	singleUserRoute := rootUserRoute.Group("/:id")
	singleUserRoute.Use(userIdExtractor, responseSerializer)
	singleUserRoute.GET("", func(context *gin.Context) {
		getUser(context, repository)
	})
	singleUserRoute.DELETE("", func(context *gin.Context) {
		deleteUser(context, repository)
	})
	singleUserRoute.PUT("", userDataExtractor, func(context *gin.Context) {
		updateUser(context, repository)
	})
}

// Извлечение ИД пользователя из URL
func userIdExtractor(context *gin.Context) {
	id, err := utils.GetPathInt64(context, "id")
	if err != nil {
		utils.AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
	}
	context.Set("userId", id)

	context.Next()
}

// Извлечение ИД пользователя из URL
func userDataExtractor(context *gin.Context) {
	var user users.UserData
	if err := context.ShouldBindJSON(&user); err != nil {
		utils.AbortErrorResponse(context, http.StatusBadRequest, err, "DA02")
	} else {
		context.Set("userData", user)
	}

	context.Next()
}

func errorHandler(context *gin.Context) {
	context.Next()

	err := context.Errors.Last()
	if err != nil {
		if err.Meta != nil {
			realError := err.Meta.(error)
			switch realError.(type) {
			case *users.UserNotFoundError:
				utils.ErrorResponse(context, http.StatusNotFound, err, "BL01")
				break
			case *users.UserInvalidError:
				utils.ErrorResponse(context, http.StatusConflict, err, "BL02")
			default:
				utils.ErrorResponse(context, http.StatusConflict, err, "FA01")
			}
		} else {
			utils.ErrorResponse(context, http.StatusInternalServerError, err, "FA02")
		}
	}
}

func metrics(context *gin.Context) {
	start := time.Now()
	url := context.Request.URL.String()
	for _, p := range context.Params {
		url = strings.Replace(url, p.Value, ":"+p.Key, 1)
	}
	method := context.Request.Method
	context.Next()

	status := strconv.Itoa(context.Writer.Status())
	elapsed := float64(time.Since(start)) / float64(time.Second)

	metricRequests.WithLabelValues(url, method, status).Inc()
	metricLatency.WithLabelValues(url, method).Observe(elapsed)
}

func responseSerializer(context *gin.Context) {
	context.Next()

	res, exists := context.Get("result")
	if exists {
		context.JSON(http.StatusOK, res)
	}
}

func updateUser(context *gin.Context, repository *users.Repository) {
	userId := context.GetInt64("userId")
	userData := context.MustGet("userData").(users.UserData)
	res, err := repository.Update(userId, userData)

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.Set("result", gin.H{
			"success": res,
		})
	}
}

func deleteUser(context *gin.Context, repository *users.Repository) {
	userId := context.GetInt64("userId")
	res, err := repository.Delete(userId)

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.Set("result", gin.H{
			"success": res,
		})
	}
}

func getUser(context *gin.Context, repository *users.Repository) {
	userId := context.GetInt64("userId")
	user, err := repository.Get(userId)

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.Set("result", user)
	}

}

func createUser(context *gin.Context, repository *users.Repository) {
	userData := context.MustGet("userData").(users.UserData)

	id, err := repository.Create(userData)

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.JSON(http.StatusCreated, gin.H{
			"id": id,
		})
	}
}
