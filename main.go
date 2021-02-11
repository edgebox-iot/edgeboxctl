package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"time"
	//"syscall"
	"github.com/edgebox-iot/sysctl/internal/diagnostics"

)

func main() {

	// load command line arguments

	version := flag.Bool("version", false, "Get the version info")
	name := flag.String("name", "edgebox", "name for the service")

	flag.Parse()

	if *version {
		printVersion()
		os.Exit(0)
	} 

	log.Printf("Starting Sysctl service for %s", *name)

	// setup signal catching
	sigs := make(chan os.Signal, 1)

	// catch all signals since not explicitly listing
	signal.Notify(sigs)

	// Cathing specific signals can be done with:
	//signal.Notify(sigs,syscall.SIGQUIT)

	// method invoked upon seeing signal
	go func() {
		s := <-sigs
		log.Printf("RECEIVED SIGNAL: %s", s)
		appCleanup()
		os.Exit(1)
	}()

	// infinite loop
	for {

		log.Printf("Executing instruction %s", *name)

		// wait random number of milliseconds
		Nsecs := 1000
		log.Printf("Next instruction executed in %dms", Nsecs)
		time.Sleep(time.Millisecond * time.Duration(Nsecs))
	}

}

// AppCleanup : cleanup app state before exit
func appCleanup() {
	log.Println("Cleaning up app status before exit")
}

func printVersion() {
	log.Printf(
		"version: %s\ncommit: %s\nbuild time: %s",
		diagnostics.Version, diagnostics.Commit, diagnostics.BuildDate,
	)
}
