package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

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
	if len(os.Args) < 4 {
		log.Fatalf("Usage: %s run <rootfs> <command> [args...]", os.Args[0])
	}

	rootfs := os.Args[2]
	cmdPath := os.Args[3]
	cmdArgs := os.Args[4:]

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

	if err := cmd.Run(); err != nil {
		log.Fatalf("Error running command: %v", err)
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

	if err := syscall.Sethostname([]byte("container")); err != nil {
		log.Fatalf("Error setting hostname: %v", err)
	}

	if err := prepareRootfs(rootfs); err != nil {
		log.Fatalf("Error preparing rootfs: %v", err)
	}

	env := os.Environ()
	if err := syscall.Exec(cmdPath, append([]string{cmdPath}, cmdArgs...), env); err != nil {
		log.Fatalf("Exec %s failed: %v", cmdPath, err)
	}
}
