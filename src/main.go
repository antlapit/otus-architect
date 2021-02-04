package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Initializing")
	handleRequests()
	fmt.Println("Started")
}

func handleRequests() {
	http.HandleFunc("/health/", getHealthStatus)
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func getHealthStatus(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "{\"status\": \"OK\"}")
}
