package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"time"
	//"syscall"
)

func main() {

	// load command line arguments

	name := flag.String("name", "edgebox", "name for the service")

	flag.Parse()

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
		AppCleanup()
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
func AppCleanup() {
	log.Println("Cleaning up app status before exit")
}
