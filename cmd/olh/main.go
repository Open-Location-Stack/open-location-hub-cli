package main

import (
	"context"
	"os"

	"github.com/formation-res/open-location-hub-cli/internal/app"
)

func main() {
	ctx := context.Background()
	if err := app.NewRootCommand().ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
