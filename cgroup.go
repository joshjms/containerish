package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const cgroupRoot = "/sys/fs/cgroup/"

type cgroupConfig struct {
	MemoryLimitByte int64
	CpuMaxUs        int64
	CpuPeriodUs     int64
	PidLimit        int64
}

func setupCgroup(pid int, config cgroupConfig) error {
	cgroupName := "containerish"
	cgroupPath := filepath.Join(cgroupRoot, cgroupName)

	// creating the cgroup
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return fmt.Errorf("error creating cgroup directory: %v", err)
	}

	if config.MemoryLimitByte > 0 {
		memoryLimitPath := filepath.Join(cgroupPath, "memory.max")
		if err := os.WriteFile(memoryLimitPath, []byte(fmt.Sprintf("%d", config.MemoryLimitByte)), 0644); err != nil {
			return fmt.Errorf("error setting memory limit: %v", err)
		}
	}

	if config.CpuMaxUs > 0 && config.CpuPeriodUs > 0 {
		cpuMaxPath := filepath.Join(cgroupPath, "cpu.max")
		cpuMaxValue := fmt.Sprintf("%d %d", config.CpuMaxUs, config.CpuPeriodUs)
		if err := os.WriteFile(cpuMaxPath, []byte(cpuMaxValue), 0644); err != nil {
			return fmt.Errorf("error setting CPU limit: %v", err)
		}
	}

	if config.PidLimit > 0 {
		pidLimitPath := filepath.Join(cgroupPath, "pids.max")
		if err := os.WriteFile(pidLimitPath, []byte(fmt.Sprintf("%d", config.PidLimit)), 0644); err != nil {
			return fmt.Errorf("error setting PID limit: %v", err)
		}
	}

	tasksPath := filepath.Join(cgroupPath, "cgroup.procs")
	if err := os.WriteFile(tasksPath, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		return fmt.Errorf("error adding process to cgroup: %v", err)
	}

	return nil
}

func readCgroupStats() (map[string]string, error) {
	stats := make(map[string]string)

	cgroupName := "containerish"
	cgroupPath := filepath.Join(cgroupRoot, cgroupName)

	memoryUsagePath := filepath.Join(cgroupPath, "memory.current")
	memoryUsageData, err := os.ReadFile(memoryUsagePath)
	if err != nil {
		return nil, fmt.Errorf("error reading memory usage: %v", err)
	}
	stats["memory.current"] = string(memoryUsageData)

	memoryPeakPath := filepath.Join(cgroupPath, "memory.peak")
	memoryPeakData, err := os.ReadFile(memoryPeakPath)
	if err != nil {
		return nil, fmt.Errorf("error reading memory peak: %v", err)
	}
	stats["memory.peak"] = string(memoryPeakData)

	cpuUsagePath := filepath.Join(cgroupPath, "cpu.stat")
	cpuUsageData, err := os.ReadFile(cpuUsagePath)
	if err != nil {
		return nil, fmt.Errorf("error reading CPU usage: %v", err)
	}
	stats["cpu.stat"] = string(cpuUsageData)

	return stats, nil
}

func cleanupCgroup() error {
	cgroupName := "containerish"
	cgroupPath := filepath.Join(cgroupRoot, cgroupName)

	if err := os.RemoveAll(cgroupPath); err != nil {
		return fmt.Errorf("error removing cgroup: %v", err)
	}

	return nil
}
