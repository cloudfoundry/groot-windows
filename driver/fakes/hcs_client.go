// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"code.cloudfoundry.org/groot-windows/driver"
	"code.cloudfoundry.org/groot-windows/hcs"
	"github.com/Microsoft/hcsshim"
)

type HCSClient struct {
	NewLayerWriterStub        func(hcsshim.DriverInfo, string, []string) (hcs.LayerWriter, error)
	newLayerWriterMutex       sync.RWMutex
	newLayerWriterArgsForCall []struct {
		arg1 hcsshim.DriverInfo
		arg2 string
		arg3 []string
	}
	newLayerWriterReturns struct {
		result1 hcs.LayerWriter
		result2 error
	}
	newLayerWriterReturnsOnCall map[int]struct {
		result1 hcs.LayerWriter
		result2 error
	}
	CreateLayerStub        func(hcsshim.DriverInfo, string, string, []string) error
	createLayerMutex       sync.RWMutex
	createLayerArgsForCall []struct {
		arg1 hcsshim.DriverInfo
		arg2 string
		arg3 string
		arg4 []string
	}
	createLayerReturns struct {
		result1 error
	}
	createLayerReturnsOnCall map[int]struct {
		result1 error
	}
	LayerExistsStub        func(hcsshim.DriverInfo, string) (bool, error)
	layerExistsMutex       sync.RWMutex
	layerExistsArgsForCall []struct {
		arg1 hcsshim.DriverInfo
		arg2 string
	}
	layerExistsReturns struct {
		result1 bool
		result2 error
	}
	layerExistsReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	GetLayerMountPathStub        func(hcsshim.DriverInfo, string) (string, error)
	getLayerMountPathMutex       sync.RWMutex
	getLayerMountPathArgsForCall []struct {
		arg1 hcsshim.DriverInfo
		arg2 string
	}
	getLayerMountPathReturns struct {
		result1 string
		result2 error
	}
	getLayerMountPathReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *HCSClient) NewLayerWriter(arg1 hcsshim.DriverInfo, arg2 string, arg3 []string) (hcs.LayerWriter, error) {
	var arg3Copy []string
	if arg3 != nil {
		arg3Copy = make([]string, len(arg3))
		copy(arg3Copy, arg3)
	}
	fake.newLayerWriterMutex.Lock()
	ret, specificReturn := fake.newLayerWriterReturnsOnCall[len(fake.newLayerWriterArgsForCall)]
	fake.newLayerWriterArgsForCall = append(fake.newLayerWriterArgsForCall, struct {
		arg1 hcsshim.DriverInfo
		arg2 string
		arg3 []string
	}{arg1, arg2, arg3Copy})
	fake.recordInvocation("NewLayerWriter", []interface{}{arg1, arg2, arg3Copy})
	fake.newLayerWriterMutex.Unlock()
	if fake.NewLayerWriterStub != nil {
		return fake.NewLayerWriterStub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.newLayerWriterReturns.result1, fake.newLayerWriterReturns.result2
}

func (fake *HCSClient) NewLayerWriterCallCount() int {
	fake.newLayerWriterMutex.RLock()
	defer fake.newLayerWriterMutex.RUnlock()
	return len(fake.newLayerWriterArgsForCall)
}

func (fake *HCSClient) NewLayerWriterArgsForCall(i int) (hcsshim.DriverInfo, string, []string) {
	fake.newLayerWriterMutex.RLock()
	defer fake.newLayerWriterMutex.RUnlock()
	return fake.newLayerWriterArgsForCall[i].arg1, fake.newLayerWriterArgsForCall[i].arg2, fake.newLayerWriterArgsForCall[i].arg3
}

func (fake *HCSClient) NewLayerWriterReturns(result1 hcs.LayerWriter, result2 error) {
	fake.NewLayerWriterStub = nil
	fake.newLayerWriterReturns = struct {
		result1 hcs.LayerWriter
		result2 error
	}{result1, result2}
}

func (fake *HCSClient) NewLayerWriterReturnsOnCall(i int, result1 hcs.LayerWriter, result2 error) {
	fake.NewLayerWriterStub = nil
	if fake.newLayerWriterReturnsOnCall == nil {
		fake.newLayerWriterReturnsOnCall = make(map[int]struct {
			result1 hcs.LayerWriter
			result2 error
		})
	}
	fake.newLayerWriterReturnsOnCall[i] = struct {
		result1 hcs.LayerWriter
		result2 error
	}{result1, result2}
}

func (fake *HCSClient) CreateLayer(arg1 hcsshim.DriverInfo, arg2 string, arg3 string, arg4 []string) error {
	var arg4Copy []string
	if arg4 != nil {
		arg4Copy = make([]string, len(arg4))
		copy(arg4Copy, arg4)
	}
	fake.createLayerMutex.Lock()
	ret, specificReturn := fake.createLayerReturnsOnCall[len(fake.createLayerArgsForCall)]
	fake.createLayerArgsForCall = append(fake.createLayerArgsForCall, struct {
		arg1 hcsshim.DriverInfo
		arg2 string
		arg3 string
		arg4 []string
	}{arg1, arg2, arg3, arg4Copy})
	fake.recordInvocation("CreateLayer", []interface{}{arg1, arg2, arg3, arg4Copy})
	fake.createLayerMutex.Unlock()
	if fake.CreateLayerStub != nil {
		return fake.CreateLayerStub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.createLayerReturns.result1
}

func (fake *HCSClient) CreateLayerCallCount() int {
	fake.createLayerMutex.RLock()
	defer fake.createLayerMutex.RUnlock()
	return len(fake.createLayerArgsForCall)
}

func (fake *HCSClient) CreateLayerArgsForCall(i int) (hcsshim.DriverInfo, string, string, []string) {
	fake.createLayerMutex.RLock()
	defer fake.createLayerMutex.RUnlock()
	return fake.createLayerArgsForCall[i].arg1, fake.createLayerArgsForCall[i].arg2, fake.createLayerArgsForCall[i].arg3, fake.createLayerArgsForCall[i].arg4
}

func (fake *HCSClient) CreateLayerReturns(result1 error) {
	fake.CreateLayerStub = nil
	fake.createLayerReturns = struct {
		result1 error
	}{result1}
}

func (fake *HCSClient) CreateLayerReturnsOnCall(i int, result1 error) {
	fake.CreateLayerStub = nil
	if fake.createLayerReturnsOnCall == nil {
		fake.createLayerReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.createLayerReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *HCSClient) LayerExists(arg1 hcsshim.DriverInfo, arg2 string) (bool, error) {
	fake.layerExistsMutex.Lock()
	ret, specificReturn := fake.layerExistsReturnsOnCall[len(fake.layerExistsArgsForCall)]
	fake.layerExistsArgsForCall = append(fake.layerExistsArgsForCall, struct {
		arg1 hcsshim.DriverInfo
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("LayerExists", []interface{}{arg1, arg2})
	fake.layerExistsMutex.Unlock()
	if fake.LayerExistsStub != nil {
		return fake.LayerExistsStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.layerExistsReturns.result1, fake.layerExistsReturns.result2
}

func (fake *HCSClient) LayerExistsCallCount() int {
	fake.layerExistsMutex.RLock()
	defer fake.layerExistsMutex.RUnlock()
	return len(fake.layerExistsArgsForCall)
}

func (fake *HCSClient) LayerExistsArgsForCall(i int) (hcsshim.DriverInfo, string) {
	fake.layerExistsMutex.RLock()
	defer fake.layerExistsMutex.RUnlock()
	return fake.layerExistsArgsForCall[i].arg1, fake.layerExistsArgsForCall[i].arg2
}

func (fake *HCSClient) LayerExistsReturns(result1 bool, result2 error) {
	fake.LayerExistsStub = nil
	fake.layerExistsReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *HCSClient) LayerExistsReturnsOnCall(i int, result1 bool, result2 error) {
	fake.LayerExistsStub = nil
	if fake.layerExistsReturnsOnCall == nil {
		fake.layerExistsReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.layerExistsReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *HCSClient) GetLayerMountPath(arg1 hcsshim.DriverInfo, arg2 string) (string, error) {
	fake.getLayerMountPathMutex.Lock()
	ret, specificReturn := fake.getLayerMountPathReturnsOnCall[len(fake.getLayerMountPathArgsForCall)]
	fake.getLayerMountPathArgsForCall = append(fake.getLayerMountPathArgsForCall, struct {
		arg1 hcsshim.DriverInfo
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("GetLayerMountPath", []interface{}{arg1, arg2})
	fake.getLayerMountPathMutex.Unlock()
	if fake.GetLayerMountPathStub != nil {
		return fake.GetLayerMountPathStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.getLayerMountPathReturns.result1, fake.getLayerMountPathReturns.result2
}

func (fake *HCSClient) GetLayerMountPathCallCount() int {
	fake.getLayerMountPathMutex.RLock()
	defer fake.getLayerMountPathMutex.RUnlock()
	return len(fake.getLayerMountPathArgsForCall)
}

func (fake *HCSClient) GetLayerMountPathArgsForCall(i int) (hcsshim.DriverInfo, string) {
	fake.getLayerMountPathMutex.RLock()
	defer fake.getLayerMountPathMutex.RUnlock()
	return fake.getLayerMountPathArgsForCall[i].arg1, fake.getLayerMountPathArgsForCall[i].arg2
}

func (fake *HCSClient) GetLayerMountPathReturns(result1 string, result2 error) {
	fake.GetLayerMountPathStub = nil
	fake.getLayerMountPathReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *HCSClient) GetLayerMountPathReturnsOnCall(i int, result1 string, result2 error) {
	fake.GetLayerMountPathStub = nil
	if fake.getLayerMountPathReturnsOnCall == nil {
		fake.getLayerMountPathReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.getLayerMountPathReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *HCSClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.newLayerWriterMutex.RLock()
	defer fake.newLayerWriterMutex.RUnlock()
	fake.createLayerMutex.RLock()
	defer fake.createLayerMutex.RUnlock()
	fake.layerExistsMutex.RLock()
	defer fake.layerExistsMutex.RUnlock()
	fake.getLayerMountPathMutex.RLock()
	defer fake.getLayerMountPathMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *HCSClient) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ driver.HCSClient = new(HCSClient)
