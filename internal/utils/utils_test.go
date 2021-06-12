// +build unit

package utils

import (
	"testing"
	"time"
)

func TestExec(t *testing.T) {
	testCommand := "echo"
	testArguments := []string{"Hello World"}

	result := Exec("/", testCommand, testArguments)

	if result != "Hello World" {
		t.Log("Expected 'Hello World' but got", "'"+result+"'")
		t.Fail()
	}
}

func ExampleExecAndStream() {
	ExecAndStream("/", "echo", []string{"Hello"})
	// Output:
	// Hello
	//
	// out:
	// Hello
	//
	// err:
}

func ExampleExecAndStreamExecutableNotFound() {
	ExecAndStream("/", "testcommand", []string{"Hello"})
	// Output:
	// cmd.Run() failed with exec: "testcommand": executable file not found in $PATH
	//
	// out:
	//
	// err:
}

func ExampleExecAndStreamError() {
	ExecAndStream("/", "man", []string{"Hello"})
	// Output:
	// cmd.Run() failed with exit status 16
	//
	// out:
	//
	// err:
	// No manual entry for Hello
}

func TestExecAndGetLines(t *testing.T) {
	testCommand := "echo"
	testArguments := []string{"$'Line1\nLine2\nLine3'"}
	var result []string

	scanner := ExecAndGetLines("/", testCommand, testArguments)

	for scanner.Scan() {
		result = append(result, scanner.Text())
	}

	if len(result) != 3 {
		t.Log("Expected 3 lines but got ", len(result))
		t.Fail()
	}
}

func TestDeleteEmptySlices(t *testing.T) {
	testSlice := []string{"Line1", "", "Line3"}

	resultSlice := DeleteEmptySlices(testSlice)

	if (len(resultSlice)) != 2 {
		t.Log("Expected 2 slices but got ", len(resultSlice))
		t.Fail()
	}

}

func TestGetPath(t *testing.T) {
	invalidPathKey := "test"

	result := GetPath(invalidPathKey)
	if result != "" {
		t.Log("Expected empty result but got ", result)
		t.Fail()
	}

	validPathKey := "wsPath"
	result = GetPath(validPathKey)
	if result != "/home/system/components/ws/" {
		t.Log("Expected /home/system/components/ws/ but got", result)
		t.Fail()
	}
}

func TestGetSQLiteFormattedDateTime(t *testing.T) {
	datetime := time.Date(2021, time.Month(1), 01, 1, 30, 15, 0, time.UTC)
	result := GetSQLiteFormattedDateTime(datetime)

	if result != "2021-01-01 01:30:15" {
		t.Log("Expected 2021-01-01 01:30:15 but got ", result)
		t.Fail()
	}
}
