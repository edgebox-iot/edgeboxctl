package edgeapps

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"github.com/edgebox-iot/sysctl/internal/utils"
)

// EdgeApp : Struct representing an EdgeApp in the system
type EdgeApp struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Status     EdgeAppStatus    `json:"status"`
	Services   []EdgeAppService `json:"services"`
	NetworkURL string           `json:"network_url"`
}

// EdgeAppStatus : Struct representing possible EdgeApp statuses (code + description)
type EdgeAppStatus struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// EdgeAppService : Struct representing a single container that can be part of an EdgeApp package
type EdgeAppService struct {
	ID        string `json:"id"`
	IsRunning bool   `json:"is_running"`
}

const configFilename = "/edgebox-compose.yml"
const envFilename = "/edgebox.env"

// GetEdgeApps : Returns a list of EdgeApp struct filled with information
func GetEdgeApps() []EdgeApp {

	var edgeApps []EdgeApp

	// Building list of available edgeapps in the system with their status

	files, err := ioutil.ReadDir(utils.GetPath("edgeAppsPath"))
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if f.IsDir() {
			// It is a folder that most probably contains an EdgeApp.
			// To be fully sure, test that edgebox-compose.yml file exists in the target directory.
			_, err := os.Stat(utils.GetPath("edgeAppsPath") + f.Name() + configFilename)
			if !os.IsNotExist(err) {
				// File exists. Start digging!

				edgeAppName := f.Name()

				edgeAppEnv, err := godotenv.Read(utils.GetPath("edgeAppsPath") + f.Name() + envFilename)

				if err != nil {
					log.Println("Error loading .env file for edgeapp " + f.Name())
				} else {
					if edgeAppEnv["EDGEAPP_NAME"] != "" {
						edgeAppName = edgeAppEnv["EDGEAPP_NAME"]
					}
				}

				edgeApp := EdgeApp{ID: f.Name(), Name: edgeAppName, Status: GetEdgeAppStatus(f.Name()), Services: GetEdgeAppServices(f.Name()), NetworkURL: f.Name() + ".edgebox.local"}
				edgeApps = append(edgeApps, edgeApp)
			}

		}
	}

	// return edgeApps
	return edgeApps
}

// GetEdgeAppStatus : Returns a struct representing the current status of the EdgeApp
func GetEdgeAppStatus(ID string) EdgeAppStatus {

	// Possible states of an EdgeApp:
	// - All services running = EdgeApp running
	// - Some services running = Problem detected, needs restart
	// - No service running = EdgeApp is off

	runningServices := 0
	status := EdgeAppStatus{0, "off"}
	services := GetEdgeAppServices(ID)
	for _, edgeAppService := range services {
		if edgeAppService.IsRunning {
			runningServices++
		}
	}

	if runningServices > 0 && runningServices != len(services) {
		status = EdgeAppStatus{2, "error"}
	}

	if runningServices == len(services) {
		status = EdgeAppStatus{1, "on"}
	}

	return status

}

// GetEdgeAppServices : Returns a
func GetEdgeAppServices(ID string) []EdgeAppService {

	cmdArgs := []string{"-r", ".services | keys[]", utils.GetPath("edgeAppsPath") + ID + configFilename}
	servicesString := utils.Exec("yq", cmdArgs)
	serviceSlices := strings.Split(servicesString, "\n")
	serviceSlices = utils.DeleteEmptySlices(serviceSlices)
	var edgeAppServices []EdgeAppService

	for _, serviceID := range serviceSlices {
		cmdArgs = []string{"-f", utils.GetPath("wsPath") + "/docker-compose.yml", "ps", "-q", serviceID}
		cmdResult := utils.Exec("docker-compose", cmdArgs)
		isRunning := false
		if cmdResult != "" {
			isRunning = true
		}
		edgeAppServices = append(edgeAppServices, EdgeAppService{ID: serviceID, IsRunning: isRunning})
	}

	return edgeAppServices

}

// RunEdgeApp : Run an EdgeApp and return its most current status
func RunEdgeApp(ID string) EdgeAppStatus {

	cmdArgs := []string{"-f", utils.GetPath("wsPath") + "/docker-compose.yml", "up", ID}
	utils.Exec("docker-compose", cmdArgs)

	return GetEdgeAppStatus(ID)

}

// StopEdgeApp : Stops an EdgeApp and return its most current status
func StopEdgeApp(ID string) EdgeAppStatus {

	cmdArgs := []string{"-f", utils.GetPath("wsPath") + "/docker-compose.yml", "down", ID}
	utils.Exec("docker-compose", cmdArgs)

	return GetEdgeAppStatus(ID)

}
