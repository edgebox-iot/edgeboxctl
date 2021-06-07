// +build unit

package utils

import (
	"testing"
	"time"
)

func TestGetSQLiteFormattedDateTime(t *testing.T) {
	datetime := time.Date(2021, time.Month(1), 01, 1, 30, 15, 0, time.UTC)
	result := GetSQLiteFormattedDateTime(datetime)

	if result != "2021-01-01 01:30:15" {
		t.Log("Expected 2021-01-01 01:30:15 but got ", result)
		t.Fail()
	}
}
