module github.com/antlapit/otus-architect/notification-service

replace github.com/antlapit/otus-architect/toolbox => ../toolbox
replace github.com/antlapit/otus-architect/api => ../api


require (
	github.com/antlapit/otus-architect/toolbox v1.0.0
	github.com/antlapit/otus-architect/api v1.0.0
	github.com/gin-gonic/gin v1.6.3
	github.com/lib/pq v1.9.0
)

go 1.15
