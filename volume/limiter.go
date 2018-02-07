package volume

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	setQuota, err := loadProc("SetQuota")
	if err != nil {
		return err
	}

	volume, err := syscall.UTF16PtrFromString(volumePath)
	if err != nil {
		return err
	}

	r0, _, err := setQuota.Call(uintptr(unsafe.Pointer(volume)), uintptr(size))
	if int32(r0) != 0 {
		return fmt.Errorf("error setting quota: %s", windowsErrorMessage(uint32(r0)))
	}

	return nil
}

func (s *Limiter) GetQuotaUsed(volumePath string) (uint64, error) {
	getQuotaUsed, err := loadProc("GetQuotaUsed")
	if err != nil {
		return 0, err
	}

	volume, err := syscall.UTF16PtrFromString(volumePath)
	if err != nil {
		return 0, err
	}

	var quotaUsed uint64

	r0, _, err := getQuotaUsed.Call(uintptr(unsafe.Pointer(volume)), uintptr(unsafe.Pointer(&quotaUsed)))
	if int32(r0) != 0 {
		return 0, fmt.Errorf("error getting quota: %s", windowsErrorMessage(uint32(r0)))
	}

	return quotaUsed, nil
}

func loadProc(proc string) (*windows.Proc, error) {
	exeFile, err := os.Executable()
	if err != nil {
		return nil, err
	}

	quota, err := windows.LoadDLL(filepath.Join(filepath.Dir(exeFile), "quota.dll"))
	if err != nil {
		return nil, err
	}

	return quota.FindProc(proc)
}

func windowsErrorMessage(code uint32) string {
	flags := uint32(windows.FORMAT_MESSAGE_FROM_SYSTEM | windows.FORMAT_MESSAGE_IGNORE_INSERTS)
	langId := uint32(windows.SUBLANG_ENGLISH_US)<<10 | uint32(windows.LANG_ENGLISH)
	buf := make([]uint16, 512)

	_, err := windows.FormatMessage(flags, uintptr(0), code, langId, buf, nil)
	if err != nil {
		return fmt.Sprintf("0x%x", code)
	}
	return strings.TrimSpace(syscall.UTF16ToString(buf))
}
