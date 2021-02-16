package edgeapps

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

// EdgeApp : Struct representing an EdgeApp in the system
type EdgeApp struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Status     EdgeAppStatus    `json:"status"`
	Services   []EdgeAppService `json:"services"`
	NetworkURL []string         `json:"network_url"`
}

// EdgeAppStatus : Struct representing possible EdgeApp statuses (code + description)
type EdgeAppStatus struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// EdgeAppService : Struct representing a single container that can be part of an EdgeApp package
type EdgeAppService struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Isrunning bool   `json:"is_running"`
}

// GetEdgeApps : Returns a list of EdgeApp struct filled with information
func GetEdgeApps() string { // []EdgeApp {

	// var edgeApps []EdgeApp

	// Building list of available edgeapps in the system.
	configFilename := "edgebox-compose.yml"
	edgeAppsPath := "/home/system/components/apps"

	files, err := ioutil.ReadDir(edgeAppsPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		fmt.Println(f.Name())
		if f.IsDir() {
			// It is a folder that most probably contains an EdgeApp.
			// To be fully sure, test that edgebox-compose.yml file exists in the target directory.
			_, err := os.Stat("/home/system/components/apps/" + f.Name() + "/" + configFilename)
			if !os.IsNotExist(err) {
				// File exists. Start digging!
				// edgeApp := EdgeApp{ID: f.Name(), Status: GetEdgeAppStatus(f.Name())}
				// edgeApps = append(edgeApps, edgeApp)
				GetEdgeAppServices(f.Name())
			}

		}
	}

	// Querying to see which apps are running.
	// cmdargs = []string{"ps", "-a"}
	// executeCommand("docker", cmdargs)
	// (...)

	// return edgeApps
	return "OK"
}

// GetEdgeAppStatus : Returns a struct representing the current status of the EdgeApp
// func GetEdgeAppStatus(ID string) EdgeAppStatus {

// 	// Possible states of an EdgeApp:
// 	// - All services running = EdgeApp running
// 	// - Some services running = Problem detected, needs restart
// 	// - No service running = EdgeApp is off

// 	services := GetEdgeAppServices(ID)

// 	return status
// }

// GetEdgeAppServices : Returns a
func GetEdgeAppServices(ID string) string {

	data, err := ioutil.ReadFile("/home/system/components/apps/" + ID + "/edgebox-compose.yml")

	// If this happens it means that no EdgeApp exists for the given ID. This func should not be called in that case.
	if err != nil {
		log.Fatal(err)
	}

	// Is application running?

	t := make(map[string]interface{})
	yaml.Unmarshal([]byte(data), &t)
	fmt.Println(t["services"])

	return "OK"

}
