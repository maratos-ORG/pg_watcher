package main

import (
	"context"
	"fmt"
	"os"

	"github.com/maratos-ORG/pg_watcher/internal/watcher"
)

// build is injected at build time via:
//
//	go build -ldflags "-X main.build=$(git describe --tags --dirty --always 2>/dev/null || echo dev)"
var build = "dev"

func main() {
	// Parse CLI flags (includes -version printing using `build`)
	fp, cp, err := watcher.ParseFlags(build)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Run the tool
	if err := watcher.Run(context.Background(), fp, cp); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
