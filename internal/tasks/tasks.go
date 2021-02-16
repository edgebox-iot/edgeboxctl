package tasks

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/edgebox-iot/sysctl/internal/diagnostics"
	"github.com/edgebox-iot/sysctl/internal/edgeapps"
	"github.com/edgebox-iot/sysctl/internal/utils"
	_ "github.com/go-sql-driver/mysql" // Mysql Driver
)

// Dbhost : Database host (can be tweaked in makefile)
var Dbhost string

// Dbname : Database name (can be tweaked in makefile)
var Dbname string

// Dbuser : Database user (can be tweaked in makefile)
var Dbuser string

// Dbpass : Database password (can be tweaked in)
var Dbpass string

// Task : Struct for Task type
type Task struct {
	ID      int            `json:"id"`
	Task    string         `json:"task"`
	Args    string         `json:"args"`
	Status  string         `json:"status"`
	Result  sql.NullString `json:"result"` // Database fields that can be null must use the sql.NullString type
	Created string         `json:"created"`
	Updated string         `json:"updated"`
}

type taskSetupTunnelArgs struct {
	BootnodeAddress string `json:"bootnode_address"`
	BootnodeToken   string `json:"bootnode_token"`
	AssignedAddress string `json:"assigned_address"`
	NodeName        string `json:"node_name"`
}

// GetNextTask : Performs a MySQL query over the device's Edgebox API
func GetNextTask() Task {

	// Will try to connect to API database, which should be running locally under WS.
	db, err := sql.Open("mysql", Dbuser+":"+Dbpass+"@tcp("+Dbhost+")/"+Dbname)

	// if there is an error opening the connection, handle it
	if err != nil {
		panic(err.Error())
	}

	// defer the close till after the main function has finished executing
	defer db.Close()

	// perform a db.Query insert
	results, err := db.Query("SELECT * FROM tasks WHERE status = 0 ORDER BY created ASC LIMIT 1;")

	// if there is an error inserting, handle it
	if err != nil {
		panic(err.Error())
	}

	var task Task

	for results.Next() {

		// for each row, scan the result into our task composite object
		err = results.Scan(&task.ID, &task.Task, &task.Args, &task.Status, &task.Result, &task.Created, &task.Updated)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}
	}

	// be careful deferring Queries if you are using transactions
	defer results.Close()

	return task

}

// ExecuteTask : Performs execution of the given task, updating the task status as it goes, and publishing the task result
func ExecuteTask(task Task) Task {

	db, err := sql.Open("mysql", Dbuser+":"+Dbpass+"@tcp("+Dbhost+")/"+Dbname)

	if err != nil {
		panic(err.Error())
	}

	defer db.Close()

	statusUpdate, err := db.Query("UPDATE tasks SET status = 1 WHERE ID = " + strconv.Itoa(task.ID))

	if err != nil {
		panic(err.Error())
	}

	for statusUpdate.Next() {

	}

	if diagnostics.Version == "dev" {
		log.Printf("Dev environemnt. Not executing tasks.")
	} else {
		log.Println("Task: " + task.Task)
		switch task.Task {
		case "setup_tunnel":

			log.Println("Setting up bootnode connection...")
			var args taskSetupTunnelArgs
			err := json.Unmarshal([]byte(task.Args), &args)
			if err != nil {
				log.Printf("Error reading arguments of setup_bootnode task: %s", err)
			} else {
				taskResult := taskSetupTunnel(args)
				task.Result = sql.NullString{String: taskResult, Valid: true}
			}

		case "start_edgeapp":
			log.Printf("Starting EdgeApp...")
			task.Result = sql.NullString{String: taskStartEdgeApp(), Valid: true}
			// ...
		case "stop_edgeapp":
			log.Printf("Stopping EdgeApp...")
			task.Result = sql.NullString{String: taskStopEdgeApp(), Valid: true}
			// ...
		}
	}

	if task.Result.Valid {
		db.Query("Update tasks SET status = 2, result = '" + task.Result.String + "' WHERE ID = " + strconv.Itoa(task.ID) + ";")
	} else {
		db.Query("Update tasks SET status = 3, result = 'Error' WHERE ID = " + strconv.Itoa(task.ID) + ";")
	}

	if err != nil {
		panic(err.Error())
	}

	returnTask := task

	return returnTask

}

// ExecuteSchedules - Run Specific tasks without input each multiple x of ticks.
func ExecuteSchedules(tick int) {

	if tick == 1 {
		// Executing on startup (first tick). Schedules run before tasks in the SystemIterator
		taskGetEdgeApps()
	}

	if tick%30 == 0 {
		// Executing every 30 ticks
		taskGetEdgeApps()

	}

	if tick%60 == 0 {
		// Every 60 ticks...

	}

	// Just add a schedule here if you need a custom one (every "tick hour", every "tick day", etc...)

}

func taskSetupTunnel(args taskSetupTunnelArgs) string {

	fmt.Println("Executing taskSetupTunnel")

	cmdargs := []string{"gen", "--name", args.NodeName, "--token", args.BootnodeToken, args.BootnodeAddress + ":8655", "--prefix", args.AssignedAddress}
	utils.Exec("tinc-boot", cmdargs)

	cmdargs = []string{"start", "tinc@dnet"}
	utils.Exec("systemctl", cmdargs)

	cmdargs = []string{"enable", "tinc@dnet"}
	utils.Exec("systemctl", cmdargs)

	output := "OK" // Better check / logging of command execution result.
	return output

}

func taskGetEdgeApps() string {

	fmt.Println("Executing taskGetEdgeApps")
	edgeapps.GetEdgeApps()

	// Saving information in the "options" table.
	return "OK"
}

func taskStartEdgeApp() string {

	return "OK"
}

func taskStopEdgeApp() string {

	return "OK"
}
