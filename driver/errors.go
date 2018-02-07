package driver

import "fmt"

type LayerExistsError struct {
	Id string
}

func (e *LayerExistsError) Error() string {
	return fmt.Sprintf("layer already exists: %s", e.Id)
}

type MissingVolumePathError struct {
	Id string
}

func (e *MissingVolumePathError) Error() string {
	return fmt.Sprintf("could not get volume path for bundle ID: %s", e.Id)
}

type EmptyDriverStoreError struct{}

func (e *EmptyDriverStoreError) Error() string {
	return fmt.Sprintf("driver store must be set")
}

type InvalidDiskLimitError struct {
	Limit int64
}

func (e *InvalidDiskLimitError) Error() string {
	return fmt.Sprintf("invalid disk limit: %d", e.Limit)
}

type DiskLimitTooSmallError struct {
	Limit int64
	Base  int64
}

func (e *DiskLimitTooSmallError) Error() string {
	return fmt.Sprintf("disk limit %d smaller than image size: %d", e.Limit, e.Base)
}
