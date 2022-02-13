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
	"github.com/mitchellh/colorstring"
	"github.com/urfave/cli/v2" // imports as package "cli",
)

const defaultNotReadySleepTime time.Duration = time.Second * 60
const defaultSleepTime time.Duration = time.Second

var errorMissingApplicationSlug = colorstring.Color("[red]Error: [white]Missing application slug")
var errorUnexpected = colorstring.Color("[red]An unexpected error ocurring and the application crashed")
var notYetImplemented = colorstring.Color("[yellow]This feature is not yet implemented, this command is only a stub.")

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
				Aliases: []string{"b"},
				Usage:   "Sets up initial structure and dependencies for the edgebox system",
				Action: func(c *cli.Context) error {
					fmt.Println("Edgebox Setup")
					return nil
				},
			},
			{
				Name:    "tunnel",
				Aliases: []string{"t"},
				Usage:   "Edgebox tunnel settings",
				Subcommands: []*cli.Command{
					{
						Name:    "setup",
						Aliases: []string{"s"},
						Usage:   "Sets up an encrypted secure connection to a tunnel",
						Action: func(c *cli.Context) error {

							argumentError := checkArgumentsPresence(c, 4)
							if argumentError != nil {
								return argumentError
							}

							task := getCommandTask(
								"setup_tunnel",
								fmt.Sprintf(
									"{\"bootnode_address\": \"%s\", \"token\": \"%s\", \"assigned_address\": \"%s\", \"node_name\": \"%s\"}",
									c.Args().Get(0),
									c.Args().Get(1),
									c.Args().Get(2),
									c.Args().Get(3),
								),
								true,
							)

							return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
						},
					},
					{
						Name:    "enable",
						Aliases: []string{"e"},
						Usage:   "enables the public tunnel (when configured)",
						Action: func(c *cli.Context) error {
							return cli.Exit(notYetImplemented, 1)
						},
					},
					{
						Name:    "disable",
						Aliases: []string{"d"},
						Usage:   "disables the public tunnel (when online)",
						Action: func(c *cli.Context) error {
							return cli.Exit(notYetImplemented, 1)
						},
					},
				},
			},
			{
				Name:    "app",
				Aliases: []string{"a"},
				Usage:   "options for edgeapp management",
				Subcommands: []*cli.Command{
					{
						Name:    "list",
						Aliases: []string{"l"},
						Usage:   "list currently installed apps and their status",
						Action: func(c *cli.Context) error {
							task := getCommandTask("list_edgeapps", "", true)
							// return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
							return cli.Exit(task.Result.String, 0)
						},
					},
					{
						Name:    "install",
						Aliases: []string{"i"},
						Usage:   "install the specified app (slug or package file)",
						Action: func(c *cli.Context) error {

							argumentError := checkArgumentsPresence(c, 1)
							if argumentError != nil {
								return argumentError
							}

							task := getCommandTask("install_edgeapp", fmt.Sprintf("{\"id\": \"%s\"}", c.Args().First()), true)

							return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
						},
					},
					{
						Name:    "remove",
						Aliases: []string{"r"},
						Usage:   "remove the specified app",
						Action: func(c *cli.Context) error {

							argumentError := checkArgumentsPresence(c, 1)
							if argumentError != nil {
								return argumentError
							}

							task := getCommandTask("remove_edgeapp", fmt.Sprintf("{\"id\": \"%s\"}", c.Args().First()), true)

							return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
						},
					},
					{
						Name:    "start",
						Aliases: []string{"s"},
						Usage:   "start the specified app",
						Action: func(c *cli.Context) error {

							argumentError := checkArgumentsPresence(c, 1)
							if argumentError != nil {
								return argumentError
							}

							task := getCommandTask("start_edgeapp", fmt.Sprintf("{\"id\": \"%s\"}", c.Args().First()), true)

							return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
						},
					},
					{
						Name:    "stop",
						Aliases: []string{"k"},
						Usage:   "stop the specified app",
						Action: func(c *cli.Context) error {

							argumentError := checkArgumentsPresence(c, 1)
							if argumentError != nil {
								return argumentError
							}

							task := getCommandTask("stop_edgeapp", fmt.Sprintf("{\"id\": \"%s\"}", c.Args().First()), true)

							return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
						},
					},
					{
						Name:    "online",
						Aliases: []string{"o"},
						Usage:   "set an app status for online access",
						Subcommands: []*cli.Command{
							{
								Name:    "enable",
								Aliases: []string{"e"},
								Usage:   "set an app as accessible online",
								Action: func(c *cli.Context) error {

									argumentError := checkArgumentsPresence(c, 2)
									if argumentError != nil {
										return argumentError
									}

									task := getCommandTask(
										"enable_online",
										fmt.Sprintf(
											"{\"id\": \"%s\", \"internet_url\": \"%s\"}",
											c.Args().Get(0),
											c.Args().Get(1),
										),
										true,
									)

									return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
								},
							},
							{
								Name:    "disable",
								Aliases: []string{"d"},
								Usage:   "set an app as local-network private",
								Action: func(c *cli.Context) error {

									argumentError := checkArgumentsPresence(c, 1)
									if argumentError != nil {
										return argumentError
									}

									task := getCommandTask("disable_online", fmt.Sprintf("{\"id\": \"%s\"}", c.Args().First()), true)

									return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
								},
							},
						},
					},
				},
			},
			{
				Name:    "dashboard",
				Aliases: []string{"d"},
				Usage:   "set dashboard access",
				Subcommands: []*cli.Command{
					{
						Name:    "enable",
						Aliases: []string{"e"},
						Usage:   "enable dashboard access",
						Action: func(c *cli.Context) error {
							argumentError := checkArgumentsPresence(c, 1)
							if argumentError != nil {
								return argumentError
							}

							task := getCommandTask("enable_public_dashboard", fmt.Sprintf("{\"internet_url\": \"%s\"}", c.Args().First()), true)
							return cli.Exit(utils.ColorJsonString(task.Result.String), 0)
						},
					},
					{
						Name:    "disable",
						Aliases: []string{"d"},
						Usage:   "disable dashboard access",
						Action: func(c *cli.Context) error {

							argumentError := checkArgumentsPresence(c, 1)
							if argumentError != nil {
								return argumentError
							}

							task := getCommandTask("disable_public_dashboard", "", true)
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

func checkArgumentsPresence(c *cli.Context, count int) error {
	if count != 0 {
		for i := 0; i < count; i++ {
			if c.Args().Get(i) == "" {
				return cli.Exit(errorMissingApplicationSlug, 1)
			}
		}
	}
	return nil
}

func getCommandTask(taskSlug string, taskArgs string, execute bool) tasks.Task {
	task := tasks.GetBaseTask(
		taskSlug,
		taskArgs,
	)

	if execute {
		task = tasks.ExecuteTask(task)
	}

	return task
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
