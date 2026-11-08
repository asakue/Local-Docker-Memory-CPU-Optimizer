package analyzer

// ClassifyRisk classifies a disk category into a RiskLevel.
func ClassifyRisk(category string) RiskLevel {
	switch category {
	case "Build Cache", "Dangling Images", "Stopped Containers":
		return RiskSafe
	case "Unused Images":
		return RiskCaution
	case "Unused Volumes":
		return RiskDangerous
	default:
		return RiskCaution
	}
}

// CalculateDiskSummary sums up reclaimable space per risk tier and selects safe items by default.
func CalculateDiskSummary(items []ReclaimableItem, totalDockerSpace int64) DiskAuditSummary {
	summary := DiskAuditSummary{
		TotalDockerSpace: totalDockerSpace,
		Items:            make([]ReclaimableItem, len(items)),
	}

	for i, item := range items {
		// Default selection: select Safe items by default, leave Caution/Dangerous unchecked
		item.Risk = ClassifyRisk(item.Category)
		if item.Risk == RiskSafe {
			item.Selected = true
		} else {
			item.Selected = false
		}

		switch item.Risk {
		case RiskSafe:
			summary.SafeReclaimable += item.Size
		case RiskCaution:
			summary.CautionReclaim += item.Size
		case RiskDangerous:
			summary.DangerReclaim += item.Size
		}

		summary.Items[i] = item
	}

	return summary
}

// BuildSystemReport constructs an overall analytics snapshot.
func BuildSystemReport(containers []ContainerMetrics, disk DiskAuditSummary, hostTotalRAM, hostUsedRAM uint64, idleCfg IdleConfig, oomCfg OOMConfig) SystemReport {
	report := SystemReport{
		HostTotalRAM:     hostTotalRAM,
		HostUsedRAM:      hostUsedRAM,
		ContainersTotal:  len(containers),
		Containers:       containers,
		Disk:             disk,
	}

	for _, c := range containers {
		if c.Status == "running" {
			report.ContainersActive++
		}
		if c.IsIdle {
			report.IdleContainers++
			report.IdleRAMWasted += c.MemoryUsage
		}
		if c.IsOOMCritical || c.IsOOMWarning || c.IsHostRAMAlert {
			report.OOMAlertCount++
		}
	}

	return report
}

