package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/BRAJESWARG/mongoapi/router"
)

func main() {
	fmt.Println("Welcome MongoDB API")
	r := router.Router()
	fmt.Println("Server is getting started...")

	log.Fatal(http.ListenAndServe(":4000", r))
	fmt.Println("Listening at port 4000 ...")
}

// go get github.com/mongodb/mongo-go-driver/v2/mongo
// go mod tidy
// brajeswar_go_user
// KoD4XjOZHUfTTifI
