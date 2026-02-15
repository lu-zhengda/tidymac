package main

import (
	"log"
	"os"

	"github.com/lu-zhengda/macbroom/internal/cli"
	"github.com/spf13/cobra/doc"
)

func main() {
	dir := "./docs/man"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatal(err)
	}
	header := &doc.GenManHeader{
		Title:   "MACBROOM",
		Section: "1",
	}
	if err := doc.GenManTree(cli.RootCmd(), header, dir); err != nil {
		log.Fatal(err)
	}
}
