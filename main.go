package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s [run|init] ...", os.Args[0])
	}

	switch os.Args[1] {
	case "run":
		run()
	case "init":
		containerInit()
	default:
		log.Fatalf("Unknown command: %s", os.Args[1])
	}
}

func run() {
	runFlags := flag.NewFlagSet("run", flag.ExitOnError)

	rootfsFlag := runFlags.String("rootfs", "", "Path to the root filesystem")
	cgFlag := runFlags.Bool("cgroup", false, "Enable cgroup limits")
	memLimitFlag := runFlags.Int64("mem-limit", 0, "Memory limit in bytes")
	timeLimitFlag := runFlags.Int64("time-limit", 0, "CPU time limit in microseconds")
	pidLimitFlag := runFlags.Int64("pid-limit", 0, "PID limit")

	runFlags.Parse(os.Args[2:])

	if runFlags.NArg() < 2 {
		log.Fatalf("Usage: %s run -rootfs=<path> [flags] <command> [args...]", os.Args[0])
	}

	if *rootfsFlag == "" {
		log.Fatalf("rootfs flag is required")
	}

	rootfs := *rootfsFlag
	useCg := *cgFlag
	memLimit := *memLimitFlag
	timeLimit := *timeLimitFlag
	pidLimit := *pidLimitFlag
	cmdPath := runFlags.Arg(0)
	cmdArgs := runFlags.Args()[1:]

	args := []string{"init", rootfs, cmdPath}
	args = append(args, cmdArgs...)

	cmd := exec.Command("/proc/self/exe", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWCGROUP |
			syscall.CLONE_NEWTIME,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      100000,
				Size:        65536,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      100000,
				Size:        65536,
			},
		},
		GidMappingsEnableSetgroups: false,
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting command: %v", err)
	}

	if useCg {
		setupCgroup(cmd.Process.Pid, cgroupConfig{
			MemoryLimitByte: memLimit,
			CpuMaxUs:        100000,
			CpuPeriodUs:     100000,
			PidLimit:        pidLimit,
		})
	}

	processFinished := make(chan struct{}, 1)

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Fatalf("Error waiting for command: %v", err)
		}
		close(processFinished)
	}()

	select {
	case <-time.After(time.Duration(timeLimit) * time.Microsecond):
		log.Printf("Process time limit exceeded")
	case <-processFinished:
		log.Printf("Process finished successfully")
	}
}

func prepareRootfs(rootfs string) error {
	if err := unix.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("error remounting / as private: %v", err)
	}

	if err := unix.Chroot(rootfs); err != nil {
		return fmt.Errorf("error changing root to %s: %v", rootfs, err)
	}

	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("error changing directory to /: %v", err)
	}

	if err := unix.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("error mounting /proc: %v", err)
	}

	const sysFlag int = unix.MS_NOSUID | unix.MS_NOEXEC | unix.MS_NODEV | unix.MS_RDONLY
	if err := unix.Mount("sysfs", "/sys", "sysfs", uintptr(sysFlag), ""); err != nil {
		return fmt.Errorf("error mounting /sys: %v", err)
	}

	const devFlag int = unix.MS_NOSUID | unix.MS_STRICTATIME | unix.MS_NOEXEC | unix.MS_NODEV
	if err := unix.Mount("tmpfs", "/dev", "tmpfs", uintptr(devFlag), "mode=755"); err != nil {
		return fmt.Errorf("error mounting /dev: %v", err)
	}

	return nil
}

func containerInit() {
	if len(os.Args) < 4 {
		log.Fatalf("Usage: %s init <rootfs> <command> [args...]", os.Args[0])
	}

	rootfs := os.Args[2]
	cmdPath := os.Args[3]
	cmdArgs := os.Args[4:]

	if err := unix.Sethostname([]byte("container")); err != nil {
		log.Fatalf("Error setting hostname: %v", err)
	}

	if err := prepareRootfs(rootfs); err != nil {
		log.Fatalf("Error preparing rootfs: %v", err)
	}

	env := os.Environ()
	if err := unix.Exec(cmdPath, append([]string{cmdPath}, cmdArgs...), env); err != nil {
		log.Fatalf("Exec %s failed: %v", cmdPath, err)
	}
}
