package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgebox-iot/edgeboxctl/internal/diagnostics"
	"github.com/edgebox-iot/edgeboxctl/internal/tasks"
	"github.com/edgebox-iot/edgeboxctl/internal/utils"
)

const defaultNotReadySleepTime time.Duration = time.Second * 60
const defaultSleepTime time.Duration = time.Second

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

	log.Printf("Starting edgeboxctl service for %s", *name)

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

	tick := 0

	// infinite loop
	for {

		if isSystemReady() {
			tick++ // Tick is an int, so eventually will "go out of ticks?" Maybe we want to reset the ticks every once in a while, to avoid working with big numbers...
			systemIterator(name, tick)
		} else {
			// Wait about 60 seconds before trying again.
			log.Printf("System not ready. Next try will be executed in 60 seconds")
			time.Sleep(defaultNotReadySleepTime)
		}

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
		"\n\nSQLite Database Location:\n %s\n\n",
		utils.GetSQLiteDbConnectionDetails(),
	)
}

// IsSystemReady : Checks hability of the service to execute commands (Only after "edgebox --build" is ran at least once via SSH, or if built for distribution)
func isSystemReady() bool {
	_, err := os.Stat(utils.GetPath(utils.WsPath) + ".ready")
	return !os.IsNotExist(err)
}

// IsDatabaseReady : Checks is it can successfully connect to the task queue db
func isDatabaseReady() bool {
	return false
}

func systemIterator(name *string, tick int) {

	log.Printf("Tick is %d", tick)

	tasks.ExecuteSchedules(tick)
	nextTask := tasks.GetNextTask()
	if nextTask.Task != "" {
		taskArguments := "No arguments"
		if nextTask.Args.Valid {
			taskArguments = nextTask.Args.String
		}
		log.Printf("Executing task %s / Args: %s", nextTask.Task, taskArguments)
		tasks.ExecuteTask(nextTask)
	} else {
		log.Printf("No tasks to execute.")
	}

	// Wait about 1 second before resumming operations.
	log.Printf("Next instruction will be executed 1 second")
	time.Sleep(defaultSleepTime)

}
