// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package lightfilenodeindex

import (
	"context"
	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/dgraph-io/badger/v4"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex/indexpb"
	"github.com/ipfs/go-cid"
	"sync"
)

// Ensure, that configServiceMock does implement configService.
// If this is not the case, regenerate this file with moq.
var _ configService = &configServiceMock{}

// configServiceMock is a mock implementation of configService.
//
//	func TestSomethingThatUsesconfigService(t *testing.T) {
//
//		// make and configure a mocked configService
//		mockedconfigService := &configServiceMock{
//			GetFilenodeDefaultLimitBytesFunc: func() uint64 {
//				panic("mock out the GetFilenodeDefaultLimitBytes method")
//			},
//			InitFunc: func(a *app.App) error {
//				panic("mock out the Init method")
//			},
//			NameFunc: func() string {
//				panic("mock out the Name method")
//			},
//		}
//
//		// use mockedconfigService in code that requires configService
//		// and then make assertions.
//
//	}
type configServiceMock struct {
	// GetFilenodeDefaultLimitBytesFunc mocks the GetFilenodeDefaultLimitBytes method.
	GetFilenodeDefaultLimitBytesFunc func() uint64

	// InitFunc mocks the Init method.
	InitFunc func(a *app.App) error

	// NameFunc mocks the Name method.
	NameFunc func() string

	// calls tracks calls to the methods.
	calls struct {
		// GetFilenodeDefaultLimitBytes holds details about calls to the GetFilenodeDefaultLimitBytes method.
		GetFilenodeDefaultLimitBytes []struct {
		}
		// Init holds details about calls to the Init method.
		Init []struct {
			// A is the a argument value.
			A *app.App
		}
		// Name holds details about calls to the Name method.
		Name []struct {
		}
	}
	lockGetFilenodeDefaultLimitBytes sync.RWMutex
	lockInit                         sync.RWMutex
	lockName                         sync.RWMutex
}

// GetFilenodeDefaultLimitBytes calls GetFilenodeDefaultLimitBytesFunc.
func (mock *configServiceMock) GetFilenodeDefaultLimitBytes() uint64 {
	if mock.GetFilenodeDefaultLimitBytesFunc == nil {
		panic("configServiceMock.GetFilenodeDefaultLimitBytesFunc: method is nil but configService.GetFilenodeDefaultLimitBytes was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetFilenodeDefaultLimitBytes.Lock()
	mock.calls.GetFilenodeDefaultLimitBytes = append(mock.calls.GetFilenodeDefaultLimitBytes, callInfo)
	mock.lockGetFilenodeDefaultLimitBytes.Unlock()
	return mock.GetFilenodeDefaultLimitBytesFunc()
}

// GetFilenodeDefaultLimitBytesCalls gets all the calls that were made to GetFilenodeDefaultLimitBytes.
// Check the length with:
//
//	len(mockedconfigService.GetFilenodeDefaultLimitBytesCalls())
func (mock *configServiceMock) GetFilenodeDefaultLimitBytesCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetFilenodeDefaultLimitBytes.RLock()
	calls = mock.calls.GetFilenodeDefaultLimitBytes
	mock.lockGetFilenodeDefaultLimitBytes.RUnlock()
	return calls
}

// Init calls InitFunc.
func (mock *configServiceMock) Init(a *app.App) error {
	if mock.InitFunc == nil {
		panic("configServiceMock.InitFunc: method is nil but configService.Init was just called")
	}
	callInfo := struct {
		A *app.App
	}{
		A: a,
	}
	mock.lockInit.Lock()
	mock.calls.Init = append(mock.calls.Init, callInfo)
	mock.lockInit.Unlock()
	return mock.InitFunc(a)
}

// InitCalls gets all the calls that were made to Init.
// Check the length with:
//
//	len(mockedconfigService.InitCalls())
func (mock *configServiceMock) InitCalls() []struct {
	A *app.App
} {
	var calls []struct {
		A *app.App
	}
	mock.lockInit.RLock()
	calls = mock.calls.Init
	mock.lockInit.RUnlock()
	return calls
}

// Name calls NameFunc.
func (mock *configServiceMock) Name() string {
	if mock.NameFunc == nil {
		panic("configServiceMock.NameFunc: method is nil but configService.Name was just called")
	}
	callInfo := struct {
	}{}
	mock.lockName.Lock()
	mock.calls.Name = append(mock.calls.Name, callInfo)
	mock.lockName.Unlock()
	return mock.NameFunc()
}

// NameCalls gets all the calls that were made to Name.
// Check the length with:
//
//	len(mockedconfigService.NameCalls())
func (mock *configServiceMock) NameCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockName.RLock()
	calls = mock.calls.Name
	mock.lockName.RUnlock()
	return calls
}

// Ensure, that IndexServiceMock does implement IndexService.
// If this is not the case, regenerate this file with moq.
var _ IndexService = &IndexServiceMock{}

// IndexServiceMock is a mock implementation of IndexService.
//
//	func TestSomethingThatUsesIndexService(t *testing.T) {
//
//		// make and configure a mocked IndexService
//		mockedIndexService := &IndexServiceMock{
//			CloseFunc: func(ctx context.Context) error {
//				panic("mock out the Close method")
//			},
//			GetFileInfoFunc: func(spaceId string, fileIds ...string) ([]*fileproto.FileInfo, error) {
//				panic("mock out the GetFileInfo method")
//			},
//			GetSpaceFilesFunc: func(spaceId string) ([]string, error) {
//				panic("mock out the GetSpaceFiles method")
//			},
//			GroupInfoFunc: func(groupId string) GroupInfo {
//				panic("mock out the GroupInfo method")
//			},
//			HadCIDFunc: func(k cid.Cid) bool {
//				panic("mock out the HadCID method")
//			},
//			HasCIDInSpaceFunc: func(spaceId string, k cid.Cid) bool {
//				panic("mock out the HasCIDInSpace method")
//			},
//			InitFunc: func(a *app.App) error {
//				panic("mock out the Init method")
//			},
//			ModifyFunc: func(txn *badger.Txn, key index.Key, query *indexpb.Operation) error {
//				panic("mock out the Modify method")
//			},
//			NameFunc: func() string {
//				panic("mock out the Name method")
//			},
//			RunFunc: func(ctx context.Context) error {
//				panic("mock out the Run method")
//			},
//			SpaceInfoFunc: func(key index.Key) SpaceInfo {
//				panic("mock out the SpaceInfo method")
//			},
//		}
//
//		// use mockedIndexService in code that requires IndexService
//		// and then make assertions.
//
//	}
type IndexServiceMock struct {
	// CloseFunc mocks the Close method.
	CloseFunc func(ctx context.Context) error

	// GetFileInfoFunc mocks the GetFileInfo method.
	GetFileInfoFunc func(spaceId string, fileIds ...string) ([]*fileproto.FileInfo, error)

	// GetSpaceFilesFunc mocks the GetSpaceFiles method.
	GetSpaceFilesFunc func(spaceId string) ([]string, error)

	// GroupInfoFunc mocks the GroupInfo method.
	GroupInfoFunc func(groupId string) GroupInfo

	// HadCIDFunc mocks the HadCID method.
	HadCIDFunc func(k cid.Cid) bool

	// HasCIDInSpaceFunc mocks the HasCIDInSpace method.
	HasCIDInSpaceFunc func(spaceId string, k cid.Cid) bool

	// InitFunc mocks the Init method.
	InitFunc func(a *app.App) error

	// ModifyFunc mocks the Modify method.
	ModifyFunc func(txn *badger.Txn, key index.Key, query *indexpb.Operation) error

	// NameFunc mocks the Name method.
	NameFunc func() string

	// RunFunc mocks the Run method.
	RunFunc func(ctx context.Context) error

	// SpaceInfoFunc mocks the SpaceInfo method.
	SpaceInfoFunc func(key index.Key) SpaceInfo

	// calls tracks calls to the methods.
	calls struct {
		// Close holds details about calls to the Close method.
		Close []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
		// GetFileInfo holds details about calls to the GetFileInfo method.
		GetFileInfo []struct {
			// SpaceId is the spaceId argument value.
			SpaceId string
			// FileIds is the fileIds argument value.
			FileIds []string
		}
		// GetSpaceFiles holds details about calls to the GetSpaceFiles method.
		GetSpaceFiles []struct {
			// SpaceId is the spaceId argument value.
			SpaceId string
		}
		// GroupInfo holds details about calls to the GroupInfo method.
		GroupInfo []struct {
			// GroupId is the groupId argument value.
			GroupId string
		}
		// HadCID holds details about calls to the HadCID method.
		HadCID []struct {
			// K is the k argument value.
			K cid.Cid
		}
		// HasCIDInSpace holds details about calls to the HasCIDInSpace method.
		HasCIDInSpace []struct {
			// SpaceId is the spaceId argument value.
			SpaceId string
			// K is the k argument value.
			K cid.Cid
		}
		// Init holds details about calls to the Init method.
		Init []struct {
			// A is the a argument value.
			A *app.App
		}
		// Modify holds details about calls to the Modify method.
		Modify []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// Key is the key argument value.
			Key index.Key
			// Query is the query argument value.
			Query *indexpb.Operation
		}
		// Name holds details about calls to the Name method.
		Name []struct {
		}
		// Run holds details about calls to the Run method.
		Run []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
		// SpaceInfo holds details about calls to the SpaceInfo method.
		SpaceInfo []struct {
			// Key is the key argument value.
			Key index.Key
		}
	}
	lockClose         sync.RWMutex
	lockGetFileInfo   sync.RWMutex
	lockGetSpaceFiles sync.RWMutex
	lockGroupInfo     sync.RWMutex
	lockHadCID        sync.RWMutex
	lockHasCIDInSpace sync.RWMutex
	lockInit          sync.RWMutex
	lockModify        sync.RWMutex
	lockName          sync.RWMutex
	lockRun           sync.RWMutex
	lockSpaceInfo     sync.RWMutex
}

// Close calls CloseFunc.
func (mock *IndexServiceMock) Close(ctx context.Context) error {
	if mock.CloseFunc == nil {
		panic("IndexServiceMock.CloseFunc: method is nil but IndexService.Close was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockClose.Lock()
	mock.calls.Close = append(mock.calls.Close, callInfo)
	mock.lockClose.Unlock()
	return mock.CloseFunc(ctx)
}

// CloseCalls gets all the calls that were made to Close.
// Check the length with:
//
//	len(mockedIndexService.CloseCalls())
func (mock *IndexServiceMock) CloseCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockClose.RLock()
	calls = mock.calls.Close
	mock.lockClose.RUnlock()
	return calls
}

// GetFileInfo calls GetFileInfoFunc.
func (mock *IndexServiceMock) GetFileInfo(spaceId string, fileIds ...string) ([]*fileproto.FileInfo, error) {
	if mock.GetFileInfoFunc == nil {
		panic("IndexServiceMock.GetFileInfoFunc: method is nil but IndexService.GetFileInfo was just called")
	}
	callInfo := struct {
		SpaceId string
		FileIds []string
	}{
		SpaceId: spaceId,
		FileIds: fileIds,
	}
	mock.lockGetFileInfo.Lock()
	mock.calls.GetFileInfo = append(mock.calls.GetFileInfo, callInfo)
	mock.lockGetFileInfo.Unlock()
	return mock.GetFileInfoFunc(spaceId, fileIds...)
}

// GetFileInfoCalls gets all the calls that were made to GetFileInfo.
// Check the length with:
//
//	len(mockedIndexService.GetFileInfoCalls())
func (mock *IndexServiceMock) GetFileInfoCalls() []struct {
	SpaceId string
	FileIds []string
} {
	var calls []struct {
		SpaceId string
		FileIds []string
	}
	mock.lockGetFileInfo.RLock()
	calls = mock.calls.GetFileInfo
	mock.lockGetFileInfo.RUnlock()
	return calls
}

// GetSpaceFiles calls GetSpaceFilesFunc.
func (mock *IndexServiceMock) GetSpaceFiles(spaceId string) ([]string, error) {
	if mock.GetSpaceFilesFunc == nil {
		panic("IndexServiceMock.GetSpaceFilesFunc: method is nil but IndexService.GetSpaceFiles was just called")
	}
	callInfo := struct {
		SpaceId string
	}{
		SpaceId: spaceId,
	}
	mock.lockGetSpaceFiles.Lock()
	mock.calls.GetSpaceFiles = append(mock.calls.GetSpaceFiles, callInfo)
	mock.lockGetSpaceFiles.Unlock()
	return mock.GetSpaceFilesFunc(spaceId)
}

// GetSpaceFilesCalls gets all the calls that were made to GetSpaceFiles.
// Check the length with:
//
//	len(mockedIndexService.GetSpaceFilesCalls())
func (mock *IndexServiceMock) GetSpaceFilesCalls() []struct {
	SpaceId string
} {
	var calls []struct {
		SpaceId string
	}
	mock.lockGetSpaceFiles.RLock()
	calls = mock.calls.GetSpaceFiles
	mock.lockGetSpaceFiles.RUnlock()
	return calls
}

// GroupInfo calls GroupInfoFunc.
func (mock *IndexServiceMock) GroupInfo(groupId string) GroupInfo {
	if mock.GroupInfoFunc == nil {
		panic("IndexServiceMock.GroupInfoFunc: method is nil but IndexService.GroupInfo was just called")
	}
	callInfo := struct {
		GroupId string
	}{
		GroupId: groupId,
	}
	mock.lockGroupInfo.Lock()
	mock.calls.GroupInfo = append(mock.calls.GroupInfo, callInfo)
	mock.lockGroupInfo.Unlock()
	return mock.GroupInfoFunc(groupId)
}

// GroupInfoCalls gets all the calls that were made to GroupInfo.
// Check the length with:
//
//	len(mockedIndexService.GroupInfoCalls())
func (mock *IndexServiceMock) GroupInfoCalls() []struct {
	GroupId string
} {
	var calls []struct {
		GroupId string
	}
	mock.lockGroupInfo.RLock()
	calls = mock.calls.GroupInfo
	mock.lockGroupInfo.RUnlock()
	return calls
}

// HadCID calls HadCIDFunc.
func (mock *IndexServiceMock) HadCID(k cid.Cid) bool {
	if mock.HadCIDFunc == nil {
		panic("IndexServiceMock.HadCIDFunc: method is nil but IndexService.HadCID was just called")
	}
	callInfo := struct {
		K cid.Cid
	}{
		K: k,
	}
	mock.lockHadCID.Lock()
	mock.calls.HadCID = append(mock.calls.HadCID, callInfo)
	mock.lockHadCID.Unlock()
	return mock.HadCIDFunc(k)
}

// HadCIDCalls gets all the calls that were made to HadCID.
// Check the length with:
//
//	len(mockedIndexService.HadCIDCalls())
func (mock *IndexServiceMock) HadCIDCalls() []struct {
	K cid.Cid
} {
	var calls []struct {
		K cid.Cid
	}
	mock.lockHadCID.RLock()
	calls = mock.calls.HadCID
	mock.lockHadCID.RUnlock()
	return calls
}

// HasCIDInSpace calls HasCIDInSpaceFunc.
func (mock *IndexServiceMock) HasCIDInSpace(spaceId string, k cid.Cid) bool {
	if mock.HasCIDInSpaceFunc == nil {
		panic("IndexServiceMock.HasCIDInSpaceFunc: method is nil but IndexService.HasCIDInSpace was just called")
	}
	callInfo := struct {
		SpaceId string
		K       cid.Cid
	}{
		SpaceId: spaceId,
		K:       k,
	}
	mock.lockHasCIDInSpace.Lock()
	mock.calls.HasCIDInSpace = append(mock.calls.HasCIDInSpace, callInfo)
	mock.lockHasCIDInSpace.Unlock()
	return mock.HasCIDInSpaceFunc(spaceId, k)
}

// HasCIDInSpaceCalls gets all the calls that were made to HasCIDInSpace.
// Check the length with:
//
//	len(mockedIndexService.HasCIDInSpaceCalls())
func (mock *IndexServiceMock) HasCIDInSpaceCalls() []struct {
	SpaceId string
	K       cid.Cid
} {
	var calls []struct {
		SpaceId string
		K       cid.Cid
	}
	mock.lockHasCIDInSpace.RLock()
	calls = mock.calls.HasCIDInSpace
	mock.lockHasCIDInSpace.RUnlock()
	return calls
}

// Init calls InitFunc.
func (mock *IndexServiceMock) Init(a *app.App) error {
	if mock.InitFunc == nil {
		panic("IndexServiceMock.InitFunc: method is nil but IndexService.Init was just called")
	}
	callInfo := struct {
		A *app.App
	}{
		A: a,
	}
	mock.lockInit.Lock()
	mock.calls.Init = append(mock.calls.Init, callInfo)
	mock.lockInit.Unlock()
	return mock.InitFunc(a)
}

// InitCalls gets all the calls that were made to Init.
// Check the length with:
//
//	len(mockedIndexService.InitCalls())
func (mock *IndexServiceMock) InitCalls() []struct {
	A *app.App
} {
	var calls []struct {
		A *app.App
	}
	mock.lockInit.RLock()
	calls = mock.calls.Init
	mock.lockInit.RUnlock()
	return calls
}

// Modify calls ModifyFunc.
func (mock *IndexServiceMock) Modify(txn *badger.Txn, key index.Key, query *indexpb.Operation) error {
	if mock.ModifyFunc == nil {
		panic("IndexServiceMock.ModifyFunc: method is nil but IndexService.Modify was just called")
	}
	callInfo := struct {
		Txn   *badger.Txn
		Key   index.Key
		Query *indexpb.Operation
	}{
		Txn:   txn,
		Key:   key,
		Query: query,
	}
	mock.lockModify.Lock()
	mock.calls.Modify = append(mock.calls.Modify, callInfo)
	mock.lockModify.Unlock()
	return mock.ModifyFunc(txn, key, query)
}

// ModifyCalls gets all the calls that were made to Modify.
// Check the length with:
//
//	len(mockedIndexService.ModifyCalls())
func (mock *IndexServiceMock) ModifyCalls() []struct {
	Txn   *badger.Txn
	Key   index.Key
	Query *indexpb.Operation
} {
	var calls []struct {
		Txn   *badger.Txn
		Key   index.Key
		Query *indexpb.Operation
	}
	mock.lockModify.RLock()
	calls = mock.calls.Modify
	mock.lockModify.RUnlock()
	return calls
}

// Name calls NameFunc.
func (mock *IndexServiceMock) Name() string {
	if mock.NameFunc == nil {
		panic("IndexServiceMock.NameFunc: method is nil but IndexService.Name was just called")
	}
	callInfo := struct {
	}{}
	mock.lockName.Lock()
	mock.calls.Name = append(mock.calls.Name, callInfo)
	mock.lockName.Unlock()
	return mock.NameFunc()
}

// NameCalls gets all the calls that were made to Name.
// Check the length with:
//
//	len(mockedIndexService.NameCalls())
func (mock *IndexServiceMock) NameCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockName.RLock()
	calls = mock.calls.Name
	mock.lockName.RUnlock()
	return calls
}

// Run calls RunFunc.
func (mock *IndexServiceMock) Run(ctx context.Context) error {
	if mock.RunFunc == nil {
		panic("IndexServiceMock.RunFunc: method is nil but IndexService.Run was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockRun.Lock()
	mock.calls.Run = append(mock.calls.Run, callInfo)
	mock.lockRun.Unlock()
	return mock.RunFunc(ctx)
}

// RunCalls gets all the calls that were made to Run.
// Check the length with:
//
//	len(mockedIndexService.RunCalls())
func (mock *IndexServiceMock) RunCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockRun.RLock()
	calls = mock.calls.Run
	mock.lockRun.RUnlock()
	return calls
}

// SpaceInfo calls SpaceInfoFunc.
func (mock *IndexServiceMock) SpaceInfo(key index.Key) SpaceInfo {
	if mock.SpaceInfoFunc == nil {
		panic("IndexServiceMock.SpaceInfoFunc: method is nil but IndexService.SpaceInfo was just called")
	}
	callInfo := struct {
		Key index.Key
	}{
		Key: key,
	}
	mock.lockSpaceInfo.Lock()
	mock.calls.SpaceInfo = append(mock.calls.SpaceInfo, callInfo)
	mock.lockSpaceInfo.Unlock()
	return mock.SpaceInfoFunc(key)
}

// SpaceInfoCalls gets all the calls that were made to SpaceInfo.
// Check the length with:
//
//	len(mockedIndexService.SpaceInfoCalls())
func (mock *IndexServiceMock) SpaceInfoCalls() []struct {
	Key index.Key
} {
	var calls []struct {
		Key index.Key
	}
	mock.lockSpaceInfo.RLock()
	calls = mock.calls.SpaceInfo
	mock.lockSpaceInfo.RUnlock()
	return calls
}
