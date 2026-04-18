package main

import (
	"flag"
	"fmt"
)

// Version is a placeholder that will be overwritten at build time.
var Version = "development"

func main() {
	// Define the --version flag
	versionFlag := flag.Bool("version", false, "print the app version")

	// Parse flags
	flag.CommandLine.Usage = func() {
		fmt.Println("Usage: nocti [options]")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Logic for the version flag
	if *versionFlag {
		fmt.Println(Version)
		return
	}

	// The app does nothing else and exits quietly if run alone.
}
