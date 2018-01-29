package privilege

import winio "github.com/Microsoft/go-winio"

type Elevator struct{}

func (e *Elevator) EnableProcessPrivileges(privileges []string) error {
	return winio.EnableProcessPrivileges(privileges)
}

func (e *Elevator) DisableProcessPrivileges(privileges []string) error {
	return winio.DisableProcessPrivileges(privileges)
}
