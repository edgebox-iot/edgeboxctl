package edgeapps

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/edgebox-iot/edgeboxctl/internal/system"
	"github.com/edgebox-iot/edgeboxctl/internal/utils"
)

// EdgeApp : Struct representing an EdgeApp in the system
type EdgeApp struct {
	ID                 string           `json:"id"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	Status             EdgeAppStatus    `json:"status"`
	Services           []EdgeAppService `json:"services"`
	InternetAccessible bool             `json:"internet_accessible"`
	NetworkURL         string           `json:"network_url"`
	InternetURL        string           `json:"internet_url"`
}

// MaybeEdgeApp : Boolean flag for validation of edgeapp existance
type MaybeEdgeApp struct {
	EdgeApp EdgeApp `json:"edge_app"`
	Valid   bool    `json:"valid"`
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
const runnableFilename = "/.run"
const myEdgeAppServiceEnvFilename = "/myedgeapp.env"
const defaultContainerOperationSleepTime time.Duration = time.Second * 10

// GetEdgeApp : Returns a EdgeApp struct with the current application information
func GetEdgeApp(ID string) MaybeEdgeApp {

	result := MaybeEdgeApp{
		EdgeApp: EdgeApp{},
		Valid:   false,
	}

	_, err := os.Stat(utils.GetPath(utils.EdgeAppsPath) + ID + configFilename)
	if !os.IsNotExist(err) {
		// File exists. Start digging!

		edgeAppName := ID
		edgeAppDescription := ""

		edgeAppEnv, err := godotenv.Read(utils.GetPath(utils.EdgeAppsPath) + ID + envFilename)

		if err != nil {
			log.Println("Error loading .env file for edgeapp " + edgeAppName)
		} else {
			if edgeAppEnv["EDGEAPP_NAME"] != "" {
				edgeAppName = edgeAppEnv["EDGEAPP_NAME"]
			}
			if edgeAppEnv["EDGEAPP_DESCRIPTION"] != "" {
				edgeAppDescription = edgeAppEnv["EDGEAPP_DESCRIPTION"]
			}
		}

		edgeAppInternetAccessible := false
		edgeAppInternetURL := ""

		myEdgeAppServiceEnv, err := godotenv.Read(utils.GetPath(utils.EdgeAppsPath) + ID + myEdgeAppServiceEnvFilename)
		if err != nil {
			log.Println("No myedge.app environment file found. Status is Network-Only")
		} else {
			if myEdgeAppServiceEnv["INTERNET_URL"] != "" {
				edgeAppInternetAccessible = true
				edgeAppInternetURL = myEdgeAppServiceEnv["INTERNET_URL"]
			}
		}

		result = MaybeEdgeApp{
			EdgeApp: EdgeApp{
				ID:                 ID,
				Name:               edgeAppName,
				Description:        edgeAppDescription,
				Status:             GetEdgeAppStatus(ID),
				Services:           GetEdgeAppServices(ID),
				InternetAccessible: edgeAppInternetAccessible,
				NetworkURL:         ID + "." + system.GetHostname() + ".local",
				InternetURL:        edgeAppInternetURL,
			},
			Valid: true,
		}

	}

	return result

}

func IsEdgeAppInstalled(ID string) bool {

	result := false

	_, err := os.Stat(utils.GetPath(utils.EdgeAppsPath) + ID + runnableFilename)
	if !os.IsNotExist(err) {
		result = true
	}

	return result

}

func SetEdgeAppInstalled(ID string) bool {

	result := true
	edgeAppPath := utils.GetPath(utils.EdgeAppsPath)

	_, err := os.Stat(edgeAppPath + ID + runnableFilename)
	if os.IsNotExist(err) {

		_, err := os.Create(edgeAppPath + ID + runnableFilename)
		result = true

		if err != nil {
			log.Fatal("Runnable file for EdgeApp could not be created!")
			result = false
		}

		buildFrameworkContainers()

	} else {

		// Is already installed.
		result = false

	}

	return result

}

func SetEdgeAppNotInstalled(ID string) bool {

	result := true
	err := os.Remove(utils.GetPath(utils.EdgeAppsPath) + ID + runnableFilename)
	if err != nil {
		result = false
		log.Fatal(err)
	}

	buildFrameworkContainers()

	return result

}

// GetEdgeApps : Returns a list of all available EdgeApps in structs filled with information
func GetEdgeApps() []EdgeApp {

	var edgeApps []EdgeApp

	// Building list of available edgeapps in the system with their status

	files, err := ioutil.ReadDir(utils.GetPath(utils.EdgeAppsPath))
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if f.IsDir() {
			// It is a folder that most probably contains an EdgeApp.
			// To be fully sure, test that edgebox-compose.yml file exists in the target directory.
			maybeEdgeApp := GetEdgeApp(f.Name())
			if maybeEdgeApp.Valid {

				edgeApp := maybeEdgeApp.EdgeApp
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

	if !IsEdgeAppInstalled(ID) {

		status = EdgeAppStatus{-1, "not-installed"}

	} else {

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

	}

	return status

}

// GetEdgeAppServices : Returns a
func GetEdgeAppServices(ID string) []EdgeAppService {
	wsPath := utils.GetPath(utils.WsPath)
	cmdArgs := []string{"-r", ".services | keys[]", utils.GetPath(utils.EdgeAppsPath) + ID + configFilename}
	servicesString := utils.Exec(utils.GetPath(utils.WsPath), "yq", cmdArgs)
	serviceSlices := strings.Split(servicesString, "\n")
	serviceSlices = utils.DeleteEmptySlices(serviceSlices)
	var edgeAppServices []EdgeAppService

	for _, serviceID := range serviceSlices {
		cmdArgs = []string{"-f", wsPath + "/docker-compose.yml", "exec", "-T", serviceID, "echo", "'Service Check'"}
		cmdResult := utils.Exec(wsPath, "docker-compose", cmdArgs)
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
	wsPath := utils.GetPath(utils.WsPath)
	services := GetEdgeAppServices(ID)
	cmdArgs := []string{}

	for _, service := range services {

		cmdArgs = []string{"-f", wsPath + "/docker-compose.yml", "start", service.ID}
		utils.Exec(wsPath, "docker-compose", cmdArgs)
	}

	// Wait for it to settle up before continuing...
	time.Sleep(defaultContainerOperationSleepTime)

	return GetEdgeAppStatus(ID)

}

// StopEdgeApp : Stops an EdgeApp and return its most current status
func StopEdgeApp(ID string) EdgeAppStatus {
	wsPath := utils.GetPath(utils.WsPath)
	services := GetEdgeAppServices(ID)
	cmdArgs := []string{}
	for _, service := range services {

		cmdArgs = []string{"-f", wsPath + "/docker-compose.yml", "stop", service.ID}
		utils.Exec(wsPath, "docker-compose", cmdArgs)
	}

	// Wait for it to settle up before continuing...
	time.Sleep(defaultContainerOperationSleepTime)

	return GetEdgeAppStatus(ID)

}

// StopAllEdgeApps: Stops all EdgeApps and returns a count of how many were stopped
func StopAllEdgeApps() int {
	edgeApps := GetEdgeApps()
	appCount := 0
	for _, edgeApp := range edgeApps {
		StopEdgeApp(edgeApp.ID)
		appCount++
	}

	return appCount

}

// StartAllEdgeApps: Starts all EdgeApps and returns a count of how many were started
func StartAllEdgeApps() int {	
	edgeApps := GetEdgeApps()
	appCount := 0
	for _, edgeApp := range edgeApps {
		RunEdgeApp(edgeApp.ID)
		appCount++
	}

	return appCount

}

func RestartEdgeAppsService() {
	buildFrameworkContainers()
}

// EnableOnline : Write environment file and rebuild the necessary containers. Rebuilds containers in project (in case of change only)
func EnableOnline(ID string, InternetURL string) MaybeEdgeApp {

	maybeEdgeApp := GetEdgeApp(ID)
	if maybeEdgeApp.Valid { // We're only going to do this operation if the EdgeApp actually exists.
		// Create the myedgeapp.env file and add the InternetURL entry to it
		envFilePath := utils.GetPath(utils.EdgeAppsPath) + ID + myEdgeAppServiceEnvFilename
		env, _ := godotenv.Unmarshal("INTERNET_URL=" + InternetURL)
		_ = godotenv.Write(env, envFilePath)
	}

	buildFrameworkContainers()

	return GetEdgeApp(ID) // Return refreshed information

}

// DisableOnline : Removes env files necessary for system external access config. Rebuilds containers in project (in case of change only).
func DisableOnline(ID string) MaybeEdgeApp {

	envFilePath := utils.GetPath(utils.EdgeAppsPath) + ID + myEdgeAppServiceEnvFilename
	_, err := godotenv.Read(envFilePath)
	if err != nil {
		log.Println("myedge.app environment file for " + ID + " not found. No need to delete.")
	} else {
		cmdArgs := []string{envFilePath}
		utils.Exec(utils.GetPath(utils.WsPath), "rm", cmdArgs)
	}

	buildFrameworkContainers()

	return GetEdgeApp(ID)

}

func EnablePublicDashboard(InternetURL string) bool {

	envFilePath := utils.GetPath(utils.ApiPath) + myEdgeAppServiceEnvFilename
	env, _ := godotenv.Unmarshal("INTERNET_URL=" + InternetURL)
	_ = godotenv.Write(env, envFilePath)

	buildFrameworkContainers()

	return true

}

func DisablePublicDashboard() bool {
	envFilePath := utils.GetPath(utils.ApiPath) + myEdgeAppServiceEnvFilename
	if !IsPublicDashboard() {
		log.Println("myedge.app environment file for the dashboard / api not found. No need to delete.")
		return false
	}

	cmdArgs := []string{envFilePath}
	utils.Exec(utils.GetPath(utils.ApiPath), "rm", cmdArgs)
	buildFrameworkContainers()
	return true
}

func IsPublicDashboard() bool {
	envFilePath := utils.GetPath(utils.ApiPath) + myEdgeAppServiceEnvFilename
	_, err := godotenv.Read(envFilePath)
	return err == nil
}

func buildFrameworkContainers() {

	wsPath := utils.GetPath(utils.WsPath)
	cmdArgs := []string{wsPath + "ws", "--build"}
	utils.ExecAndStream(wsPath, "sh", cmdArgs)

	time.Sleep(defaultContainerOperationSleepTime)

}
