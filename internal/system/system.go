package system

import (
	"fmt"
	"strconv"

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
