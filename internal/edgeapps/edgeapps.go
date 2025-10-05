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
	"github.com/edgebox-iot/edgeboxctl/internal/diagnostics"
)

// EdgeApp : Struct representing an EdgeApp in the system
type EdgeApp struct {
	ID                 string           `json:"id"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	Experimental	   bool             `json:"experimental"`
	Status             EdgeAppStatus    `json:"status"`
	Services           []EdgeAppService `json:"services"`
	InternetAccessible bool             `json:"internet_accessible"`
	NetworkURL         string           `json:"network_url"`
	InternetURL        string           `json:"internet_url"`
	Options			   []EdgeAppOption  `json:"options"`
	NeedsConfig		   bool             `json:"needs_config"`
	Login              EdgeAppLogin	 	`json:"login"`
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

type EdgeAppOption struct {
	Key             string `json:"key"`
	Value           string `json:"value"`
	DefaultValue    string `json:"default_value"`
	Format          string `json:"format"`
	Description     string `json:"description"`
	IsSecret        bool   `json:"is_secret"`
	IsInstallLocked bool   `json:"is_install_locked"`
}

type EdgeAppLogin struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}

const configFilename = "/edgebox-compose.yml"
const envFilename = "/edgebox.env"
const optionsTemplateFilename = "/edgeapp.template.env"
const optionsEnvFilename = "/edgeapp.env"
const authEnvFilename = "/auth.env"
const runnableFilename = "/.run"
const appdataFoldername = "/appdata"
const postInstallFilename = "/edgebox-postinstall.done"
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
		edgeAppExperimental := false
		edgeAppOptions := []EdgeAppOption{}

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
			if edgeAppEnv["EDGEAPP_EXPERIMENTAL"] == "true" {
				edgeAppExperimental = true
			}
		}

		needsConfig := false
		hasFilledOptions := false
		edgeAppOptionsTemplate, err := godotenv.Read(utils.GetPath(utils.EdgeAppsPath) + ID + optionsTemplateFilename)
		if err != nil {
			log.Println("Error loading options template file for edgeapp " + edgeAppName)
		} else {
			// Try to read the edgeAppOptionsEnv file
			edgeAppOptionsEnv, err := godotenv.Read(utils.GetPath(utils.EdgeAppsPath) + ID + optionsEnvFilename)
			if err != nil {
				log.Println("Error loading options env file for edgeapp " + edgeAppName)
			} else {
				hasFilledOptions = true
			}

			for key, value := range edgeAppOptionsTemplate {
				
				optionFilledValue := ""
				if hasFilledOptions {
					// Check if key exists in edgeAppOptionsEnv
					optionFilledValue = edgeAppOptionsEnv[key]
				}

				format := ""
				defaultValue := ""
				description := ""
				installLocked := false

				// Parse value to separate by | and get the format, installLocked, description and default value
				// Format is the first element
				// InstallLocked is the second element
				// Description is the third element
				// Default value is the fourth element

				valueSlices := strings.Split(value, "|")
				if len(valueSlices) > 0 {
					format = valueSlices[0]
				}
				if len(valueSlices) > 1 {
					installLocked = valueSlices[1] == "true"
				}
				if len(valueSlices) > 2 {
					description = valueSlices[2]
				}
				if len(valueSlices) > 3 {
					defaultValue = valueSlices[3]
				}

				// // If value contains ">|", then get everything that is to the right of it as the description
				// // and get everything between "<>" as the format
				// if strings.Contains(value, ">|") {
				// 	description = strings.Split(value, ">|")[1]
				// 	// Check if description has default value. That would be everything that is to the right of the last "|"
				// 	if strings.Contains(description, "|") {
				// 		defaultValue = strings.Split(description, "|")[1]
				// 		description = strings.Split(description, "|")[0]
				// 	}

				// 	value = strings.Split(value, ">|")[0]
				// 	// Remove the initial < from value
				// 	value = strings.TrimPrefix(value, "<")
				// } else {
				// 	// Trim initial < and final > from value
				// 	value = strings.TrimPrefix(value, "<")
				// 	value = strings.TrimSuffix(value, ">")
				// }

				isSecret := false

				// Check if the lowercased key string contains the letters "pass", "secret", "key"
				lowercaseKey := strings.ToLower(key)
				// check if lowercaseInput contains "pass", "key", or "secret", or "token"
				if strings.Contains(lowercaseKey, "pass") ||
					strings.Contains(lowercaseKey, "key") ||
					strings.Contains(lowercaseKey, "secret") ||
					strings.Contains(lowercaseKey, "token") {
					isSecret = true
				}

				currentOption := EdgeAppOption{
					Key:             key,
					Value:           optionFilledValue,
					DefaultValue:    defaultValue,
					Description:     description,
					Format:          format,
					IsSecret:        isSecret,
					IsInstallLocked: installLocked,
				}
				edgeAppOptions = append(edgeAppOptions, currentOption)

				if optionFilledValue == "" {
					needsConfig = true
				}

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

		edgeAppBasicAuthEnabled := false
		edgeAppBasicAuthUsername := ""
		edgeAppBasicAuthPassword := ""

		edgeAppAuthEnv, err := godotenv.Read(utils.GetPath(utils.EdgeAppsPath) + ID + authEnvFilename)
		if err != nil {
			log.Println("No auth.env file found. Login status is disabled.")
		} else {
			if edgeAppAuthEnv["USERNAME"] != "" && edgeAppAuthEnv["PASSWORD"] != "" {
				edgeAppBasicAuthEnabled = true
				edgeAppBasicAuthUsername = edgeAppAuthEnv["USERNAME"]
				edgeAppBasicAuthPassword = edgeAppAuthEnv["PASSWORD"]
			}
		}

		result = MaybeEdgeApp{
			EdgeApp: EdgeApp{
				ID:                 ID,
				Name:               edgeAppName,
				Description:        edgeAppDescription,
				Experimental:       edgeAppExperimental,
				Status:             GetEdgeAppStatus(ID),
				Services:           GetEdgeAppServices(ID),
				InternetAccessible: edgeAppInternetAccessible,
				NetworkURL:         ID + "." + system.GetHostname() + ".local",
				InternetURL:        edgeAppInternetURL,
				Options: 		    edgeAppOptions,
				NeedsConfig:        needsConfig,
				Login:				EdgeAppLogin{edgeAppBasicAuthEnabled, edgeAppBasicAuthUsername, edgeAppBasicAuthPassword},
				
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

func writeAppRunnableFiles(ID string) bool {
	edgeAppPath := utils.GetPath(utils.EdgeAppsPath)
	_, err := os.Stat(edgeAppPath + ID + runnableFilename)
	if os.IsNotExist(err) {
		_, err := os.Create(edgeAppPath + ID + runnableFilename)
		if err != nil {
			log.Fatal("Runnable file for EdgeApp could not be created!")
			return false
		}

		// Check the block default apps option
        blockDefaultAppsOption := utils.ReadOption("DASHBOARD_BLOCK_DEFAULT_APPS_PUBLIC_ACCESS")
        if blockDefaultAppsOption != "yes" {
            // Create myedgeapp.env file with default network URL
            envFilePath := edgeAppPath + ID + myEdgeAppServiceEnvFilename
            
			var networkURL string
			domainName := utils.ReadOption("DOMAIN_NAME")

			if domainName != "" {
				networkURL = ID + "." + domainName
			} else if diagnostics.GetReleaseVersion() == diagnostics.CLOUD_VERSION {
				cluster := utils.ReadOption("CLUSTER") 
				username := utils.ReadOption("USERNAME")
				if cluster != "" && username != "" {
					networkURL = username + "-" + ID + "." + cluster
				}
			} else {
				networkURL = ID + "." + system.GetHostname() + ".local" // default 
			}
			
            env, _ := godotenv.Unmarshal("INTERNET_URL=" + networkURL)
            err = godotenv.Write(env, envFilePath)
            if err != nil {
                log.Printf("Error creating myedgeapp.env file: %s", err)
                // result = false
            }
        }
	}
	return true
}

func SetEdgeAppInstalled(ID string) bool {

	result := true

	if writeAppRunnableFiles(ID) {
		
		buildFrameworkContainers()

	} else {

		// Is already installed.
		result = false

	}

	return result

}

func SetEdgeAppBulkInstalled(IDs []string) bool {

	result := true

	for _, ID := range IDs {
		writeAppRunnableFiles(ID)
	}

	buildFrameworkContainers()

	return result

}


func SetEdgeAppNotInstalled(ID string) bool {

	// Stop the app first
	StopEdgeApp(ID)

	// Now remove any files
	result := true
	
	err := os.Remove(utils.GetPath(utils.EdgeAppsPath) + ID + runnableFilename)
	if err != nil {
		result = false
		log.Println(err)
	}

	err = os.Remove(utils.GetPath(utils.EdgeAppsPath) + ID + authEnvFilename)
	if err != nil {
		result = false
		log.Println(err)
	}

	err = os.RemoveAll(utils.GetPath(utils.EdgeAppsPath) + ID + appdataFoldername)
	if err != nil {
		result = false
		log.Println(err)
	}

	err = os.Remove(utils.GetPath(utils.EdgeAppsPath) + ID + myEdgeAppServiceEnvFilename)
	if err != nil {
		result = false
		log.Println(err)
	}

	err = os.Remove(utils.GetPath(utils.EdgeAppsPath) + ID + optionsEnvFilename)
	if err != nil {
		result = false
		log.Println(err)
	}

	err = os.Remove(utils.GetPath(utils.EdgeAppsPath) + ID + postInstallFilename)
	if err != nil {
		result = false
		log.Println(err)
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
		shouldBeRunning := false
		isRunning := false

		// Is service "runnable" when .run lockfile in the app folder
		_, err := os.Stat(utils.GetPath(utils.EdgeAppsPath) + ID + runnableFilename)
		if !os.IsNotExist(err) {
			shouldBeRunning = true
		}

		// Check if the service is actually running
		if shouldBeRunning {
			cmdArgs = []string{"-f", wsPath + "/docker-compose.yml", "exec", "-T", serviceID, "echo", "'Service Check'"}
			cmdResult := utils.Exec(wsPath, "docker", append([]string{"compose"}, cmdArgs...))
			if cmdResult != "" {
				isRunning = true
			}
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
		utils.Exec(wsPath, "docker", append([]string{"compose"}, cmdArgs...))
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
		utils.Exec(wsPath, "docker", append([]string{"compose"}, cmdArgs...))
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
