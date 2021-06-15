module github.com/antlapit/otus-architect/product-search-service

replace github.com/antlapit/otus-architect/toolbox => ../toolbox

replace github.com/antlapit/otus-architect/api => ../api

require (
	github.com/Masterminds/squirrel v1.5.0
	github.com/antlapit/otus-architect/api v1.0.0
	github.com/antlapit/otus-architect/toolbox v1.0.0
	github.com/gin-gonic/gin v1.6.3
	github.com/go-redis/redis/v8 v8.10.0
	github.com/lib/pq v1.9.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/prometheus/common v0.15.0
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b // indirect
	golang.org/x/sys v0.0.0-20210423185535-09eb48e85fd7 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
)

go 1.15
