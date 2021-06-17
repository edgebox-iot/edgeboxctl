package tasks

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/edgebox-iot/edgeboxctl/internal/diagnostics"
	"github.com/edgebox-iot/edgeboxctl/internal/edgeapps"
	"github.com/edgebox-iot/edgeboxctl/internal/storage"
	"github.com/edgebox-iot/edgeboxctl/internal/system"
	"github.com/edgebox-iot/edgeboxctl/internal/utils"
	_ "github.com/go-sql-driver/mysql" // Mysql Driver
	_ "github.com/mattn/go-sqlite3"    // SQlite Driver
)

// Task : Struct for Task type
type Task struct {
	ID      int            `json:"id"`
	Task    string         `json:"task"`
	Args    sql.NullString `json:"args"` // Database fields that can be null must use the sql.NullString type
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

type taskStartEdgeAppArgs struct {
	ID string `json:"id"`
}

type taskInstallEdgeAppArgs struct {
	ID string `json:"id"`
}

type taskRemoveEdgeAppArgs struct {
	ID string `json:"id"`
}

type taskStopEdgeAppArgs struct {
	ID string `json:"id"`
}

type taskEnableOnlineArgs struct {
	ID          string `json:"id"`
	InternetURL string `json:"internet_url"`
}

type taskDisableOnlineArgs struct {
	ID string `json:"id"`
}

type taskEnablePublicDashboardArgs struct {
	InternetURL string `json:"internet_url`
}

// GetNextTask : Performs a MySQL query over the device's Edgebox API
func GetNextTask() Task {

	// Will try to connect to API database, which should be running locally under WS.
	db, err := sql.Open("sqlite3", utils.GetSQLiteDbConnectionDetails())

	// if there is an error opening the connection, handle it
	if err != nil {
		panic(err.Error())
	}

	results, err := db.Query("SELECT id, task, args, status, result, created, updated FROM task WHERE status = 0 ORDER BY created ASC LIMIT 1;")

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

	results.Close()
	db.Close()

	return task

}

// ExecuteTask : Performs execution of the given task, updating the task status as it goes, and publishing the task result
func ExecuteTask(task Task) Task {

	db, err := sql.Open("sqlite3", utils.GetSQLiteDbConnectionDetails())

	if err != nil {
		panic(err.Error())
	}

	statement, err := db.Prepare("UPDATE task SET status = ?, updated = ? WHERE ID = ?;") // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	formatedDatetime := utils.GetSQLiteFormattedDateTime(time.Now())

	_, err = statement.Exec(1, formatedDatetime, strconv.Itoa(task.ID)) // Execute SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	if diagnostics.Version == "dev" {
		log.Printf("Dev environemnt. Not executing tasks.")
	} else {
		log.Println("Task: " + task.Task)
		switch task.Task {
		case "setup_tunnel":

			log.Println("Setting up bootnode connection...")
			var args taskSetupTunnelArgs
			err := json.Unmarshal([]byte(task.Args.String), &args)
			if err != nil {
				log.Printf("Error reading arguments of setup_bootnode task: %s", err)
			} else {
				taskResult := taskSetupTunnel(args)
				task.Result = sql.NullString{String: taskResult, Valid: true}
			}

		case "install_edgeapp":

			log.Println("Installing EdgeApp...")
			var args taskInstallEdgeAppArgs
			err := json.Unmarshal([]byte(task.Args.String), &args)
			if err != nil {
				log.Printf("Error reading arguments of install_edgeapp task: %s", err)
			} else {
				taskResult := taskInstallEdgeApp(args)
				task.Result = sql.NullString{String: taskResult, Valid: true}
			}

		case "remove_edgeapp":

			log.Println("Removing EdgeApp...")
			var args taskRemoveEdgeAppArgs
			err := json.Unmarshal([]byte(task.Args.String), &args)
			if err != nil {
				log.Printf("Error reading arguments of remove_edgeapp task: %s", err)
			} else {
				taskResult := taskRemoveEdgeApp(args)
				task.Result = sql.NullString{String: taskResult, Valid: true}
			}

		case "start_edgeapp":

			log.Println("Starting EdgeApp...")
			var args taskStartEdgeAppArgs
			err := json.Unmarshal([]byte(task.Args.String), &args)
			if err != nil {
				log.Printf("Error reading arguments of start_edgeapp task: %s", err)
			} else {
				taskResult := taskStartEdgeApp(args)
				task.Result = sql.NullString{String: taskResult, Valid: true}
			}

		case "stop_edgeapp":

			log.Println("Stopping EdgeApp...")
			var args taskStopEdgeAppArgs
			err := json.Unmarshal([]byte(task.Args.String), &args)
			if err != nil {
				log.Printf("Error reading arguments of stop_edgeapp task: %s", err)
			} else {
				taskResult := taskStopEdgeApp(args)
				task.Result = sql.NullString{String: taskResult, Valid: true}
			}

		case "enable_online":

			log.Println("Enabling online access to EdgeApp...")
			var args taskEnableOnlineArgs
			err := json.Unmarshal([]byte(task.Args.String), &args)
			if err != nil {
				log.Printf("Error reading arguments of enable_online task: %s", err)
			} else {
				taskResult := taskEnableOnline(args)
				task.Result = sql.NullString{String: taskResult, Valid: true}
			}

		case "disable_online":

			log.Println("Disabling online access to EdgeApp...")
			var args taskDisableOnlineArgs
			err := json.Unmarshal([]byte(task.Args.String), &args)
			if err != nil {
				log.Printf("Error reading arguments of enable_online task: %s", err)
			} else {
				taskResult := taskDisableOnline(args)
				task.Result = sql.NullString{String: taskResult, Valid: true}
			}

		case "enable_public_dashboard":

			log.Println("Enabling online access to Dashboard...")
			var args taskEnablePublicDashboardArgs
			err := json.Unmarshal([]byte(task.Args.String), &args)
			if err != nil {
				log.Printf("Error reading arguments of enable_public_dashboard task: %s", err)
			} else {
				taskResult := taskEnablePublicDashboard(args)
				task.Result = sql.NullString{String: taskResult, Valid: true}
			}

		case "disable_public_dashboard":

			log.Println("Disabling online access to Dashboard...")
			taskResult := taskDisablePublicDashboard()
			task.Result = sql.NullString{String: taskResult, Valid: true}

		}

	}

	statement, err = db.Prepare("Update task SET status = ?, result = ?, updated = ? WHERE ID = ?;") // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}
	if err != nil {
		log.Fatal(err.Error())
	}

	formatedDatetime = utils.GetSQLiteFormattedDateTime(time.Now())

	if task.Result.Valid {
		_, err = statement.Exec(2, task.Result.String, formatedDatetime, strconv.Itoa(task.ID)) // Execute SQL Statement with result info
		if err != nil {
			log.Fatal(err.Error())
		}
	} else {
		_, err = statement.Exec(3, "Error", formatedDatetime, strconv.Itoa(task.ID)) // Execute SQL Statement with Error info
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	if err != nil {
		panic(err.Error())
	}

	db.Close()

	returnTask := task

	return returnTask

}

// ExecuteSchedules - Run Specific tasks without input each multiple x of ticks.
func ExecuteSchedules(tick int) {

	if tick == 1 {

		ip := taskGetSystemIP()
		log.Println("System IP is: " + ip)

		release := taskSetReleaseVersion()
		log.Println("Setting api option flag for Edgeboxctl (" + release + " version)")

		// Executing on startup (first tick). Schedules run before tasks in the SystemIterator
		uptime := taskGetSystemUptime()
		log.Println("Uptime is " + uptime + " seconds (" + system.GetUptimeFormatted() + ")")

		log.Println(taskGetStorageDevices())
		log.Println(taskGetEdgeApps())

	}

	if tick%5 == 0 {
		// Executing every 5 ticks
		taskGetSystemUptime()
		log.Println(taskGetStorageDevices())
	}

	if tick%30 == 0 {
		// Executing every 30 ticks
		log.Println(taskGetEdgeApps())
	}

	if tick%60 == 0 {
		ip := taskGetSystemIP()
		log.Println("System IP is: " + ip)
	}

	// Just add a schedule here if you need a custom one (every "tick hour", every "tick day", etc...)

}

func taskSetupTunnel(args taskSetupTunnelArgs) string {

	fmt.Println("Executing taskSetupTunnel")

	cmdargs := []string{"gen", "--name", args.NodeName, "--token", args.BootnodeToken, args.BootnodeAddress + ":8655", "--prefix", args.AssignedAddress}
	utils.Exec(utils.GetPath("wsPath"), "tinc-boot", cmdargs)

	cmdargs = []string{"start", "tinc@dnet"}
	utils.Exec(utils.GetPath("wsPath"), "systemctl", cmdargs)

	cmdargs = []string{"enable", "tinc@dnet"}
	utils.Exec(utils.GetPath("wsPath"), "systemctl", cmdargs)

	output := "OK" // Better check / logging of command execution result.
	return output

}

func taskInstallEdgeApp(args taskInstallEdgeAppArgs) string {

	fmt.Println("Executing taskInstallEdgeApp for " + args.ID)

	result := edgeapps.SetEdgeAppInstalled(args.ID)

	resultJSON, _ := json.Marshal(result)

	taskGetEdgeApps()

	return string(resultJSON)

}

func taskRemoveEdgeApp(args taskRemoveEdgeAppArgs) string {

	fmt.Println("Executing taskRemoveEdgeApp for " + args.ID)

	// Making sure the application is stopped before setting it as removed.
	edgeapps.StopEdgeApp(args.ID)
	result := edgeapps.SetEdgeAppNotInstalled(args.ID)

	resultJSON, _ := json.Marshal(result)

	taskGetEdgeApps()

	return string(resultJSON)

}

func taskStartEdgeApp(args taskStartEdgeAppArgs) string {

	fmt.Println("Executing taskStartEdgeApp for " + args.ID)

	result := edgeapps.RunEdgeApp(args.ID)

	resultJSON, _ := json.Marshal(result)

	taskGetEdgeApps() // This task will imediatelly update the entry in the api database.

	return string(resultJSON)

}

func taskStopEdgeApp(args taskStopEdgeAppArgs) string {

	fmt.Println("Executing taskStopEdgeApp for " + args.ID)

	result := edgeapps.StopEdgeApp(args.ID)

	resultJSON, _ := json.Marshal(result)

	taskGetEdgeApps() // This task will imediatelly update the entry in the api database.

	return string(resultJSON)

}

func taskEnableOnline(args taskEnableOnlineArgs) string {

	fmt.Println("Executing taskEnableOnline for " + args.ID)

	result := edgeapps.EnableOnline(args.ID, args.InternetURL)

	resultJSON, _ := json.Marshal(result)

	taskGetEdgeApps()

	return string(resultJSON)

}

func taskDisableOnline(args taskDisableOnlineArgs) string {

	fmt.Println("Executing taskDisableOnline for " + args.ID)

	result := edgeapps.DisableOnline(args.ID)

	resultJSON, _ := json.Marshal(result)

	taskGetEdgeApps()

	return string(resultJSON)

}

func taskEnablePublicDashboard(args taskEnablePublicDashboardArgs) string {

	fmt.Println("Enabling taskEnablePublicDashboard")
	result := edgeapps.EnablePublicDashboard(args.InternetURL)
	if result {
		return "{result: true}"

	}

	return "{result: false}"

}

func taskDisablePublicDashboard() string {
	
	fmt.Println("Executing taskDisablePublicDashboard")
	result := edgeapps.DisablePublicDashboard()
	if result {
		return "{result: true}"
	}
	return "{result: false}"

}

func taskSetReleaseVersion() string {

	fmt.Println("Executing taskSetReleaseVersion")

	db, err := sql.Open("sqlite3", utils.GetSQLiteDbConnectionDetails())

	if err != nil {
		log.Fatal(err.Error())
	}

	statement, err := db.Prepare("REPLACE into option (name, value, created, updated) VALUES (?, ?, ?, ?);") // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	formatedDatetime := utils.GetSQLiteFormattedDateTime(time.Now())

	_, err = statement.Exec("RELEASE_VERSION", diagnostics.Version, formatedDatetime, formatedDatetime) // Execute SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	db.Close()

	return diagnostics.Version
}

func taskGetEdgeApps() string {

	fmt.Println("Executing taskGetEdgeApps")

	edgeApps := edgeapps.GetEdgeApps()
	edgeAppsJSON, _ := json.Marshal(edgeApps)

	db, err := sql.Open("sqlite3", utils.GetSQLiteDbConnectionDetails())

	if err != nil {
		log.Fatal(err.Error())
	}

	statement, err := db.Prepare("REPLACE into option (name, value, created, updated) VALUES (?, ?, ?, ?);") // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	formatedDatetime := utils.GetSQLiteFormattedDateTime(time.Now())

	_, err = statement.Exec("EDGEAPPS_LIST", string(edgeAppsJSON), formatedDatetime, formatedDatetime) // Execute SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	db.Close()

	return string(edgeAppsJSON)

}

func taskGetSystemUptime() string {
	fmt.Println("Executing taskGetSystemUptime")

	uptime := system.GetUptimeInSeconds()

	db, err := sql.Open("sqlite3", utils.GetSQLiteDbConnectionDetails())

	if err != nil {
		log.Fatal(err.Error())
	}

	statement, err := db.Prepare("REPLACE into option (name, value, created, updated) VALUES (?, ?, ?, ?);") // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	formatedDatetime := utils.GetSQLiteFormattedDateTime(time.Now())

	_, err = statement.Exec("SYSTEM_UPTIME", uptime, formatedDatetime, formatedDatetime) // Execute SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	db.Close()

	return uptime

}

func taskGetStorageDevices() string {
	fmt.Println("Executing taskGetStorageDevices")

	devices := storage.GetDevices(diagnostics.Version)
	devicesJSON, _ := json.Marshal(devices)

	db, err := sql.Open("sqlite3", utils.GetSQLiteDbConnectionDetails())

	if err != nil {
		log.Fatal(err.Error())
	}

	statement, err := db.Prepare("REPLACE into option (name, value, created, updated) VALUES (?, ?, ?, ?);") // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	formatedDatetime := utils.GetSQLiteFormattedDateTime(time.Now())

	_, err = statement.Exec("STORAGE_DEVICES_LIST", devicesJSON, formatedDatetime, formatedDatetime) // Execute SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	db.Close()

	return string(devicesJSON)

}

func taskGetSystemIP() string {
	fmt.Println("Executing taskGetStorageDevices")

	ip := system.GetIP()

	db, err := sql.Open("sqlite3", utils.GetSQLiteDbConnectionDetails())

	if err != nil {
		log.Fatal(err.Error())
	}

	statement, err := db.Prepare("REPLACE into option (name, value, created, updated) VALUES (?, ?, ?, ?);") // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	formatedDatetime := utils.GetSQLiteFormattedDateTime(time.Now())

	_, err = statement.Exec("IP_ADDRESS", ip, formatedDatetime, formatedDatetime) // Execute SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	db.Close()

	return ip
}
