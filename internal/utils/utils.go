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
	err := cmd.Run()
	if err != nil {
		log.Println(fmt.Sprint(err) + ": " + stderr.String())
	}

	log.Println("Result: " + out.String())

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

	// const apiEnvFileLocation = "/home/system/components/api/edgebox.env"
	const apiEnvFileLocation = "/home/jpt/Repositories/edgebox/api/edgebox.env"

	var apiEnv map[string]string
	apiEnv, err := godotenv.Read(apiEnvFileLocation)

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	Dbhost := "127.0.0.1:" + apiEnv["HOST_MACHINE_MYSQL_PORT"]
	Dbname := apiEnv["MYSQL_DATABASE"]
	Dbuser := apiEnv["MYSQL_USER"]
	Dbpass := apiEnv["MYSQL_PASSWORD"]

	return Dbuser + ":" + Dbpass + "@tcp(" + Dbhost + ")/" + Dbname

}
