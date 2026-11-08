package analyzer

import "time"

// RiskLevel defines the safety level for reclaiming Docker resources.
type RiskLevel string

const (
	RiskSafe      RiskLevel = "SAFE"      // Build cache, dangling images, stopped containers
	RiskCaution   RiskLevel = "CAUTION"   // Unused tagged images
	RiskDangerous RiskLevel = "DANGEROUS" // Unused volumes
)

// ContainerMetrics holds live resource consumption for a container.
type ContainerMetrics struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Project        string    `json:"project"` // from com.docker.compose.project label
	Image          string    `json:"image"`
	Status         string    `json:"status"` // running, paused, exited
	State          string    `json:"state"`
	Created        time.Time `json:"created"`
	StartedAt      time.Time `json:"started_at"`
	CPUPercent     float64   `json:"cpu_percent"`
	MemoryUsage    uint64    `json:"memory_usage_bytes"`
	MemoryLimit    uint64    `json:"memory_limit_bytes"`
	HasCustomLimit bool      `json:"has_custom_limit"`
	NetRxBytes     uint64    `json:"net_rx_bytes"`
	NetTxBytes     uint64    `json:"net_tx_bytes"`

	// Analysis fields
	IsIdle         bool      `json:"is_idle"`
	IdleDuration   time.Duration `json:"idle_duration"`
	MemoryPercent  float64   `json:"memory_percent"` // % of container limit or % of host RAM
	IsOOMWarning   bool      `json:"is_oom_warning"` // >80% limit
	IsOOMCritical  bool      `json:"is_oom_critical"` // >90% limit
	IsHostRAMAlert bool      `json:"is_host_ram_alert"` // >30% host RAM when no limit
}

// ReclaimableItem represents an item or category of disk space that can be reclaimed.
type ReclaimableItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"` // "Build Cache", "Dangling Images", "Stopped Containers", "Unused Images", "Volumes"
	Size        int64     `json:"size_bytes"`
	Risk        RiskLevel `json:"risk_level"`
	Description string    `json:"description"`
	Selected    bool      `json:"selected"`
}

// DiskAuditSummary holds high-level summary of reclaimable disk space.
type DiskAuditSummary struct {
	TotalDockerSpace int64             `json:"total_docker_space_bytes"`
	SafeReclaimable  int64             `json:"safe_reclaimable_bytes"`
	CautionReclaim   int64             `json:"caution_reclaimable_bytes"`
	DangerReclaim    int64             `json:"danger_reclaimable_bytes"`
	Items            []ReclaimableItem `json:"items"`
}

// SystemReport represents a complete snapshot of containers, leaks, and disk space.
type SystemReport struct {
	HostTotalRAM     uint64             `json:"host_total_ram_bytes"`
	HostUsedRAM      uint64             `json:"host_used_ram_bytes"`
	ContainersTotal  int                `json:"containers_total"`
	ContainersActive int                `json:"containers_active"`
	IdleContainers   int                `json:"idle_containers_count"`
	IdleRAMWasted    uint64             `json:"idle_ram_wasted_bytes"`
	OOMAlertCount    int                `json:"oom_alert_count"`
	Containers       []ContainerMetrics `json:"containers"`
	Disk             DiskAuditSummary   `json:"disk"`
}

