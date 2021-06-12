package utils

import (
	"bufio"
	"bytes"
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
func ExecAndStream(path string, command string, args []string) {

	cmd := exec.Command(command, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Dir = path

	err := cmd.Run()

	if err != nil {
		fmt.Printf("cmd.Run() failed with %s\n", err)
	}

	outStr, errStr := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())
	fmt.Printf("\nout:\n%s\nerr:\n%s\n", outStr, errStr)

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
	apiEnv, err := godotenv.Read(GetPath("apiEnvFileLocation"))

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
