package storage

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/edgebox-iot/edgeboxctl/internal/utils"
	"github.com/shirou/gopsutil/disk"
)

// Device : Struct representing a storage device in the system
type Device struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Size       string       `json:"size"`
	InUse      bool         `json:"in_use"`
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

type UsageStat struct {
	Total   string `json:"total"`
	Used    string `json:"used"`
	Free    string `json:"free"`
	Percent string `json:"percent"`
}

// Partition : Struct representing a partition / filesystem (Empty Mountpoint means it is not mounted)
type Partition struct {
	ID         string    `json:"id"`
	Size       string    `json:"size"`
	MAJ        string    `json:"maj"`
	MIN        string    `json:"min"`
	RM         string    `json:"rm"`
	RO         string    `json:"ro"`
	Filesystem string    `json:"filesystem"`
	Mountpoint string    `json:"mountpoint"`
	UsageStat  UsageStat `json:"usage_stat"`
}

const mainDiskID = "mmcblk0"

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
	currentDeviceInUseFlag := false

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
			// Clean up on the latest device being prepared. Append all partitions found and delete the currentPartitions list afterwards.
			// The first device found should not run the cleanup lines below

			if !firstDevice {
				currentDevice.Partitions = currentPartitions

				if !currentDeviceInUseFlag {
					currentDevice.Status.ID = 0
					currentDevice.Status.Description = "not configured"
				}

				currentDevice.InUse = currentDeviceInUseFlag
				currentDeviceInUseFlag = false
				currentPartitions = []Partition{}
				devices = append(devices, currentDevice)
			} else {
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

			mountpoint := ""
			if len(deviceRawInfo) >= 7 {
				mountpoint = deviceRawInfo[6]
				currentDeviceInUseFlag = true
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

	currentDevice.Partitions = currentPartitions
	if !currentDeviceInUseFlag {
		currentDevice.Status.ID = 0
		currentDevice.Status.Description = "Not configured"
	}
	currentDevice.InUse = currentDeviceInUseFlag
	devices = append([]Device{currentDevice}, devices...) // Prepending the first device...

	devices = getDevicesSpaceUsage(devices)

	return devices
}

func getDevicesSpaceUsage(devices []Device) []Device {

	for deviceIndex, device := range devices {

		if device.InUse {

			for partitionIndex, partition := range device.Partitions {

				s, _ := disk.Usage(partition.Mountpoint)
				if s.Total == 0 {
					continue
				}

				partitionUsagePercent := fmt.Sprintf("%2.f%%", s.UsedPercent)
				devices[deviceIndex].Partitions[partitionIndex].UsageStat = UsageStat{Total: strconv.FormatUint(s.Total, 10), Used: strconv.FormatUint(s.Used, 10), Free: strconv.FormatUint(s.Free, 10), Percent: partitionUsagePercent}

			}

		}

	}

	return devices
}
