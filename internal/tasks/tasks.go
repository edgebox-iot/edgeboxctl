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
	"io/ioutil"
	"bufio"
	"path/filepath"

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

type cloudflaredTunnelJson struct {
	AccountTag string `json:"AccountTag"`
	TunnelSecret string `json:"TunnelSecret"`
	TunnelID string `json:"TunnelID"`
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
		_, err = statement.Exec(STATUS_FINISHED, task.Result.String, formatedDatetime, strconv.Itoa(task.ID)) // Execute SQL Statement with result info
		if err != nil {
			log.Fatal(err.Error())
		}
	} else {
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
	}

	if tick%60 == 0 {
		ip := taskGetSystemIP()
		log.Println("System IP is: " + ip)
	}

	// Just add a schedule here if you need a custom one (every "tick hour", every "tick day", etc...)

}

func taskSetupTunnel(args taskSetupTunnelArgs) string {
	fmt.Println("Executing taskSetupTunnel")
	wsPath := utils.GetPath(utils.WsPath)	

	// Stop a the service if it is running
	fmt.Println("Stopping cloudflared service")
	cmdargs := []string{"stop", "cloudflared"}
	utils.Exec(wsPath, "systemctl", cmdargs)

	fmt.Println("Removing possibly previous service install.")
	cmd := exec.Command("cloudflared", "service", "uninstall")
	cmd.Start()
	cmd.Wait()

	fmt.Println("Removing cloudflared files")
	cmdargs = []string{"-rf", "/home/system/.cloudflared"}
	utils.Exec(wsPath, "rm", cmdargs)
	cmdargs = []string{"-rf", "/etc/cloudflared/config.yml"}
	utils.Exec(wsPath, "rm", cmdargs)
	cmdargs = []string{"-rf", "/root/.cloudflared/cert.pem"}
	utils.Exec(wsPath, "rm", cmdargs)

	fmt.Println("Creating cloudflared folder")
	cmdargs = []string{"/home/system/.cloudflared"}
	utils.Exec(wsPath, "mkdir", cmdargs)

	cmd = exec.Command("sh", "/home/system/components/edgeboxctl/scripts/cloudflared_login.sh")
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

		// fmt.Println("Moving certificate to global folder.")
		// cmdargs = []string{"/home/system/.cloudflared/cert.pem", "/etc/cloudflared/cert.pem"}
		// utils.Exec(wsPath, "cp", cmdargs)

		fmt.Println("Deleting possible previous tunnel.")
		// Configure the service and start it
		cmd := exec.Command("sh", "/home/system/components/edgeboxctl/scripts/cloudflared_tunnel_delete.sh")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(stdout)
		err = cmd.Start()
		if err != nil {
			panic(err)
		}
		for scanner.Scan() {
			fmt.Println(scanner.Text())
			text := scanner.Text()
			fmt.Println(text)
		}
		if scanner.Err() != nil {
			cmd.Process.Kill()
			cmd.Wait()
			panic(scanner.Err())
		}
		

		fmt.Println("Creating Tunnel for Edgebox.")
		cmd = exec.Command("sh", "/home/system/components/edgeboxctl/scripts/cloudflared_tunnel_create.sh")
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		scanner = bufio.NewScanner(stdout)
		err = cmd.Start()
		if err != nil {
			panic(err)
		}
		for scanner.Scan() {
			fmt.Println(scanner.Text())
			text := scanner.Text()
			fmt.Println(text)
		}
		if scanner.Err() != nil {
			cmd.Process.Kill()
			cmd.Wait()
			panic(scanner.Err())
		}

		// This also needs to be executed in root and non root variants

		fmt.Println("Reading cloudflared folder to get the JSON file.")
		isRoot := false
		dir := "/home/system/.cloudflared/"
		dir2 := "/root/.cloudflared/"
		files, err := os.ReadDir(dir)
		if err != nil {
			panic(err)
		}

		var jsonFile os.DirEntry
		for _, file := range files {
			// check if file has json extension
			if filepath.Ext(file.Name()) == ".json" {
				fmt.Println("Non-Root JSON file found: " + file.Name())
				jsonFile = file
			}
		}

		// If the files are not in the home folder, try the root folder
		if jsonFile == nil {
			files, err = os.ReadDir(dir2)
			if err != nil {
				panic(err)
			}
			for _, file := range files {
				// check if file has json extension
				if filepath.Ext(file.Name()) == ".json" {
					fmt.Println("Root JSON file found: " + file.Name())
					jsonFile = file
					isRoot = true
				}
			}
		}

		if jsonFile == nil {
			panic("No JSON file found in directory")
		}

		fmt.Println("Reading JSON file.")
		targetDir := "/home/system/.cloudflared/"
		if isRoot {
			targetDir = "/root/.cloudflared/"
		}

		jsonFilePath := filepath.Join(targetDir, jsonFile.Name())
		jsonBytes, err := ioutil.ReadFile(jsonFilePath)
		if err != nil {
			panic(err)
		}

		fmt.Println("Parsing JSON file.")
		var data cloudflaredTunnelJson
		err = json.Unmarshal(jsonBytes, &data)
		if err != nil {
			log.Printf("Error reading tunnel JSON file: %s", err)
		}

		fmt.Println(data)

		// print propertie tunnel_id from json file
		fmt.Println("Tunnel ID is:" + data.TunnelID)

		// create the config.yml file with the following content in each line:
		// "url": "http://localhost:80"
		// "tunnel": "<TunnelID>"
		// "credentials-file": "/root/.cloudflared/<tunnelID>.json"

		file := "/home/system/.cloudflared/config.yml"
		f, err := os.Create(file)
		if err != nil {
			panic(err)
		}

		defer f.Close()

		_, err = f.WriteString("url: http://localhost:80\ntunnel: " + data.TunnelID + "\ncredentials-file: " + jsonFilePath)

		if err != nil {
			panic(err)
		}

		fmt.Println("Creating DNS Routes for @ and *.")
		cmd = exec.Command("cloudflared", "tunnel", "route", "dns", "-f" ,"edgebox", "*." + args.DomainName)
		cmd.Start()
		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}

		domainNameInfo := args.DomainName
		utils.WriteOption("DOMAIN_NAME", domainNameInfo)

		cmd = exec.Command("cloudflared", "tunnel", "route", "dns", "-f" ,"edgebox", args.DomainName)
		cmd.Start()
		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Installing systemd service.")
		cmd = exec.Command("cloudflared", "--config", "/home/system/.cloudflared/config.yml", "service", "install")
		cmd.Start()
		cmd.Wait()

		fmt.Println("Starting tunnel.")
		cmd = exec.Command("systemctl", "start", "cloudflared")
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		scanner = bufio.NewScanner(stdout)
		err = cmd.Start()
		if err != nil {
			panic(err)
		}
		for scanner.Scan() {
			fmt.Println(scanner.Text())
			text := scanner.Text()
			fmt.Println(text)
		}
		if scanner.Err() != nil {
			cmd.Process.Kill()
			cmd.Wait()
			panic(scanner.Err())
		}

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
