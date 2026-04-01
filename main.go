package main

import "github.com/devsternrassler/vidscribe/cmd"

// Set via goreleaser ldflags: -X main.version=... -X main.commit=... -X main.date=...
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersion(version, commit, date)
	cmd.Execute()
}
