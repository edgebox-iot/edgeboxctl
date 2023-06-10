package tasks

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"
	"os/exec"
	"strings"
	"os"
	"bufio"

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
	DomainName string `json:"domain_name"`
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
	InternetURL string `json:"internet_url"`
}

type taskSetupBackupsArgs struct {
	Service string `json:"service"`
	AccessKeyID string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	RepositoryName string `json:"repository_name"`
	RepositoryPassword string `json:"repository_password"`
}


const STATUS_CREATED int = 0
const STATUS_EXECUTING int = 1
const STATUS_FINISHED int = 2
const STATUS_ERROR int = 3

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

	fmt.Println("Changing task status to executing: " + task.Task)
	_, err = statement.Exec(STATUS_EXECUTING, formatedDatetime, strconv.Itoa(task.ID)) // Execute SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	if diagnostics.GetReleaseVersion() == diagnostics.DEV_VERSION {
		log.Printf("Dev environemnt. Not executing tasks.")
	} else {
		log.Println("Task: " + task.Task)
		log.Println("Args: " + task.Args.String)
		switch task.Task {
		case "setup_backups":

			log.Println("Setting up Backups Destination...")
			var args taskSetupBackupsArgs
			err := json.Unmarshal([]byte(task.Args.String), &args)
			if err != nil {
				log.Println("Error reading arguments of setup_backups task.")
			} else {
				taskResult := taskSetupBackups(args)
				taskResultBool := true
				// Check if returned taskResult string contains "error"
				if strings.Contains(taskResult, "error") {
					taskResultBool = false
				}
				task.Result = sql.NullString{String: taskResult, Valid: taskResultBool}
			}

		case "start_backup":

			log.Println("Backing up Edgebox...")
			taskResult := taskBackup()
			taskResultBool := true
			// Check if returned taskResult string contains "error"
			if strings.Contains(taskResult, "error") {
				taskResultBool = false
			}
			task.Result = sql.NullString{String: taskResult, Valid: taskResultBool}

		case "restore_backup":
			log.Println("Attempting to Restore Last Backup to Edgebox")
			taskResult := taskRestoreBackup()
			taskResultBool := true
			// Check if returned taskResult string contains "error"
			if strings.Contains(taskResult, "error") {
				taskResultBool = false
			}
			task.Result = sql.NullString{String: taskResult, Valid: taskResultBool}

		case "setup_tunnel":

			log.Println("Setting up Cloudflare Tunnel...")
			var args taskSetupTunnelArgs
			err := json.Unmarshal([]byte(task.Args.String), &args)
			if err != nil {
				log.Printf("Error reading arguments of setup_tunnel task: %s", err)
				status := "{\"status\": \"error\", \"message\": \"The Domain Name you are going to Authorize must be provided beforehand! Please insert a domain name and try again.\"}"
				utils.WriteOption("TUNNEL_STATUS", status)
			} else {
				taskResult := taskSetupTunnel(args)
				task.Result = sql.NullString{String: taskResult, Valid: true}
			}

		case "start_tunnel":

			log.Println("Starting Cloudflare Tunnel...")
			taskResult := taskStartTunnel()
			task.Result = sql.NullString{String: taskResult, Valid: true}

		case "stop_tunnel":

			log.Println("Stopping Cloudflare Tunnel...")
			taskResult := taskStopTunnel()
			task.Result = sql.NullString{String: taskResult, Valid: true}
		
		case "disable_tunnel":

			log.Println("Disabling Cloudflare Tunnel...")
			taskResult := taskDisableTunnel()
			task.Result = sql.NullString{String: taskResult, Valid: true}

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
		fmt.Println("Task Result: " + task.Result.String)
		_, err = statement.Exec(STATUS_FINISHED, task.Result.String, formatedDatetime, strconv.Itoa(task.ID)) // Execute SQL Statement with result info
		if err != nil {
			log.Fatal(err.Error())
		}
	} else {
		fmt.Println("Error executing task with result: " + task.Result.String)
		_, err = statement.Exec(STATUS_ERROR, "Error", formatedDatetime, strconv.Itoa(task.ID)) // Execute SQL Statement with Error info
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

		hostname := taskGetHostname()
		log.Println("Hostname is " + hostname)

		// if diagnostics.Version == "cloud" && !edgeapps.IsPublicDashboard() {
		// 	taskEnablePublicDashboard(taskEnablePublicDashboardArgs{
		// 		InternetURL: hostname + ".myedge.app",
		// 	})
		// }

		if diagnostics.GetReleaseVersion() == diagnostics.CLOUD_VERSION {
			log.Println("Setting up cloud version options (name, email, api token)")
			taskSetupCloudOptions()
		}

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
		// RESET SOME VARIABLES HERE IF NEEDED, SINCE SYSTEM IS UNBLOCKED
		utils.WriteOption("BACKUP_IS_WORKING", "0")

		// Check is Last Backup time (in unix time) is older than 1 h
		lastBackup := utils.ReadOption("BACKUP_LAST_RUN")
		if lastBackup != "" {
			lastBackupTime, err := strconv.ParseInt(lastBackup, 10, 64)
			if err != nil {
				log.Println("Error parsing last backup time: " + err.Error())
			} else {
				secondsSinceLastBackup := time.Now().Unix() - lastBackupTime
				if secondsSinceLastBackup > 3600 {
					// If last backup is older than 1 hour, set BACKUP_IS_WORKING to 0
					log.Println("Last backup was older than 1 hour, performing auto backup...")
					log.Println(taskAutoBackup())
				} else {

					log.Println("Last backup is " + fmt.Sprint(secondsSinceLastBackup) + " seconds old (less than 1 hour ago), skipping auto backup...")
		
				}
			}
		} else {
			log.Println("Last backup time not found, skipping performing auto backup...")
		}

	}

	if tick%60 == 0 {
		ip := taskGetSystemIP()
		log.Println("System IP is: " + ip)
	}

	if tick%3600 == 0 {
		// Executing every 3600 ticks (1 hour)
	}

	// Just add a schedule here if you need a custom one (every "tick hour", every "tick day", etc...)

}

func taskSetupBackups(args taskSetupBackupsArgs) string {
	fmt.Println("Executing taskSetupBackups" + args.Service)
	// ...
	service_url := ""
	key_id_name := "AWS_ACCESS_KEY_ID"
	key_secret_name := "AWS_SECRET_ACCESS_KEY"
	repo_location := "/home/system/components/apps/"
	service_found := false

	switch args.Service {
		case "s3":
			service_url = "s3.amazonaws.com/"
			service_found = true
		case "b2":
			service_url = ""
			key_id_name = "B2_ACCOUNT_ID"
			key_secret_name = "B2_ACCOUNT_KEY"
			service_found = true
		case "wasabi":
			service_found = true
			service_url = "s3.wasabisys.com/"
	}

	if !service_found {
		fmt.Println("Service not found")
		return "{\"status\": \"error\", \"message\": \"Service not found\"}"
	}

	fmt.Println("Creating env vars for authentication with backup service")
	os.Setenv(key_id_name, args.AccessKeyID)
	os.Setenv(key_secret_name, args.SecretAccessKey)

	fmt.Println("Creating restic password file")
	
	system.CreateBackupsPasswordFile(args.RepositoryPassword)

	fmt.Println("Initializing restic repository")
	utils.WriteOption("BACKUP_IS_WORKING", "1")

	cmdArgs := []string{"-r", args.Service + ":" + service_url + args.RepositoryName + ":" + repo_location, "init", "--password-file", utils.GetPath(utils.BackupPasswordFileLocation), "--verbose=3"}
	
	result := utils.ExecAndStream(repo_location, "restic", cmdArgs)

	utils.WriteOption("BACKUP_IS_WORKING", "0")

	// Write backup settings to table
	utils.WriteOption("BACKUP_SERVICE", args.Service)
	utils.WriteOption("BACKUP_SERVICE_URL", service_url)
	utils.WriteOption("BACKUP_REPOSITORY_NAME", args.RepositoryName)
	utils.WriteOption("BACKUP_REPOSITORY_PASSWORD", args.RepositoryPassword)
	utils.WriteOption("BACKUP_REPOSITORY_ACCESS_KEY_ID", args.AccessKeyID)
	utils.WriteOption("BACKUP_REPOSITORY_SECRET_ACCESS_KEY", args.SecretAccessKey)
	utils.WriteOption("BACKUP_REPOSITORY_LOCATION", repo_location)

	// See if result contains the substring "Fatal:"
	if strings.Contains(result, "Fatal:") {
		fmt.Println("Error initializing restic repository")

		utils.WriteOption("BACKUP_STATUS", "error")
		utils.WriteOption("BACKUP_ERROR_MESSAGE", result)

		return "{\"status\": \"error\", \"message\": \"" + result + "\"}"
	}

	// Save options to database
	utils.WriteOption("BACKUP_STATUS", "initiated")

	// Populate Stats right away
	taskGetBackupStatus()
	
	return "{\"status\": \"ok\"}"
	
}

func taskRemoveBackups() string {

	fmt.Println("Executing taskRemoveBackups")

	// ...	This deletes the restic repository
	// cmdArgs := []string{"-r", "s3:https://s3.amazonaws.com/edgebox-backups:/home/system/components/apps/", "forget", "latest", "--password-file", utils.GetPath(utils.BackupPasswordFileLocation), "--verbose=3"}
	
	utils.WriteOption("BACKUP_STATUS", "")
	utils.WriteOption("BACKUP_IS_WORKING", "0")

	return "{\"status\": \"ok\"}"
	
}

func taskBackup() string {
	fmt.Println("Executing taskBackup")

	// Load Backup Options
	backup_service := utils.ReadOption("BACKUP_SERVICE")
	backup_service_url := utils.ReadOption("BACKUP_SERVICE_URL")
	backup_repository_name := utils.ReadOption("BACKUP_REPOSITORY_NAME")
	// backup_repository_password := utils.ReadOption("BACKUP_REPOSITORY_PASSWORD")
	backup_repository_access_key_id := utils.ReadOption("BACKUP_REPOSITORY_ACCESS_KEY_ID")
	backup_repository_secret_access_key := utils.ReadOption("BACKUP_REPOSITORY_SECRET_ACCESS_KEY")
	backup_repository_location := utils.ReadOption("BACKUP_REPOSITORY_LOCATION")

	key_id_name := "AWS_ACCESS_KEY_ID"
	key_secret_name := "AWS_SECRET_ACCESS_KEY"
	service_found := false

	switch backup_service {
		case "s3":
			service_found = true
		case "b2":
			key_id_name = "B2_ACCOUNT_ID"
			key_secret_name = "B2_ACCOUNT_KEY"
			service_found = true
		case "wasabi":
			service_found = true
	}

	if !service_found {
		fmt.Println("Service not found")
		return "{\"status\": \"error\", \"message\": \"Backup Service not found\"}"
	}

	fmt.Println("Creating env vars for authentication with backup service")
	fmt.Println(key_id_name)
	os.Setenv(key_id_name, backup_repository_access_key_id)
	fmt.Println(key_secret_name)
	os.Setenv(key_secret_name, backup_repository_secret_access_key)


	utils.WriteOption("BACKUP_IS_WORKING", "1")

	// ...	This backs up the restic repository
	cmdArgs := []string{"-r", backup_service + ":" + backup_service_url + backup_repository_name + ":" + backup_repository_location, "backup", backup_repository_location, "--password-file", utils.GetPath(utils.BackupPasswordFileLocation), "--verbose=3"}
	result := utils.ExecAndStream(backup_repository_location, "restic", cmdArgs)

	utils.WriteOption("BACKUP_IS_WORKING", "0")
	// Write as Unix timestamp
	utils.WriteOption("BACKUP_LAST_RUN", strconv.FormatInt(time.Now().Unix(), 10))

	// See if result contains the substring "Fatal:"
	if strings.Contains(result, "Fatal:") {
		fmt.Println("Error backing up")
		utils.WriteOption("BACKUP_STATUS", "error")
		utils.WriteOption("BACKUP_ERROR_MESSAGE", result)
		return "{\"status\": \"error\", \"message\": \"" + result + "\"}"
	}

	utils.WriteOption("BACKUP_STATUS", "working")
	taskGetBackupStatus()
	return "{\"status\": \"ok\"}"
	
}

func taskRestoreBackup() string {
	fmt.Println("Executing taskRestoreBackup")

	// Load Backup Options
	backup_service := utils.ReadOption("BACKUP_SERVICE")
	backup_service_url := utils.ReadOption("BACKUP_SERVICE_URL")
	backup_repository_name := utils.ReadOption("BACKUP_REPOSITORY_NAME")
	// backup_repository_password := utils.ReadOption("BACKUP_REPOSITORY_PASSWORD")
	backup_repository_access_key_id := utils.ReadOption("BACKUP_REPOSITORY_ACCESS_KEY_ID")
	backup_repository_secret_access_key := utils.ReadOption("BACKUP_REPOSITORY_SECRET_ACCESS_KEY")
	backup_repository_location := utils.ReadOption("BACKUP_REPOSITORY_LOCATION")

	key_id_name := "AWS_ACCESS_KEY_ID"
	key_secret_name := "AWS_SECRET_ACCESS_KEY"
	service_found := false

	switch backup_service {
		case "s3":
			service_found = true
		case "b2":
			key_id_name = "B2_ACCOUNT_ID"
			key_secret_name = "B2_ACCOUNT_KEY"
			service_found = true
		case "wasabi":
			service_found = true
	}

	if !service_found {
		fmt.Println("Service not found")
		return "{\"status\": \"error\", \"message\": \"Backup Service not found\"}"
	}

	fmt.Println("Creating env vars for authentication with backup service")
	fmt.Println(key_id_name)
	os.Setenv(key_id_name, backup_repository_access_key_id)
	fmt.Println(key_secret_name)
	os.Setenv(key_secret_name, backup_repository_secret_access_key)


	utils.WriteOption("BACKUP_IS_WORKING", "1")

	fmt.Println("Stopping All EdgeApps")
	// Stop All EdgeApps
	edgeapps.StopAllEdgeApps()

	// Copy all files in /home/system/components/apps/ to a backup folder
	fmt.Println("Copying all files in /home/system/components/apps/ to a backup folder")
	os.MkdirAll(utils.GetPath(utils.EdgeAppsBackupPath + "temp/"), 0777)
	system.CopyDir(utils.GetPath(utils.EdgeAppsPath), utils.GetPath(utils.EdgeAppsBackupPath + "temp/"))

	fmt.Println("Removing all files in /home/system/components/apps/")
	os.RemoveAll(utils.GetPath(utils.EdgeAppsPath))

	// Create directory /home/system/components/apps/
	fmt.Println("Creating directory /home/system/components/apps/")
	os.MkdirAll(utils.GetPath(utils.EdgeAppsPath), 0777)

	// ...	This restores up the restic repository
	cmdArgs := []string{"-r", backup_service + ":" + backup_service_url + backup_repository_name + ":" + backup_repository_location, "restore", "latest", "--target", "/", "--path", backup_repository_location, "--password-file", utils.GetPath(utils.BackupPasswordFileLocation), "--verbose=3"}
	result := utils.ExecAndStream(backup_repository_location, "restic", cmdArgs)

	taskGetBackupStatus()

	edgeapps.RestartEdgeAppsService()

	utils.WriteOption("BACKUP_IS_WORKING", "0")

	// See if result contains the substring "Fatal:"
	if strings.Contains(result, "Fatal:") {
		// Copy all files from backup folder to /home/system/components/apps/
		os.MkdirAll(utils.GetPath(utils.EdgeAppsPath), 0777)
		system.CopyDir(utils.GetPath(utils.EdgeAppsBackupPath + "temp/"), utils.GetPath(utils.EdgeAppsPath))

		fmt.Println("Error restoring backup: ")
		utils.WriteOption("BACKUP_STATUS", "error")
		utils.WriteOption("BACKUP_ERROR_MESSAGE", result)
		return "{\"status\": \"error\", \"message\": \"" + result + "\"}"
	}

	utils.WriteOption("BACKUP_STATUS", "working")
	taskGetBackupStatus()
	return "{\"status\": \"ok\"}"
	
}

func taskAutoBackup() string {
	fmt.Println("Executing taskAutoBackup")

	// Get Backup Status
	backup_status := utils.ReadOption("BACKUP_STATUS")
	// We only backup is the status is "working"
	if backup_status == "working" {
		return taskBackup()
	} else {
		fmt.Println("Backup status is not working... skipping")
		return "{\"status\": \"skipped\"}"		
	}
}

func taskGetBackupStatus() string {
	fmt.Println("Executing taskGetBackupStatus")

	// Load Backup Options
	backup_service := utils.ReadOption("BACKUP_SERVICE")
	backup_service_url := utils.ReadOption("BACKUP_SERVICE_URL")
	backup_repository_name := utils.ReadOption("BACKUP_REPOSITORY_NAME")
	// backup_repository_password := utils.ReadOption("BACKUP_REPOSITORY_PASSWORD")
	backup_repository_access_key_id := utils.ReadOption("BACKUP_REPOSITORY_ACCESS_KEY_ID")
	backup_repository_secret_access_key := utils.ReadOption("BACKUP_REPOSITORY_SECRET_ACCESS_KEY")
	backup_repository_location := utils.ReadOption("BACKUP_REPOSITORY_LOCATION")

	key_id_name := "AWS_ACCESS_KEY_ID"
	key_secret_name := "AWS_SECRET_ACCESS_KEY"
	service_found := false

	switch backup_service {
		case "s3":
			service_found = true
		case "b2":
			key_id_name = "B2_ACCOUNT_ID"
			key_secret_name = "B2_ACCOUNT_KEY"
			service_found = true
		case "wasabi":
			service_found = true
	}

	if !service_found {
		fmt.Println("Service not found")
		return "{\"status\": \"error\", \"message\": \"Backup Service not found\"}"
	}

	fmt.Println("Creating env vars for authentication with backup service")
	os.Setenv(key_id_name, backup_repository_access_key_id)
	os.Setenv(key_secret_name, backup_repository_secret_access_key)

	// ...	This gets the restic repository status
	cmdArgs := []string{"-r", backup_service + ":" + backup_service_url + backup_repository_name + ":" + backup_repository_location, "stats", "--password-file", utils.GetPath(utils.BackupPasswordFileLocation), "--verbose=3"}
	utils.WriteOption("BACKUP_STATS", utils.ExecAndStream(backup_repository_location, "restic", cmdArgs))

	return "{\"status\": \"ok\"}"
	
}

func taskSetupTunnel(args taskSetupTunnelArgs) string {
	fmt.Println("Executing taskSetupTunnel")
	wsPath := utils.GetPath(utils.WsPath)	

	// Stop a the service if it is running
	system.StopService("cloudflared")

	// Uninstall the service if it is installed
	system.RemoveTunnelService()

	fmt.Println("Creating cloudflared folder")
	cmdargs := []string{"/home/system/.cloudflared"}
	utils.Exec(wsPath, "mkdir", cmdargs)

	cmd := exec.Command("sh", "/home/system/components/edgeboxctl/scripts/cloudflared_login.sh")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(stdout)
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	url := ""
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		text := scanner.Text()
		if strings.Contains(text, "https://") {
			url = text
			fmt.Println("Tunnel setup is requesting auth with URL: " + url)
			status := "{\"status\": \"waiting\", \"login_link\": \"" + url + "\"}"
			utils.WriteOption("TUNNEL_STATUS", status)
			break
		}
	}
	if scanner.Err() != nil {
		cmd.Process.Kill()
		cmd.Wait()
		panic(scanner.Err())
	}

	go func() {
		fmt.Println("Running async")
		cmd.Wait()

		// Keep retrying to read cert.pem file until it is created
		// When running as a service, the cert is saved to a different folder,
		// so we check both :)
		for {
			_, err := os.Stat("/home/system/.cloudflared/cert.pem")
			_, err2 := os.Stat("/root/.cloudflared/cert.pem")
			if err == nil || err2 == nil {
				fmt.Println("cert.pem file detected")
				break
			}
			time.Sleep(1 * time.Second)
			fmt.Println("Waiting for cert.pem file to be created")
		}

		fmt.Println("Tunnel auth setup finished without errors.")
		status := "{\"status\": \"starting\", \"login_link\": \"" + url + "\"}"
		utils.WriteOption("TUNNEL_STATUS", status)

		// Remove old tunnel if it exists, and create from scratch
		system.DeleteTunnel()

		// Create new tunnel (destination config file is param)
		system.CreateTunnel("/home/system/.cloudflared/config.yml")

		fmt.Println("Creating DNS Routes for @ and *.")
		cmd = exec.Command("cloudflared", "tunnel", "route", "dns", "-f" ,"edgebox", "*." + args.DomainName)
		cmd.Start()
		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}

		cmd = exec.Command("cloudflared", "tunnel", "route", "dns", "-f" ,"edgebox", args.DomainName)
		cmd.Start()
		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}

		domainNameInfo := args.DomainName
		utils.WriteOption("DOMAIN_NAME", domainNameInfo)

		// Install service with given config file
		system.InstallTunnelService("/home/system/.cloudflared/config.yml")

		// Start the service
		system.StartService("cloudflared")

		if err != nil {
			fmt.Println("Tunnel auth setup finished with errors.")
			status := "{\"status\": \"error\", \"login_link\": \"" + url + "\"}"
			utils.WriteOption("TUNNEL_STATUS", status)
			log.Fatal(err)
		} else {
			fmt.Println("Tunnel auth setup finished without errors.")
			status := "{\"status\": \"connected\", \"login_link\": \"" + url + "\", \"domain\": \"" + args.DomainName + "\"}"
			utils.WriteOption("TUNNEL_STATUS", status)
		}

		fmt.Println("Finished running async")
	}()

    return "{\"url\": \"" + url + "\"}"
}

func taskStartTunnel() string {
	fmt.Println("Executing taskStartTunnel")
	system.StartService("cloudflared")
	domainName := utils.ReadOption("DOMAIN_NAME")
	status := "{\"status\": \"connected\", \"domain\": \"" + domainName + "\"}"
	utils.WriteOption("TUNNEL_STATUS", status)
	return "{\"status\": \"ok\"}"
}

func taskStopTunnel() string {
	fmt.Println("Executing taskStopTunnel")
	system.StopService("cloudflared")
	domainName := utils.ReadOption("DOMAIN_NAME")
	status := "{\"status\": \"stopped\", \"domain\": \"" + domainName + "\"}"
	utils.WriteOption("TUNNEL_STATUS", status)
	return "{\"status\": \"ok\"}"
}

func taskDisableTunnel() string {
	fmt.Println("Executing taskDisableTunnel")
	system.StopService("cloudflared")
	system.DeleteTunnel()
	system.RemoveTunnelService()
	utils.DeleteOption("DOMAIN_NAME")
	utils.DeleteOption("TUNNEL_STATUS")
	return "{\"status\": \"ok\"}"
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

		utils.WriteOption("PUBLIC_DASHBOARD", args.InternetURL)
		return "{result: true}"

	}

	return "{result: false}"
}

func taskDisablePublicDashboard() string {
	fmt.Println("Executing taskDisablePublicDashboard")
	result := edgeapps.DisablePublicDashboard()
	utils.WriteOption("PUBLIC_DASHBOARD", "")
	if result {
		return "{result: true}"
	}
	return "{result: false}"
}

func taskSetReleaseVersion() string {

	fmt.Println("Executing taskSetReleaseVersion")

	utils.WriteOption("RELEASE_VERSION", diagnostics.Version)

	return diagnostics.Version
}

func taskGetEdgeApps() string {
	fmt.Println("Executing taskGetEdgeApps")

	edgeApps := edgeapps.GetEdgeApps()
	edgeAppsJSON, _ := json.Marshal(edgeApps)

	utils.WriteOption("EDGEAPPS_LIST", string(edgeAppsJSON))
	return string(edgeAppsJSON)
}

func taskGetSystemUptime() string {
	fmt.Println("Executing taskGetSystemUptime")
	uptime := system.GetUptimeInSeconds()
	utils.WriteOption("SYSTEM_UPTIME", uptime)
	return uptime
}

func taskGetStorageDevices() string {
	fmt.Println("Executing taskGetStorageDevices")

	devices := storage.GetDevices(diagnostics.GetReleaseVersion())
	devicesJSON, _ := json.Marshal(devices)

	utils.WriteOption("STORAGE_DEVICES_LIST", string(devicesJSON))

	return string(devicesJSON)
}

func taskGetSystemIP() string {
	fmt.Println("Executing taskGetStorageDevices")
	ip := system.GetIP()
	utils.WriteOption("IP_ADDRESS", ip)
	return ip
}

func taskGetHostname() string {
	fmt.Println("Executing taskGetHostname")
	hostname := system.GetHostname()
	utils.WriteOption("HOSTNAME", hostname)
	return hostname
}

func taskSetupCloudOptions() {
	fmt.Println("Executing taskSetupCloudOptions")
	system.SetupCloudOptions()
}
