############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder

WORKDIR /src
COPY ./src .

RUN go build -o /go/bin/main

############################
# STEP 2 build a small image
############################
FROM scratch

COPY --from=builder /go/bin/main /go/bin/main

EXPOSE 8000

ENTRYPOINT ["/go/bin/main"]
