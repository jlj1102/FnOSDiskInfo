//go:build linux

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

// linux/nvme.h: struct nvme_admin_cmd is 72 bytes on x86_64, so
// NVME_IOCTL_ADMIN_CMD = _IOWR('N', 0x41, struct nvme_admin_cmd) = 0xC0484E41.
const nvmeIoctlAdminCmd = 0xC0484E41

type nvmeAdminCmd struct {
	opcode      uint8
	flags       uint8
	commandID   uint16
	nsid        uint32
	cdw2        uint32
	cdw3        uint32
	metadata    uint64
	addr        uint64
	metadataLen uint32
	dataLen     uint32
	cdw10       uint32
	cdw11       uint32
	cdw12       uint32
	cdw13       uint32
	cdw14       uint32
	cdw15       uint32
	timeoutMs   uint32
	result      uint32
}

// nvmeIdentify issues an Identify Controller admin command (opcode 0x06,
// CNS 0x01) and returns the 4 KiB payload; nil on any failure (not root, no
// device node, unsupported kernel, all-zero reply).
func nvmeIdentify(devPath string) []byte {
	dev := nvmeControllerDev(devPath)
	if dev == "" {
		return nil
	}
	fd, err := syscall.Open(dev, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil
	}
	defer syscall.Close(fd)
	buf := make([]byte, 4096)
	cmd := nvmeAdminCmd{
		opcode:    0x06, // Identify
		cdw10:     0x01, // CNS = Identify Controller
		addr:      uint64(uintptr(unsafe.Pointer(&buf[0]))),
		dataLen:   4096,
		timeoutMs: 10000,
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), nvmeIoctlAdminCmd, uintptr(unsafe.Pointer(&cmd)))
	runtime.KeepAlive(buf)
	if errno != 0 {
		return nil
	}
	// Guard against a "successful" empty reply: Identify always carries the
	// model/serial strings (bytes 4-63).
	for _, b := range buf[4:64] {
		if b != 0 {
			return buf
		}
	}
	return nil
}
