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
