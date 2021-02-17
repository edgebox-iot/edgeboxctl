package edgeapps

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

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
const edgeAppsPath = "/home/jpt/Repositories/edgebox/apps/"
const wsPath = "/home/jpt/Repositories/edgebox/ws/"

// GetEdgeApps : Returns a list of EdgeApp struct filled with information
func GetEdgeApps() []EdgeApp {

	var edgeApps []EdgeApp

	// Building list of available edgeapps in the system with their status

	files, err := ioutil.ReadDir(edgeAppsPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if f.IsDir() {
			// It is a folder that most probably contains an EdgeApp.
			// To be fully sure, test that edgebox-compose.yml file exists in the target directory.
			_, err := os.Stat(edgeAppsPath + f.Name() + configFilename)
			if !os.IsNotExist(err) {
				// File exists. Start digging!
				edgeApp := EdgeApp{ID: f.Name(), Status: GetEdgeAppStatus(f.Name()), Services: GetEdgeAppServices(f.Name()), NetworkURL: f.Name() + ".edgebox.local"}
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

	log.Println("Finding " + ID + " EdgeApp Services")

	// strConfigFile := string(configFile) // convert content to a 'string'

	cmdArgs := []string{"-r", ".services | keys[]", edgeAppsPath + ID + configFilename}
	servicesString := utils.Exec("yq", cmdArgs)
	serviceSlices := strings.Split(servicesString, "\n")
	serviceSlices = utils.DeleteEmptySlices(serviceSlices)
	var edgeAppServices []EdgeAppService

	for _, serviceID := range serviceSlices {
		log.Println(serviceID)
		cmdArgs = []string{"-f", wsPath + "/docker-compose.yml", "ps", "-q", serviceID}
		cmdResult := utils.Exec("docker-compose", cmdArgs)
		isRunning := false
		if cmdResult != "" {
			isRunning = true
		}
		edgeAppServices = append(edgeAppServices, EdgeAppService{ID: serviceID, IsRunning: isRunning})
	}

	return edgeAppServices

}
