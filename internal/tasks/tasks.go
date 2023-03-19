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
