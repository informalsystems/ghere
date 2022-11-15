package main

import (
	"context"
	"os"
)

func main() {
	rc := newRootCmd()
	if err := rc.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
