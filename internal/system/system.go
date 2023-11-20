package system

import (
	"fmt"
	"strconv"
	"strings"
	"log"
	"os"
	"io"
	"os/exec"
	"bufio"
	"path/filepath"
	"io/ioutil"
	"encoding/json"

	"github.com/edgebox-iot/edgeboxctl/internal/utils"

	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/host"
)

type cloudflaredTunnelJson struct {
	AccountTag string `json:"AccountTag"`
	TunnelSecret string `json:"TunnelSecret"`
	TunnelID string `json:"TunnelID"`
}

// GetUptimeInSeconds: Returns a value (as string) of the total system uptime
func GetUptimeInSeconds() string {
	uptime, _ := host.Uptime()

	return strconv.FormatUint(uptime, 10)
}

// GetUptimeFormatted: Returns a humanized version that can be useful for logging
func GetUptimeFormatted() string {
	uptime, _ := host.Uptime()

	days := uptime / (60 * 60 * 24)
	hours := (uptime - (days * 60 * 60 * 24)) / (60 * 60)
	minutes := ((uptime - (days * 60 * 60 * 24)) - (hours * 60 * 60)) / 60
	return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
}

// GetIP: Returns the ip address of the instance
func GetIP() string {
	ip := ""

	// Trying to find a valid IP (For direct connection, not tunneled)
	ethResult := utils.ExecAndGetLines("/", "ip", []string{"-o", "-4", "addr", "list", "eth0"})
	for ethResult.Scan() {
		adapterRawInfo := strings.Fields(ethResult.Text())
		if ip == "" {
			ip = strings.Split(adapterRawInfo[3], "/")[0]
		}
	}

	// If no IP was found yet, try wlan0
	if ip == "" {
		wlanResult := utils.ExecAndGetLines("/", "ip", []string{"-o", "-4", "addr", "list", "wlan0"})
		for wlanResult.Scan() {
			adapterRawInfo := strings.Fields(wlanResult.Text())
			if ip == "" {
				ip = strings.Split(adapterRawInfo[3], "/")[0]
			}
		}
	}

	return ip
}

func GetHostname() string {
	return utils.Exec("/", "hostname", []string{})
}

// SetupCloudOptions: Reads the designated env file looking for options to write into the options table. Meant to be used on initial setup. Deletes source env file after operation.
func SetupCloudOptions() {

	var cloudEnv map[string]string
	cloudEnvFileLocationPath := utils.GetPath(utils.CloudEnvFileLocation)
	cloudEnv, err := godotenv.Read(cloudEnvFileLocationPath)

	if err != nil {
		fmt.Println("Error loading .env file for cloud version setup")
		return
	}

	if cloudEnv["NAME"] != "" {
		utils.WriteOption("NAME", cloudEnv["NAME"])
	}

	if cloudEnv["EMAIL"] != "" {
		utils.WriteOption("EMAIL", cloudEnv["EMAIL"])
	}

	if cloudEnv["USERNAME"] != "" {
		utils.WriteOption("USERNAME", cloudEnv["USERNAME"])
	}

	if cloudEnv["CLUSTER"] != "" {
		utils.WriteOption("CLUSTER", cloudEnv["CLUSTER"])
	}

	if cloudEnv["CLUSTER_IP"] != "" {
		utils.WriteOption("CLUSTER_IP", cloudEnv["CLUSTER_IP"])
	}

	if cloudEnv["CLUSTER_SSH_PORT"] != "" {
		utils.WriteOption("CLUSTER_SSH_PORT", cloudEnv["CLUSTER_SSH_PORT"])
	}

	if cloudEnv["EDGEBOXIO_API_TOKEN"] != "" {
		utils.WriteOption("EDGEBOXIO_API_TOKEN", cloudEnv["EDGEBOXIO_API_TOKEN"])
	}

	// In the end of this operation takes place, remove the env file as to not overwrite any options once they are set.
	utils.Exec("/", "rm", []string{cloudEnvFileLocationPath})
}

func StartSystemLogger() {
	fmt.Println("Starting system logger")
	loggerPath := utils.GetPath(utils.LoggerPath)
	utils.Exec(loggerPath, "make", []string{"start"})
}

// UpdateSystemLoggerServices: Updates the services.txt file with the services that are currently running
func UpdateSystemLoggerServices(services []string) {
	fmt.Println("Updating system logger services:")
	fmt.Println(services)
	loggerPath := utils.GetPath(utils.LoggerPath)

	utils.Exec(loggerPath, "bash", []string{"-c", "rm services.txt && touch services.txt"})

	for _, service := range services {
		fmt.Println("Adding " + service + " to services.txt")
		utils.Exec(loggerPath, "bash", []string{"-c", "echo " + service + " >> services.txt"})
	}

	// Add empty line at the end of file (best practice)
	utils.Exec(loggerPath, "bash", []string{"-c", "echo '' >> services.txt"})
}

// StartWs: Starts the webserver service for Edgeapps
func StartWs() {
	wsPath := utils.GetPath(utils.WsPath)
	fmt.Println("Starting WS")
	cmdargs := []string{"-b"}
	utils.Exec(wsPath, "./ws", cmdargs)
}

// StartService: Starts a service
func StartService(serviceID string) {
	wsPath := utils.GetPath(utils.WsPath)
	fmt.Println("Starting" + serviceID + "service")
	cmdargs := []string{"start", serviceID}
	utils.Exec(wsPath, "systemctl", cmdargs)
}

// StopService: Stops a service
func StopService(serviceID string) {
	wsPath := utils.GetPath(utils.WsPath)
	fmt.Println("Stopping" + serviceID + "service")
	cmdargs := []string{"stop", "cloudflared"}
	utils.Exec(wsPath, "systemctl", cmdargs)
}

// RestartService: Restarts a service
func RestartService(serviceID string) {
	wsPath := utils.GetPath(utils.WsPath)	
	fmt.Println("Restarting" + serviceID + "service")
	cmdargs := []string{"restart", serviceID}
	utils.Exec(wsPath, "systemctl", cmdargs)
}

// GetServiceStatus: Returns the status output of a service
func GetServiceStatus(serviceID string) string {
	wsPath := utils.GetPath(utils.WsPath)	
	cmdargs := []string{"status", serviceID}
	return utils.Exec(wsPath, "systemctl", cmdargs)
}

func CreateBackupsPasswordFile(password string) {
	// Create a password file for backups
	backupPasswordFile := utils.GetPath(utils.BackupPasswordFileLocation)
	backupPasswordFileDir := filepath.Dir(backupPasswordFile)

	if _, err := os.Stat(backupPasswordFileDir); os.IsNotExist(err) {
		os.MkdirAll(backupPasswordFileDir, 0755)
	}

	// Write the password to the file, overriting an existing file
	err := ioutil.WriteFile(backupPasswordFile, []byte(password), 0644)
	if err != nil {
		panic(err)
	}
} 

// CreateTunnel: Creates a tunnel via cloudflared, needs to be authenticated first
func CreateTunnel(configDestination string) {
	fmt.Println("Creating Tunnel for Edgebox.")
	cmd := exec.Command("sh", "/home/system/components/edgeboxctl/scripts/cloudflared_tunnel_create.sh")
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

	fmt.Println("Tunnel ID is:" + data.TunnelID)

	// create the config.yml file with the following content in each line:
	// "url": "http://localhost:80"
	// "tunnel": "<TunnelID>"
	// "credentials-file": "/root/.cloudflared/<tunnelID>.json"

	file := configDestination
	f, err := os.Create(file)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	_, err = f.WriteString("url: http://localhost:80\ntunnel: " + data.TunnelID + "\ncredentials-file: " + jsonFilePath)

	if err != nil {
		panic(err)
	}
}

// DeleteTunnel: Deletes a tunnel via cloudflared, this does not remove the service
func DeleteTunnel() {
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
}

// InstallTunnelService: Installs the tunnel service
func InstallTunnelService(config string) {
	fmt.Println("Installing cloudflared service.")
	cmd := exec.Command("cloudflared", "--config", config, "service", "install")
	cmd.Start()
	cmd.Wait()
}

// RemoveTunnelService: Removes the tunnel service
func RemoveTunnelService() {
	wsPath := utils.GetPath(utils.WsPath)	
	fmt.Println("Removing possibly previous service install.")
	cmd := exec.Command("cloudflared", "service", "uninstall")
	cmd.Start()
	cmd.Wait()

	fmt.Println("Removing cloudflared files")
	cmdargs := []string{"-rf", "/home/system/.cloudflared"}
	utils.Exec(wsPath, "rm", cmdargs)
	cmdargs = []string{"-rf", "/etc/cloudflared/config.yml"}
	utils.Exec(wsPath, "rm", cmdargs)
	cmdargs = []string{"-rf", "/root/.cloudflared/cert.pem"}
	utils.Exec(wsPath, "rm", cmdargs)
}

func CopyDir(src string, dest string) error {
    srcInfo, err := os.Stat(src)
    if err != nil {
        return err
    }
    if !srcInfo.IsDir() {
        return fmt.Errorf("%s is not a directory", src)
    }

    err = os.MkdirAll(dest, srcInfo.Mode())
    if err != nil {
        return err
    }

    items, err := ioutil.ReadDir(src)
    if err != nil {
        return err
    }

    for _, item := range items {
        srcPath := filepath.Join(src, item.Name())
        destPath := filepath.Join(dest, item.Name())

        if item.IsDir() {
            err = CopyDir(srcPath, destPath)
            if err != nil {
                fmt.Printf("error copying directory %s to %s: %s\n", srcPath, destPath, err.Error())
            }
        } else {
            err = CopyFile(srcPath, destPath)
            if err != nil {
                fmt.Printf("error copying file %s to %s: %s\n", srcPath, destPath, err.Error())
            }
        }
    }

    return nil
}

func CopyFile(src string, dest string) error {
    srcFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer srcFile.Close()

    destFile, err := os.Create(dest)
    if err != nil {
        return err
    }
    defer destFile.Close()

    _, err = io.Copy(destFile, srcFile)
    if err != nil {
        return err
    }

    err = destFile.Sync()
    if err != nil {
        return err
    }

    srcInfo, err := os.Stat(src)
    if err != nil {
        return err
    }

    err = os.Chmod(dest, srcInfo.Mode())
    if err != nil {
        return err
    }

    return nil
}

