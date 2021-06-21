package system

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/edgebox-iot/edgeboxctl/internal/utils"

	"github.com/shirou/gopsutil/host"
)

func GetUptimeInSeconds() string {
	uptime, _ := host.Uptime()

	return strconv.FormatUint(uptime, 10)
}

func GetUptimeFormatted() string {
	uptime, _ := host.Uptime()

	days := uptime / (60 * 60 * 24)
	hours := (uptime - (days * 60 * 60 * 24)) / (60 * 60)
	minutes := ((uptime - (days * 60 * 60 * 24)) - (hours * 60 * 60)) / 60
	return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
}

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
