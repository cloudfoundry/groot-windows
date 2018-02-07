package volume

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const DISK_QUOTA_OVERHEAD = 10 * 1024

type Limiter struct{}

func (l *Limiter) SetQuota(volumePath string, size uint64) error {
	if size == 0 {
		return nil
	}

	size += DISK_QUOTA_OVERHEAD

	exeFile, err := os.Executable()
	if err != nil {
		return err
	}

	quota, err := windows.LoadDLL(filepath.Join(filepath.Dir(exeFile), "quota.dll"))
	if err != nil {
		return err
	}

	setQuota, err := quota.FindProc("SetQuota")
	if err != nil {
		return err
	}

	volume, err := syscall.UTF16PtrFromString(volumePath)
	if err != nil {
		return err
	}

	r0, _, err := setQuota.Call(uintptr(unsafe.Pointer(volume)), uintptr(size))
	if int32(r0) != 0 {
		return fmt.Errorf("error setting quota: %s\n", err.Error())
	}

	return nil
}
