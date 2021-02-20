package utils

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"

	"github.com/joho/godotenv"
)

// Exec : Runs a terminal Command, catches and logs errors, returns the result.
func Exec(command string, args []string) string {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Dir = GetPath("wsPath")
	err := cmd.Run()
	if err != nil {
		// TODO: Deal with possibility of error in command, allow explicit error handling and return proper formatted stderr
		log.Println(fmt.Sprint(err) + ": " + stderr.String()) // ... Silence...
	}

	log.Println("Result: " + out.String()) // ... Silence ...

	return out.String()

}

// DeleteEmptySlices : Given a string array, delete empty entries.
func DeleteEmptySlices(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

// GetMySQLDbConnectionDetails : Returns the necessary string as connection info for SQL.db()
func GetMySQLDbConnectionDetails() string {

	var apiEnv map[string]string
	apiEnv, err := godotenv.Read(GetPath("apiEnvFileLocation"))

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	Dbhost := "127.0.0.1:" + apiEnv["HOST_MACHINE_MYSQL_PORT"]
	Dbname := apiEnv["MYSQL_DATABASE"]
	Dbuser := apiEnv["MYSQL_USER"]
	Dbpass := apiEnv["MYSQL_PASSWORD"]

	return Dbuser + ":" + Dbpass + "@tcp(" + Dbhost + ")/" + Dbname

}

// GetPath : Returns either the hardcoded path, or a overwritten value via .env file at project root. Register paths here for seamless working code between dev and prod environments ;)
func GetPath(pathKey string) string {

	// Read whole of .env file to map.
	var env map[string]string
	env, err := godotenv.Read()
	targetPath := ""

	if err != nil {
		// log.Println("Project .env file not found withing project root. Using only hardcoded path variables.")
		// Do Nothing...
	}

	switch pathKey {
	case "apiEnvFileLocation":

		if env["API_ENV_FILE_LOCATION"] != "" {
			targetPath = env["API_ENV_FILE_LOCATION"]
		} else {
			targetPath = "/home/system/components/api/edgebox.env"
		}

	case "edgeAppsPath":

		if env["EDGEAPPS_PATH"] != "" {
			targetPath = env["EDGEAPPS_PATH"]
		} else {
			targetPath = "/home/system/components/apps/"
		}

	case "wsPath":

		if env["WS_PATH"] != "" {
			targetPath = env["WS_PATH"]
		} else {
			targetPath = "/home/system/components/ws/"
		}

	default:

		log.Printf("path_key %s nonexistant in GetPath().\n", pathKey)

	}

	return targetPath

}
