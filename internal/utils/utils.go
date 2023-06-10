package utils

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ExecAndStream : Runs a terminal command, but streams progress instead of outputting. Ideal for long lived process that need to be logged.
func ExecAndStream(path string, command string, args []string) string {

	cmd := exec.Command(command, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Dir = path

	err := cmd.Run()

	outStr, errStr := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())

	returnVal := outStr
	if err != nil {
		fmt.Printf("cmd.Run() failed with %s\n", err)
		returnVal = errStr
	}

	fmt.Printf("\nout:\n%s\nerr:\n%s\n", outStr, errStr)

	return returnVal
}

// Exec : Runs a terminal Command, catches and logs errors, returns the result.
func Exec(path string, command string, args []string) string {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Dir = path
	err := cmd.Run()
	if err != nil {
		// TODO: Deal with possibility of error in command, allow explicit error handling and return proper formatted stderr
		// log.Println(fmt.Sprint(err) + ": " + stderr.String()) // ... Silence...
	}

	// log.Println("Result: " + out.String()) // ... Silence ...

	return strings.Trim(out.String(), " \n")

}

// Exec : Runs a terminal Command, returns the result as a *bufio.Scanner type, split in lines and ready to parse.
func ExecAndGetLines(path string, command string, args []string) *bufio.Scanner {
	cmdOutput := Exec(path, command, args)
	cmdOutputReader := strings.NewReader(cmdOutput)
	scanner := bufio.NewScanner(cmdOutputReader)
	scanner.Split(bufio.ScanLines)

	return scanner
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

// GetSQLiteDbConnectionDetails : Returns the necessary string as connection info for SQL.db()
func GetSQLiteDbConnectionDetails() string {

	var apiEnv map[string]string
	apiEnv, err := godotenv.Read(GetPath(ApiEnvFileLocation))

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	return apiEnv["SQLITE_DATABASE"] // Will read from api project edgebox.env file

}

// GetSQLiteFormattedDateTime: Given a Time, Returns a string that is formatted ready to be inserted into an SQLite Datetime field using sql.Prepare.
func GetSQLiteFormattedDateTime(t time.Time) string {
	// This date is used to indicate the layout.
	const datetimeLayout = "2006-01-02 15:04:05"
	formatedDatetime := t.Format(datetimeLayout)

	return formatedDatetime
}

const BackupPasswordFileLocation string = "backupPasswordFileLocation"
const CloudEnvFileLocation string = "cloudEnvFileLocation"
const ApiEnvFileLocation string = "apiEnvFileLocation"
const ApiPath string = "apiPath"
const EdgeAppsPath string = "edgeAppsPath"
const EdgeAppsBackupPath string = "edgeAppsBackupPath"
const WsPath string = "wsPath"

// GetPath : Returns either the hardcoded path, or a overwritten value via .env file at project root. Register paths here for seamless working code between dev and prod environments ;)
func GetPath(pathKey string) string {

	// Read whole of .env file to map.
	var env map[string]string
	env, err := godotenv.Read()
	var targetPath string

	if err != nil {
		targetPath = ""
	}

	switch pathKey {
	case CloudEnvFileLocation:

		if env["CLOUD_ENV_FILE_LOCATION"] != "" {
			targetPath = env["CLOUD_ENV_FILE_LOCATION"]
		} else {
			targetPath = "/home/system/components/edgeboxctl/cloud.env"
		}

	case ApiEnvFileLocation:

		if env["API_ENV_FILE_LOCATION"] != "" {
			targetPath = env["API_ENV_FILE_LOCATION"]
		} else {
			targetPath = "/home/system/components/api/edgebox.env"
		}

	case ApiPath:

		if env["API_PATH"] != "" {
			targetPath = env["API_PATH"]
		} else {
			targetPath = "/home/system/components/api/"
		}

	case EdgeAppsPath:

		if env["EDGEAPPS_PATH"] != "" {
			targetPath = env["EDGEAPPS_PATH"]
		} else {
			targetPath = "/home/system/components/apps/"
		}

	case EdgeAppsBackupPath:
		if env["EDGEAPPS_BACKUP_PATH"] != "" {
			targetPath = env["EDGEAPPS_BACKUP_PATH"]
		} else {
			targetPath = "/home/system/components/backups/"
		}

	case WsPath:

		if env["WS_PATH"] != "" {
			targetPath = env["WS_PATH"]
		} else {
			targetPath = "/home/system/components/ws/"
		}

	case BackupPasswordFileLocation:

		if env["BACKUP_PASSWORD_FILE_LOCATION"] != "" {
			targetPath = env["BACKUP_PASSWORD_FILE_LOCATION"]
		} else {
			targetPath = "/home/system/components/backups/pw.txt"
		}

	default:

		log.Printf("path_key %s nonexistant in GetPath().\n", pathKey)

	}

	return targetPath

}

// WriteOption : Writes a key value pair option into the api shared database
func WriteOption(optionKey string, optionValue string) {

	db, err := sql.Open("sqlite3", GetSQLiteDbConnectionDetails())

	if err != nil {
		log.Fatal(err.Error())
	}

	statement, err := db.Prepare("REPLACE into option (name, value, created, updated) VALUES (?, ?, ?, ?);") // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	formatedDatetime := GetSQLiteFormattedDateTime(time.Now())

	_, err = statement.Exec(optionKey, optionValue, formatedDatetime, formatedDatetime) // Execute SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	db.Close()
}

// ReadOption : Reads a key value pair option from the api shared database
func ReadOption(optionKey string) string {
	
	db, err := sql.Open("sqlite3", GetSQLiteDbConnectionDetails())

	if err != nil {
		log.Fatal(err.Error())
	}

	var optionValue string

	err = db.QueryRow("SELECT value FROM option WHERE name = ?", optionKey).Scan(&optionValue)

	if err != nil {
		log.Println(err.Error())
	}

	db.Close()

	return optionValue
}

// DeleteOption : Deletes a key value pair option from the api shared database
func DeleteOption(optionKey string) {
	
	db, err := sql.Open("sqlite3", GetSQLiteDbConnectionDetails())

	if err != nil {
		log.Fatal(err.Error())
	}

	statement, err := db.Prepare("DELETE FROM option WHERE name = ?;") // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = statement.Exec(optionKey) // Execute SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}

	db.Close()
}
