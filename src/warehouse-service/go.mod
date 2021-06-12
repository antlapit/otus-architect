module github.com/antlapit/otus-architect/warehouse-service

replace github.com/antlapit/otus-architect/toolbox => ../toolbox

replace github.com/antlapit/otus-architect/api => ../api

require (
	github.com/antlapit/otus-architect/api v1.0.0
	github.com/antlapit/otus-architect/toolbox v1.0.0
	github.com/appleboy/gin-jwt/v2 v2.6.4
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/gin-gonic/gin v1.6.3
	github.com/lib/pq v1.9.0
)

go 1.15
