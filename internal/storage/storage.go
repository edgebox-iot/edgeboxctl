package storage

import (
	"bufio"
	"fmt"
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
	Partitions []Partition  `json:"partitions"`
	Status     DeviceStatus `json:"status"`
}

// DeviceStatus : Struct representing possible storage device statuses (code + description)
type DeviceStatus struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// MaybeDevice : Boolean flag for validation of device existance
type MaybeDevice struct {
	Device Device `json:"device"`
	Valid  bool   `json:"valid"`
}

// Partition : Struct representing a partition / filesystem (Empty Mountpoint means it is not mounted)
type Partition struct {
	ID         string `json:"id"`
	Size       string `json:"size"`
	MAJ        string `json:"maj"`
	MIN        string `json:"min"`
	RM         string `json:"rm"`
	RO         string `json:"ro"`
	Filesystem string `json:"filesystem"`
	Mountpoint string `json:"mountpoint"`
}

const mainDiskID = "mmcblk0"

func GetDevice() MaybeDevice {

	result := MaybeDevice{
		Device: Device{},
		Valid:  false,
	}

	return result

}

// GetDevices : Returns a list of all available sotrage devices in structs filled with information
func GetDevices() []Device {

	var devices []Device

	cmdArgs := []string{"--raw", "--noheadings"}
	cmdOutput := utils.Exec("lsblk", cmdArgs)
	cmdOutputReader := strings.NewReader(cmdOutput)
	scanner := bufio.NewScanner(cmdOutputReader)
	scanner.Split(bufio.ScanLines)

	var currentDevice Device
	var currentPartitions []Partition

	firstDevice := true

	for scanner.Scan() {
		// 1 Device is represented here. Extract words in order for filling a Device struct
		// Example deviceRawInfo: "mmcblk0 179:0 0 29.7G 0 disk"
		deviceRawInfo := strings.Fields(scanner.Text())
		majMin := strings.SplitN(deviceRawInfo[1], ":", 2)

		isDevice := true
		if deviceRawInfo[5] == "part" {
			isDevice = false
		}

		if isDevice {
			// Clean up on the last device being prepared. Append all partitions found and delete the currentPartitions list afterwards.
			// The first device found should not run the cleanup lines below

			fmt.Println("Processing Device")

			if !firstDevice {
				fmt.Println("Appending finalized device info to list")
				currentDevice.Partitions = currentPartitions
				currentPartitions = []Partition{}
				devices = append(devices, currentDevice)
			} else {
				fmt.Println("First device, not appending to list")
				firstDevice = false
			}

			mainDevice := false

			device := Device{
				ID:         deviceRawInfo[0],
				Name:       deviceRawInfo[0],
				Size:       deviceRawInfo[3],
				MainDevice: mainDevice,
				MAJ:        majMin[0],
				MIN:        majMin[1],
				RM:         deviceRawInfo[2],
				RO:         deviceRawInfo[4],
				Status:     DeviceStatus{ID: 1, Description: "healthy"},
			}

			if device.ID == mainDiskID {
				device.MainDevice = true
			}

			currentDevice = device

		} else {

			fmt.Println("Processing Partition")

			mountpoint := ""
			if len(deviceRawInfo) > 7 {
				mountpoint = deviceRawInfo[7]
			}

			// It is a partition, part of the last device read.
			partition := Partition{
				ID:         deviceRawInfo[0],
				Size:       deviceRawInfo[3],
				MAJ:        majMin[0],
				MIN:        majMin[1],
				RM:         deviceRawInfo[2],
				RO:         deviceRawInfo[4],
				Filesystem: "",
				Mountpoint: mountpoint,
			}

			currentPartitions = append(currentPartitions, partition)

		}

	}

	devices = append([]Device{currentDevice}, devices...) // Prepending the first device...

	return devices
}
