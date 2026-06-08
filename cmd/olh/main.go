package main

import (
	"context"
	"fmt"
	"os"

	"github.com/formation-res/open-location-hub-cli/internal/app"
)

func main() {
	ctx := context.Background()
	if err := app.NewRootCommand().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
