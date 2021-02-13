package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgebox-iot/sysctl/internal/diagnostics"
	"github.com/edgebox-iot/sysctl/internal/tasks"
)

func main() {

	// load command line arguments

	version := flag.Bool("version", false, "Get the version info")
	db := flag.Bool("database", false, "Get database connection info")
	name := flag.String("name", "edgebox", "Name for the service")

	flag.Parse()

	if *version {
		printVersion()
		os.Exit(0)
	}

	if *db {
		printDbDetails()
		os.Exit(0)
	}

	log.Printf("Starting Sysctl service for %s", *name)

	// setup signal catching
	sigs := make(chan os.Signal, 1)

	// catch all signals since not explicitly listing
	signal.Notify(sigs, syscall.SIGQUIT)

	// Cathing specific signals can be done with:
	//signal.Notify(sigs,syscall.SIGQUIT)

	// method invoked upon seeing signal
	go func() {
		s := <-sigs
		log.Printf("RECEIVED SIGNAL: %s", s)
		appCleanup()
		os.Exit(1)
	}()

	printVersion()

	printDbDetails()

	dbQueryResult := tasks.PerformQuery()
	log.Printf("Query result: %s", dbQueryResult)

	// infinite loop
	for {

		systemIterator(name)

	}

}

// AppCleanup : cleanup app state before exit
func appCleanup() {
	log.Println("Cleaning up app status before exit")
}

func printVersion() {
	fmt.Printf(
		"\nversion: %s\ncommit: %s\nbuild time: %s\n",
		diagnostics.Version, diagnostics.Commit, diagnostics.BuildDate,
	)
}

func printDbDetails() {
	fmt.Printf(
		"\n\nDatabase Connection Information:\nHost: %s\nuser: %s\npassword: %s\n\n",
		tasks.Dbhost, tasks.Dbuser, tasks.Dbpass,
	)
}

// IsSystemReady : Checks hability of the service to execute commands (Only after "edgebox --build" is ran at least once via SSH, or if built for distribution)
func isSystemReady() bool {
	_, err := os.Stat("/home/system/components/ws")
	return !os.IsNotExist(err)
}

// IsDatabaseReady : Checks is it can successfully connect to the task queue db
func isDatabaseReady() bool {
	return false
}

// getNextInstruction : Retrieves next instruction from the database
func getNextInstruction(name *string) string {
	log.Printf("Fetching next instruction for %s", *name)
	return "Test Instruction Command"
}

func systemIterator(name *string) {

	if !isSystemReady() {
		// Wait about 60 seconds before trying again.
		log.Printf("System not ready. Next try will be executed in 60 seconds")
		time.Sleep(time.Millisecond * time.Duration(60000))
	} else {
		// Wait about 1 second before resumming operations.
		log.Printf("Next instruction will be executed 1 second")
		time.Sleep(time.Millisecond * time.Duration(1000))
		log.Printf("Next instruction: %s", getNextInstruction(name))
	}

}
