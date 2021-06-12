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
	}

}

func TestGetPath(t *testing.T) {
	invalidPathKey := "test"

	result := GetPath(invalidPathKey)
	if result != "" {
		t.Log("Expected empty result but got ", result)
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
