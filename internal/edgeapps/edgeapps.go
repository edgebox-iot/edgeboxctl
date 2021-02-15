package edgeapps

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

// EdgeApp : Struct representing an EdgeApp in the system
type EdgeApp struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Status             EdgeAppStatus `json:"status"`
	InternetAccessible bool          `json:"internet_accessible"`
	InternetURL        string        `json:"internet_url"`
	NetworkURL         string        `json:"network_url"`
}

// EdgeAppStatus : Struct representing possible EdgeApp statuses (code + description)
type EdgeAppStatus struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// GetEdgeApps : Returns a list of EdgeApp struct filled with information
func GetEdgeApps() []EdgeApp {

	var edgeApps []EdgeApp

	// Building list of available edgeapps in the system.
	configFilename := "edgebox-compose.yml"
	// envFilename := "edgebox.env"
	// postinstallFilename := "edgebox-postinstall.txt"
	edgeAppsPath := "/home/system/components/apps"
	var edgeAppsList []string

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
				edgeAppsList = append(edgeAppsList, f.Name())
			}

		}
	}

	// Querying to see which apps are running.
	// cmdargs = []string{"ps", "-a"}
	// executeCommand("docker", cmdargs)
	// (...)

	return edgeApps

}
