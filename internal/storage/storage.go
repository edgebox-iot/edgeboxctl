package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/edgebox-iot/edgeboxctl/internal/diagnostics"
	"github.com/edgebox-iot/edgeboxctl/internal/utils"
	"github.com/shirou/gopsutil/disk"
)

// Device : Struct representing a storage device in the system
type Device struct {
	ID         DeviceIdentifier `json:"id"`
	Name       string           `json:"name"`
	Size       string           `json:"size"`
	InUse      bool             `json:"in_use"`
	MainDevice bool             `json:"main_device"`
	MAJ        string           `json:"maj"`
	MIN        string           `json:"min"`
	RM         string           `json:"rm"`
	RO         string           `json:"ro"`
	Partitions []Partition      `json:"partitions"`
	Status     DeviceStatus     `json:"status"`
	UsageStat  UsageStat        `json:"usage_stat"`
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
	Total      uint64     `json:"total"`
	Used       uint64     `json:"used"`
	Free       uint64     `json:"free"`
	Percent    string     `json:"percent"`
	UsageSplit UsageSplit `json:"usage_split"`
}

type UsageSplit struct {
	OS       uint64 `json:"os"`
	EdgeApps uint64 `json:"edgeapps"`
	Buckets  uint64 `json:"buckets"`
	Others   uint64 `json:"others"`
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

type DeviceIdentifier string

const (
	DISK_TYPE_SDA   DeviceIdentifier = "sda"
	DISK_TYPE_MCBLK DeviceIdentifier = "mmcblk0"
	DISK_TYPE_VDA   DeviceIdentifier = "vda"
)

func GetDeviceIdentifier(release_version diagnostics.ReleaseVersion) DeviceIdentifier {
	switch release_version {
	case diagnostics.CLOUD_VERSION:
		return DISK_TYPE_VDA
	case diagnostics.PROD_VERSION:
		return DISK_TYPE_MCBLK
	}

	return DISK_TYPE_SDA
}

// GetDevices : Returns a list of all available sotrage devices in structs filled with information
func GetDevices(release_version diagnostics.ReleaseVersion) []Device {

	var devices []Device

	cmdArgs := []string{"--raw", "--bytes", "--noheadings"}
	scanner := utils.ExecAndGetLines("/", "lsblk", cmdArgs)

	var currentDevice Device
	var currentPartitions []Partition

	firstDevice := true
	currentDeviceInUseFlag := false

	mainDiskID := GetDeviceIdentifier(release_version)

	for scanner.Scan() {
		// 1 Device is represented here. Extract words in order for filling a Device struct
		// Example deviceRawInfo: "mmcblk0 179:0 0 29.7G 0 disk"
		deviceRawInfo := strings.Fields(scanner.Text())
		majMin := strings.SplitN(deviceRawInfo[1], ":", 2)

		isDevice := false
		isPartition := false
		if deviceRawInfo[5] == "part" {
			isDevice = false
			isPartition = true

		} else if deviceRawInfo[5] == "disk" {
			isDevice = true
			isPartition = false
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
				ID:         DeviceIdentifier(deviceRawInfo[0]),
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

		} else if isPartition {

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

		} else {
			fmt.Println("Found device not compatible with Edgebox, ignoring.")
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

			deviceUsageStat := UsageStat{}

			for partitionIndex, partition := range device.Partitions {

				if partition.Mountpoint != "" {

					s, _ := disk.Usage(partition.Mountpoint)

					if s.Total == 0 {
						continue
					}

					partitionUsagePercent := fmt.Sprintf("%2.f%%", s.UsedPercent)
					osUsageSplit := (uint64)(0)
					edgeappsUsageSplit := (uint64)(0)
					bucketsUsageSplit := (uint64)(0)
					othersUsageSplit := (uint64)(0)

					edgeappsDirSize, _ := getDirSize(utils.GetPath(utils.EdgeAppsPath))
					// TODO for later: Figure out to get correct paths for each partition...
					wsAppDataDirSize, _ := getDirSize("/home/system/components/ws/appdata")

					if partition.Mountpoint == "/" {
						edgeappsUsageSplit = edgeappsDirSize + wsAppDataDirSize
						deviceUsageStat.UsageSplit.EdgeApps += edgeappsUsageSplit
					}

					if device.MainDevice {
						osUsageSplit = (s.Used - othersUsageSplit - bucketsUsageSplit - edgeappsUsageSplit)
					} else {
						othersUsageSplit = (s.Used - bucketsUsageSplit - edgeappsUsageSplit)
					}

					partitionUsageSplit := UsageSplit{
						OS:       osUsageSplit,
						EdgeApps: edgeappsUsageSplit,
						Buckets:  bucketsUsageSplit,
						Others:   othersUsageSplit,
					}

					deviceUsageStat.Total = deviceUsageStat.Total + s.Total
					deviceUsageStat.Used = deviceUsageStat.Used + s.Used
					deviceUsageStat.Free = deviceUsageStat.Free + s.Free
					deviceUsageStat.UsageSplit.OS = deviceUsageStat.UsageSplit.OS + osUsageSplit
					deviceUsageStat.UsageSplit.EdgeApps = deviceUsageStat.UsageSplit.EdgeApps + edgeappsUsageSplit
					deviceUsageStat.UsageSplit.Buckets = deviceUsageStat.UsageSplit.Buckets + bucketsUsageSplit
					deviceUsageStat.UsageSplit.Others = deviceUsageStat.UsageSplit.Others + othersUsageSplit

					devices[deviceIndex].Partitions[partitionIndex].UsageStat = UsageStat{
						Total:      s.Total,
						Used:       s.Used,
						Free:       s.Free,
						Percent:    partitionUsagePercent,
						UsageSplit: partitionUsageSplit,
					}

				}

			}

			devices[deviceIndex].UsageStat = deviceUsageStat
			totalDevicePercentUsage := fmt.Sprintf("%2.f%%", (float32(devices[deviceIndex].UsageStat.Used)/float32(devices[deviceIndex].UsageStat.Total))*100)
			devices[deviceIndex].UsageStat.Percent = totalDevicePercentUsage

		}

	}

	return devices
}

func getDirSize(path string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += (uint64)(info.Size())
		}
		return err
	})
	return size, err
}
