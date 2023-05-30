package backups

// import (
// 	"fmt"
// 	"os"
// 	"path/filepath"
// 	"strings"

// 	"github.com/edgebox-iot/edgeboxctl/internal/diagnostics"
// 	"github.com/edgebox-iot/edgeboxctl/internal/utils"
// 	"github.com/shirou/gopsutil/disk"
// )

// Repository : Struct representing the backup repository of a device in the system
type Repository struct {
	ID         string 			`json:"id"`
	FileCount  int64            `json:"file_count"`
	Size       string           `json:"size"`
	Snapshots  []Snapshot      	`json:"snapshots"`
	Status     string     		`json:"status"`
	// UsageStat  UsageStat        `json:"usage_stat"`
}

// Snapshot : Struct representing a single snapshot in the backup repository
type Snapshot struct {
	ID         string 			`json:"id"`
	time 	   string           `json:"time"`
}