package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgebox-iot/edgeboxctl/internal/diagnostics"
	"github.com/edgebox-iot/edgeboxctl/internal/tasks"
	"github.com/edgebox-iot/edgeboxctl/internal/utils"
	"github.com/urfave/cli/v2" // imports as package "cli"
)

const defaultNotReadySleepTime time.Duration = time.Second * 60
const defaultSleepTime time.Duration = time.Second

func main() {

	app := &cli.App{
		Name:  "edgeboxctl",
		Usage: "A tool to facilitate hosting apps and securing your personal data",
		Action: func(c *cli.Context) error {
			startService() // Defaults to start as a service if no commands or flags are passed
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "bootstrap",
				Aliases: []string{"a"},
				Usage:   "Setps up initial structure and dependencies for the edgebox system",
				Action: func(c *cli.Context) error {
					fmt.Println("Edgebox Setup")
					return nil
				},
			},
			{
				Name:    "app",
				Aliases: []string{"t"},
				Usage:   "options for edgeapp management",
				Subcommands: []*cli.Command{
					{
						Name:  "start",
						Usage: "start the specified app",
						Action: func(c *cli.Context) error {
							task := tasks.ExecuteTask(tasks.GetBaseTask(
								"start_edgeapp",
								fmt.Sprintf("{\"id\": \"%s\"}", c.Args().First()),
							))

							return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
						},
					},
					{
						Name:  "stop",
						Usage: "stop the specified app",
						Action: func(c *cli.Context) error {

							task := tasks.ExecuteTask(tasks.GetBaseTask(
								"stop_edgeapp",
								fmt.Sprintf("{\"id\": \"%s\"}", c.Args().First()),
							))

							return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
						},
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func startService() {
	log.Printf("Starting edgeboxctl service")

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
			systemIterator(tick)
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
	_, err := os.Stat("/home/system/components/ws/.ready")
	return !os.IsNotExist(err)
}

// IsDatabaseReady : Checks is it can successfully connect to the task queue db
func isDatabaseReady() bool {
	return false
}

func systemIterator(tick int) {

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
