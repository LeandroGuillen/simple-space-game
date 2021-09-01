package main

import (
	"corion/bin/api"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	myAPI := api.NewAPI()
	myAPI.Start()
}
