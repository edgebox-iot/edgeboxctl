package utils

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
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
