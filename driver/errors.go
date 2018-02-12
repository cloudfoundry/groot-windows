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
