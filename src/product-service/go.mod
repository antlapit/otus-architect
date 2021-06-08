module github.com/antlapit/otus-architect/product-service

replace github.com/antlapit/otus-architect/toolbox => ../toolbox

replace github.com/antlapit/otus-architect/api => ../api

require (
	github.com/antlapit/otus-architect/api v1.0.0
	github.com/antlapit/otus-architect/toolbox v1.0.0
	github.com/appleboy/gin-jwt/v2 v2.6.4
	github.com/gin-gonic/gin v1.6.3
	github.com/lib/pq v1.9.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/prometheus/common v0.15.0
	go.mongodb.org/mongo-driver v1.5.3
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
)

go 1.15
