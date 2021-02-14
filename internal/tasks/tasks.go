package tasks

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	_ "github.com/go-sql-driver/mysql" // Mysql Driver
)

// Version : The release version
var Version string

// Commit : The commit of this release
var Commit string

// BuildDate : The release build date
var BuildDate string

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

type taskSetupBootnodeArgs struct {
	BootnodeAddress string `json:"bootnode_address"`
	BootnodeToken   string `json:"bootnode_token"`
	AssignedAddress string `json:"assigned_address"`
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

	if Version == "dev" {
		fmt.Println("Dev environemnt. Not executing tasks.")
	} else {
		switch task.Task {
		case "setup_bootnode":

			fmt.Println("Setting up bootnode connection...")
			var args taskSetupBootnodeArgs
			err := json.Unmarshal([]byte(task.Args), &args)
			if err != nil {
				fmt.Println("Error reading arguments of setup_bootnode task: %s", err)
			} else {
				task.Result = sql.NullString{String: taskSetupBootnode(args), Valid: true}
			}

		case "start_edgeapp":
			fmt.Println("Starting EdgeApp...")
			task.Result = sql.NullString{String: taskStartEdgeApp(), Valid: true}
			// ...
		case "stop_edgeapp":
			fmt.Println("Stopping EdgeApp...")
			task.Result = sql.NullString{String: taskStopEdgeApp(), Valid: true}
			// ...
		}
	}

	if task.Result.Valid {
		db.Query("Update tasks SET status = 2, result = '" + task.Result.String + "' WHERE ID = " + strconv.Itoa(task.ID) + ";")
	} else {
		db.Query("Update tasks SET status = 2, result = 'Invalid Task' WHERE ID = " + strconv.Itoa(task.ID) + ";")
	}

	if err != nil {
		panic(err.Error())
	}

	returnTask := task

	return returnTask

}

// Internal functions, for each task

func taskSetupBootnode(args taskSetupBootnodeArgs) string {

	out, err := exec.Command("tinc-boot", "gen", "--token "+args.BootnodeToken, args.BootnodeAddress+":8655", "--prefix "+args.AssignedAddress).Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
	output := string(out[:])
	fmt.Println(output)

	out, err = exec.Command("systemctl", "start", "tinc@dnet").Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
	output = string(out[:])
	fmt.Println(output)

	out, err = exec.Command("systemctl", "enable", "tinc@dnet").Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
	output = string(out[:])
	fmt.Println(output)

	return output

}

func taskStartEdgeApp() string {

	return "OK"
}

func taskStopEdgeApp() string {

	return "OK"
}
