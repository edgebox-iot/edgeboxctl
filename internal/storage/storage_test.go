// +build unit

package storage

import (
	"testing"
)

func TestGetDevices(t *testing.T) {

	t.Log("Testing with release version dev")
	assertGetDevices(GetDevices("dev"), t)
	t.Log("Testing with release version prod")
	assertGetDevices(GetDevices("prod"), t)
	t.Log("Testing with release version cloud")
	assertGetDevices(GetDevices("cloud"), t)

}

func assertGetDevices(devices []Device, t *testing.T) {

	if len(devices) == 0 {
		t.Log("Expecting at least 1 block device, 0 elements found in slice")
		t.Fail()
	}

	foundDevice := false

	t.Log("Looking for a mmcblk0, sda or dva device")
	for _, device := range devices {

		if device.ID == "mmcblk0" || device.ID == "sda" || device.ID == "vda" {
			t.Log("Found target device", device.ID)
			foundDevice = true
		}

	}

	if !foundDevice {
		t.Log("Expected to find device mmcblk0, sda or dva but did not. Devices:", devices)
		t.Fail()
	}
}
