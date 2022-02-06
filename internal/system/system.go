package system

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/edgebox-iot/edgeboxctl/internal/utils"

	"github.com/shirou/gopsutil/host"
	"github.com/joho/godotenv"

)

// GetUptimeInSeconds: Returns a value (as string) of the total system uptime
func GetUptimeInSeconds() string {
	uptime, _ := host.Uptime()

	return strconv.FormatUint(uptime, 10)
}

// GetUptimeFormatted: Returns a humanized version that can be useful for logging
func GetUptimeFormatted() string {
	uptime, _ := host.Uptime()

	days := uptime / (60 * 60 * 24)
	hours := (uptime - (days * 60 * 60 * 24)) / (60 * 60)
	minutes := ((uptime - (days * 60 * 60 * 24)) - (hours * 60 * 60)) / 60
	return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
}

// GetIP: Returns the ip address of the instance 
func GetIP() string {
	ip := ""

	// Trying to find a valid IP (For direct connection, not tunneled)
	ethResult := utils.ExecAndGetLines("/", "ip", []string{"-o", "-4", "addr", "list", "eth0"})
	for ethResult.Scan() {
		adapterRawInfo := strings.Fields(ethResult.Text())
		if ip == "" {
			ip = strings.Split(adapterRawInfo[3], "/")[0]
		}
	}

	// If no IP was found yet, try wlan0
	if ip == "" {
		wlanResult := utils.ExecAndGetLines("/", "ip", []string{"-o", "-4", "addr", "list", "wlan0"})
		for wlanResult.Scan() {
			adapterRawInfo := strings.Fields(wlanResult.Text())
			if ip == "" {
				ip = strings.Split(adapterRawInfo[3], "/")[0]
			}
		}
	}

	return ip
}

func GetHostname() string {
	return utils.Exec("/", "hostname", []string{})
}

// SetupCloudOptions: Reads the designated env file looking for options to write into the options table. Meant to be used on initial setup. Deletes source env file after operation.
func SetupCloudOptions() {

	var cloudEnv map[string]string
	cloudEnv, err := godotenv.Read(utils.GetPath("cloudEnvFileLocation"))

	if err != nil {
		fmt.Println("Error loading .env file for cloud version setup")
	}

	if cloudEnv["NAME"] != "" {
		utils.WriteOption("NAME", cloudEnv["NAME"])
	}

	if cloudEnv["EMAIL"] != "" {
		utils.WriteOption("EMAIL", cloudEnv["EMAIL"])
	}

	if cloudEnv["EDGEBOXIO_API_TOKEN"] != "" {
		utils.WriteOption("EDGEBOXIO_API_TOKEN", cloudEnv["EDGEBOXIO_API_TOKEN"])
	}

	// In the end of this operation takes place, remove the env file as to not overwrite any options once they are set.
	utils.Exec("/", "rm", []string{utils.GetPath("cloudEnvFileLocation")})

}
