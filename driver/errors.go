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
	return fmt.Sprintf("could not get volume path from layer: %s", e.Id)
}
