// +build unit

package storage

import (
	"testing"
)

func TestGetDevices(t *testing.T) {
	result := GetDevices()

	if len(result) == 0 {
		t.Log("Expecting at least 1 block device, 0 elements found in slice")
		t.Fail()
	}

	foundDevice := false

	t.Log("Looking for a mmcblk0 or sda device")
	for _, device := range result {
		if device.ID == "mmcblk0" || device.ID == "sda" {
			t.Log("Found target device", device.ID)
			foundDevice = true
		}
	}

	if !foundDevice {
		t.Log("Expected to find device mmcblk0 but did not. Devices:", result)
		t.Fail()
	}
}
