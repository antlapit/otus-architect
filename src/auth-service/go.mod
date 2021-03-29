module gitlab.com/antlapit/otus-architect/auth-service

replace gitlab.com/antlapit/otus-architect/toolbox => ../toolbox

require (
	github.com/appleboy/gin-jwt/v2 v2.6.4
	github.com/gin-gonic/gin v1.6.3
	github.com/lib/pq v1.9.0
	gitlab.com/antlapit/otus-architect/toolbox v1.0.0
	golang.org/x/crypto v0.0.0-20201217014255-9d1352758620
)

go 1.15
