package storage

import (
	"bufio"
	"strings"

	"github.com/edgebox-iot/edgeboxctl/internal/utils"
)

// Device : Struct representing a storage device in the system
type Device struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Size       string       `json:"size"`
	MainDevice bool         `json:"main_device"`
	MAJ        string       `json:"maj"`
	MIN        string       `json:"min"`
	RM         string       `json:"rm"`
	RO         string       `json:"ro"`
	Status     DeviceStatus `json:"status"`
}

// DeviceStatus : Struct representing possible storage device statuses (code + description)
type DeviceStatus struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// GetDevices : Returns a list of all available sotrage devices in structs filled with information
func GetDevices() []Device {

	var devices []Device

	cmdArgs := []string{"--raw", "--nodeps", "--noheadings"}
	cmdOutput := utils.Exec("lsblk", cmdArgs)
	cmdOutputReader := strings.NewReader(cmdOutput)
	scanner := bufio.NewScanner(cmdOutputReader)
	scanner.Split(bufio.ScanLines)

	mainDevice := true

	for scanner.Scan() {
		// 1 Device is represented here. Extract words in order for filling a Device struct
		// Example deviceRawInfo: "mmcblk0 179:0 0 29.7G 0 disk"

		deviceRawInfo := strings.Fields(scanner.Text())
		majMin := strings.SplitN(deviceRawInfo[1], ":", 2)

		device := Device{
			ID:         deviceRawInfo[0],
			Name:       deviceRawInfo[0],
			Size:       deviceRawInfo[3],
			MainDevice: mainDevice,
			MAJ:        majMin[0],
			MIN:        majMin[1],
			RO:         deviceRawInfo[4],
			RM:         deviceRawInfo[2],
			Status:     DeviceStatus{ID: 1, Description: "healthy"},
		}

		// Once the first device is found, set to false.
		if mainDevice == true {
			mainDevice = false
		}

		devices = append(devices, device)
	}

	return devices
}
