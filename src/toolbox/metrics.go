package toolbox

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"strconv"
	"strings"
	"time"
)

var (
	metricRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "otus_architect_requests_total",
		Help: "The total number of processed requests",
	}, []string{"apiKey", "status"})

	metricLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "otus_architect_requests_latency",
		Help:    "Latency of processed requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"apiKey"})
)

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

	apiKey := url + " " + method
	metricRequests.WithLabelValues(apiKey, status).Inc()
	metricLatency.WithLabelValues(apiKey).Observe(elapsed)
}

func initMetricsApi(engine *gin.Engine) {
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

func initHealthApi(engine *gin.Engine) {
	engine.GET("/health", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"status": "OK",
		})
	})
}

func initDbParamsApi(engine *gin.Engine, config *DatabaseConfig) {
	if config == nil {
		return
	}
	engine.GET("/db", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"host":     config.Host,
			"port":     config.Port,
			"user":     config.User,
			"password": config.Password,
			"name":     config.Name,
		})
	})
}

func initMongoParamsApi(engine *gin.Engine, config *MongoConfig) {
	if config == nil {
		return
	}
	engine.GET("/mongo", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"host":     config.Host,
			"port":     config.Port,
			"user":     config.User,
			"password": config.Password,
			"name":     config.Name,
		})
	})
}

func initTechResources(engine *gin.Engine, config *DatabaseConfig, mongoConfig *MongoConfig) {
	initMetricsApi(engine)
	initHealthApi(engine)
	initDbParamsApi(engine, config)
	initMongoParamsApi(engine, mongoConfig)
}
