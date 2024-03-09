package main

import (
	"context"
	"log"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	psbotfunc "github.com/SotaEndo0214/pbbotfunc"
)

func main() {
	funcframework.RegisterHTTPFunctionContext(context.Background(), "/slack/events", psbotfunc.PokemonSleepFoods)
	port := "8080"
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
