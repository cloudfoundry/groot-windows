package toot

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type Toot struct {
	BaseDir string
}

func (t *Toot) Unpack(logger lager.Logger, id string, parentIDs []string, layerTar io.Reader) error {
	logger.Info("unpack-info")
	logger.Debug("unpack-debug")

	if _, exists := os.LookupEnv("TOOT_UNPACK_ERROR"); exists {
		return errors.New("unpack-err")
	}

	layerTarContents, err := ioutil.ReadAll(layerTar)
	must(err)
	saveObject([]interface{}{
		UnpackArgs{ID: id, ParentIDs: parentIDs, LayerTarContents: layerTarContents},
	}, t.pathTo(UnpackArgsFileName))
	return nil
}

func (t *Toot) Bundle(logger lager.Logger, id string, layerIDs []string) (specs.Spec, error) {
	logger.Info("bundle-info")
	logger.Debug("bundle-debug")

	if _, exists := os.LookupEnv("TOOT_BUNDLE_ERROR"); exists {
		return specs.Spec{}, errors.New("bundle-err")
	}

	saveObject([]interface{}{
		BundleArgs{ID: id, LayerIDs: layerIDs},
	}, t.pathTo(BundleArgsFileName))
	return BundleRuntimeSpec, nil
}

func (t *Toot) Delete(logger lager.Logger, id string) error {
	logger.Info("delete-info")
	logger.Debug("delete-debug")

	if _, exists := os.LookupEnv("TOOT_DELETE_ERROR"); exists {
		return errors.New("delete-err")
	}

	saveObject([]interface{}{
		DeleteArgs{BundleID: id},
	}, t.pathTo(DeleteArgsFileName))
	return nil
}

func (t *Toot) Exists(logger lager.Logger, layerID string) bool {
	logger.Info("exists-info")
	logger.Debug("exists-debug")

	if _, exists := os.LookupEnv("TOOT_LAYER_EXISTS"); exists {
		return true
	}

	saveObject([]interface{}{
		ExistsArgs{LayerID: layerID},
	}, t.pathTo(ExistsArgsFileName),
	)
	return false
}

const (
	UnpackArgsFileName = "unpack-args"
	BundleArgsFileName = "bundle-args"
	ExistsArgsFileName = "exists-args"
	DeleteArgsFileName = "delete-args"
)

var (
	BundleRuntimeSpec = specs.Spec{Root: &specs.Root{Path: "toot-rootfs-path"}}
)

type ExistsCalls []ExistsArgs
type ExistsArgs struct {
	LayerID string
}

type DeleteCalls []DeleteArgs
type DeleteArgs struct {
	BundleID string
}

type UnpackCalls []UnpackArgs
type UnpackArgs struct {
	ID               string
	ParentIDs        []string
	LayerTarContents []byte
}

type BundleCalls []BundleArgs
type BundleArgs struct {
	ID       string
	LayerIDs []string
}

func (t *Toot) pathTo(filename string) string {
	return filepath.Join(t.BaseDir, filename)
}

func saveObject(obj []interface{}, pathname string) {
	if _, err := os.Stat(pathname); err == nil {
		currentCall := obj[0]
		loadObject(&obj, pathname)
		obj = append(obj, currentCall)
	}

	serialisedObj, err := json.Marshal(obj)
	must(err)
	must(ioutil.WriteFile(pathname, serialisedObj, 0600))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func loadObject(obj *[]interface{}, pathname string) {
	file, err := os.Open(pathname)
	defer file.Close()
	must(err)

	err = json.NewDecoder(file).Decode(obj)
	must(err)
}
