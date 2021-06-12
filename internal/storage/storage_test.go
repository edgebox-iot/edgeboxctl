// +build unit

package storage

import (
	"fmt"
	"testing"
)

func TestGetDevices(t *testing.T) {
	result := GetDevices()

	fmt.Println(result)
}
