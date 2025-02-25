// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package lightfilenodestore

import (
	"github.com/anyproto/any-sync/app"
	"github.com/dgraph-io/badger/v4"
	blocks "github.com/ipfs/go-block-format"
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
//			GetFilenodeStoreDirFunc: func() string {
//				panic("mock out the GetFilenodeStoreDir method")
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

	// GetFilenodeStoreDirFunc mocks the GetFilenodeStoreDir method.
	GetFilenodeStoreDirFunc func() string

	// InitFunc mocks the Init method.
	InitFunc func(a *app.App) error

	// NameFunc mocks the Name method.
	NameFunc func() string

	// calls tracks calls to the methods.
	calls struct {
		// GetFilenodeDefaultLimitBytes holds details about calls to the GetFilenodeDefaultLimitBytes method.
		GetFilenodeDefaultLimitBytes []struct {
		}
		// GetFilenodeStoreDir holds details about calls to the GetFilenodeStoreDir method.
		GetFilenodeStoreDir []struct {
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
	lockGetFilenodeStoreDir          sync.RWMutex
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

// GetFilenodeStoreDir calls GetFilenodeStoreDirFunc.
func (mock *configServiceMock) GetFilenodeStoreDir() string {
	if mock.GetFilenodeStoreDirFunc == nil {
		panic("configServiceMock.GetFilenodeStoreDirFunc: method is nil but configService.GetFilenodeStoreDir was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetFilenodeStoreDir.Lock()
	mock.calls.GetFilenodeStoreDir = append(mock.calls.GetFilenodeStoreDir, callInfo)
	mock.lockGetFilenodeStoreDir.Unlock()
	return mock.GetFilenodeStoreDirFunc()
}

// GetFilenodeStoreDirCalls gets all the calls that were made to GetFilenodeStoreDir.
// Check the length with:
//
//	len(mockedconfigService.GetFilenodeStoreDirCalls())
func (mock *configServiceMock) GetFilenodeStoreDirCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetFilenodeStoreDir.RLock()
	calls = mock.calls.GetFilenodeStoreDir
	mock.lockGetFilenodeStoreDir.RUnlock()
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

// Ensure, that StoreServiceMock does implement StoreService.
// If this is not the case, regenerate this file with moq.
var _ StoreService = &StoreServiceMock{}

// StoreServiceMock is a mock implementation of StoreService.
//
//	func TestSomethingThatUsesStoreService(t *testing.T) {
//
//		// make and configure a mocked StoreService
//		mockedStoreService := &StoreServiceMock{
//			CreateLinkFileBlockFunc: func(txn *badger.Txn, spaceId string, fileId string, k cid.Cid) error {
//				panic("mock out the CreateLinkFileBlock method")
//			},
//			CreateLinkGroupSpaceFunc: func(txn *badger.Txn, groupId string, spaceId string) error {
//				panic("mock out the CreateLinkGroupSpace method")
//			},
//			GetBlockFunc: func(txn *badger.Txn, k cid.Cid) (*BlockObj, error) {
//				panic("mock out the GetBlock method")
//			},
//			GetFileFunc: func(txn *badger.Txn, spaceId string, fileId string) (*FileObj, error) {
//				panic("mock out the GetFile method")
//			},
//			GetGroupFunc: func(txn *badger.Txn, groupId string) (*GroupObj, error) {
//				panic("mock out the GetGroup method")
//			},
//			GetSpaceFunc: func(txn *badger.Txn, spaceId string) (*SpaceObj, error) {
//				panic("mock out the GetSpace method")
//			},
//			HadCIDFunc: func(txn *badger.Txn, k cid.Cid) (bool, error) {
//				panic("mock out the HadCID method")
//			},
//			HasCIDInSpaceFunc: func(txn *badger.Txn, spaceId string, k cid.Cid) (bool, error) {
//				panic("mock out the HasCIDInSpace method")
//			},
//			HasLinkFileBlockFunc: func(txn *badger.Txn, spaceId string, fileId string, k cid.Cid) (bool, error) {
//				panic("mock out the HasLinkFileBlock method")
//			},
//			InitFunc: func(a *app.App) error {
//				panic("mock out the Init method")
//			},
//			NameFunc: func() string {
//				panic("mock out the Name method")
//			},
//			PushBlockFunc: func(txn *badger.Txn, spaceId string, block blocks.Block) error {
//				panic("mock out the PushBlock method")
//			},
//			TxUpdateFunc: func(f func(txn *badger.Txn) error) error {
//				panic("mock out the TxUpdate method")
//			},
//			TxViewFunc: func(f func(txn *badger.Txn) error) error {
//				panic("mock out the TxView method")
//			},
//			WriteFileFunc: func(txn *badger.Txn, file *FileObj) error {
//				panic("mock out the WriteFile method")
//			},
//			WriteGroupFunc: func(txn *badger.Txn, obj *GroupObj) error {
//				panic("mock out the WriteGroup method")
//			},
//			WriteSpaceFunc: func(txn *badger.Txn, spaceObj *SpaceObj) error {
//				panic("mock out the WriteSpace method")
//			},
//		}
//
//		// use mockedStoreService in code that requires StoreService
//		// and then make assertions.
//
//	}
type StoreServiceMock struct {
	// CreateLinkFileBlockFunc mocks the CreateLinkFileBlock method.
	CreateLinkFileBlockFunc func(txn *badger.Txn, spaceId string, fileId string, k cid.Cid) error

	// CreateLinkGroupSpaceFunc mocks the CreateLinkGroupSpace method.
	CreateLinkGroupSpaceFunc func(txn *badger.Txn, groupId string, spaceId string) error

	// GetBlockFunc mocks the GetBlock method.
	GetBlockFunc func(txn *badger.Txn, k cid.Cid) (*BlockObj, error)

	// GetFileFunc mocks the GetFile method.
	GetFileFunc func(txn *badger.Txn, spaceId string, fileId string) (*FileObj, error)

	// GetGroupFunc mocks the GetGroup method.
	GetGroupFunc func(txn *badger.Txn, groupId string) (*GroupObj, error)

	// GetSpaceFunc mocks the GetSpace method.
	GetSpaceFunc func(txn *badger.Txn, spaceId string) (*SpaceObj, error)

	// HadCIDFunc mocks the HadCID method.
	HadCIDFunc func(txn *badger.Txn, k cid.Cid) (bool, error)

	// HasCIDInSpaceFunc mocks the HasCIDInSpace method.
	HasCIDInSpaceFunc func(txn *badger.Txn, spaceId string, k cid.Cid) (bool, error)

	// HasLinkFileBlockFunc mocks the HasLinkFileBlock method.
	HasLinkFileBlockFunc func(txn *badger.Txn, spaceId string, fileId string, k cid.Cid) (bool, error)

	// InitFunc mocks the Init method.
	InitFunc func(a *app.App) error

	// NameFunc mocks the Name method.
	NameFunc func() string

	// PushBlockFunc mocks the PushBlock method.
	PushBlockFunc func(txn *badger.Txn, spaceId string, block blocks.Block) error

	// TxUpdateFunc mocks the TxUpdate method.
	TxUpdateFunc func(f func(txn *badger.Txn) error) error

	// TxViewFunc mocks the TxView method.
	TxViewFunc func(f func(txn *badger.Txn) error) error

	// WriteFileFunc mocks the WriteFile method.
	WriteFileFunc func(txn *badger.Txn, file *FileObj) error

	// WriteGroupFunc mocks the WriteGroup method.
	WriteGroupFunc func(txn *badger.Txn, obj *GroupObj) error

	// WriteSpaceFunc mocks the WriteSpace method.
	WriteSpaceFunc func(txn *badger.Txn, spaceObj *SpaceObj) error

	// calls tracks calls to the methods.
	calls struct {
		// CreateLinkFileBlock holds details about calls to the CreateLinkFileBlock method.
		CreateLinkFileBlock []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// SpaceId is the spaceId argument value.
			SpaceId string
			// FileId is the fileId argument value.
			FileId string
			// K is the k argument value.
			K cid.Cid
		}
		// CreateLinkGroupSpace holds details about calls to the CreateLinkGroupSpace method.
		CreateLinkGroupSpace []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// GroupId is the groupId argument value.
			GroupId string
			// SpaceId is the spaceId argument value.
			SpaceId string
		}
		// GetBlock holds details about calls to the GetBlock method.
		GetBlock []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// K is the k argument value.
			K cid.Cid
		}
		// GetFile holds details about calls to the GetFile method.
		GetFile []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// SpaceId is the spaceId argument value.
			SpaceId string
			// FileId is the fileId argument value.
			FileId string
		}
		// GetGroup holds details about calls to the GetGroup method.
		GetGroup []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// GroupId is the groupId argument value.
			GroupId string
		}
		// GetSpace holds details about calls to the GetSpace method.
		GetSpace []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// SpaceId is the spaceId argument value.
			SpaceId string
		}
		// HadCID holds details about calls to the HadCID method.
		HadCID []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// K is the k argument value.
			K cid.Cid
		}
		// HasCIDInSpace holds details about calls to the HasCIDInSpace method.
		HasCIDInSpace []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// SpaceId is the spaceId argument value.
			SpaceId string
			// K is the k argument value.
			K cid.Cid
		}
		// HasLinkFileBlock holds details about calls to the HasLinkFileBlock method.
		HasLinkFileBlock []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// SpaceId is the spaceId argument value.
			SpaceId string
			// FileId is the fileId argument value.
			FileId string
			// K is the k argument value.
			K cid.Cid
		}
		// Init holds details about calls to the Init method.
		Init []struct {
			// A is the a argument value.
			A *app.App
		}
		// Name holds details about calls to the Name method.
		Name []struct {
		}
		// PushBlock holds details about calls to the PushBlock method.
		PushBlock []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// SpaceId is the spaceId argument value.
			SpaceId string
			// Block is the block argument value.
			Block blocks.Block
		}
		// TxUpdate holds details about calls to the TxUpdate method.
		TxUpdate []struct {
			// F is the f argument value.
			F func(txn *badger.Txn) error
		}
		// TxView holds details about calls to the TxView method.
		TxView []struct {
			// F is the f argument value.
			F func(txn *badger.Txn) error
		}
		// WriteFile holds details about calls to the WriteFile method.
		WriteFile []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// File is the file argument value.
			File *FileObj
		}
		// WriteGroup holds details about calls to the WriteGroup method.
		WriteGroup []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// Obj is the obj argument value.
			Obj *GroupObj
		}
		// WriteSpace holds details about calls to the WriteSpace method.
		WriteSpace []struct {
			// Txn is the txn argument value.
			Txn *badger.Txn
			// SpaceObj is the spaceObj argument value.
			SpaceObj *SpaceObj
		}
	}
	lockCreateLinkFileBlock  sync.RWMutex
	lockCreateLinkGroupSpace sync.RWMutex
	lockGetBlock             sync.RWMutex
	lockGetFile              sync.RWMutex
	lockGetGroup             sync.RWMutex
	lockGetSpace             sync.RWMutex
	lockHadCID               sync.RWMutex
	lockHasCIDInSpace        sync.RWMutex
	lockHasLinkFileBlock     sync.RWMutex
	lockInit                 sync.RWMutex
	lockName                 sync.RWMutex
	lockPushBlock            sync.RWMutex
	lockTxUpdate             sync.RWMutex
	lockTxView               sync.RWMutex
	lockWriteFile            sync.RWMutex
	lockWriteGroup           sync.RWMutex
	lockWriteSpace           sync.RWMutex
}

// CreateLinkFileBlock calls CreateLinkFileBlockFunc.
func (mock *StoreServiceMock) CreateLinkFileBlock(txn *badger.Txn, spaceId string, fileId string, k cid.Cid) error {
	if mock.CreateLinkFileBlockFunc == nil {
		panic("StoreServiceMock.CreateLinkFileBlockFunc: method is nil but StoreService.CreateLinkFileBlock was just called")
	}
	callInfo := struct {
		Txn     *badger.Txn
		SpaceId string
		FileId  string
		K       cid.Cid
	}{
		Txn:     txn,
		SpaceId: spaceId,
		FileId:  fileId,
		K:       k,
	}
	mock.lockCreateLinkFileBlock.Lock()
	mock.calls.CreateLinkFileBlock = append(mock.calls.CreateLinkFileBlock, callInfo)
	mock.lockCreateLinkFileBlock.Unlock()
	return mock.CreateLinkFileBlockFunc(txn, spaceId, fileId, k)
}

// CreateLinkFileBlockCalls gets all the calls that were made to CreateLinkFileBlock.
// Check the length with:
//
//	len(mockedStoreService.CreateLinkFileBlockCalls())
func (mock *StoreServiceMock) CreateLinkFileBlockCalls() []struct {
	Txn     *badger.Txn
	SpaceId string
	FileId  string
	K       cid.Cid
} {
	var calls []struct {
		Txn     *badger.Txn
		SpaceId string
		FileId  string
		K       cid.Cid
	}
	mock.lockCreateLinkFileBlock.RLock()
	calls = mock.calls.CreateLinkFileBlock
	mock.lockCreateLinkFileBlock.RUnlock()
	return calls
}

// CreateLinkGroupSpace calls CreateLinkGroupSpaceFunc.
func (mock *StoreServiceMock) CreateLinkGroupSpace(txn *badger.Txn, groupId string, spaceId string) error {
	if mock.CreateLinkGroupSpaceFunc == nil {
		panic("StoreServiceMock.CreateLinkGroupSpaceFunc: method is nil but StoreService.CreateLinkGroupSpace was just called")
	}
	callInfo := struct {
		Txn     *badger.Txn
		GroupId string
		SpaceId string
	}{
		Txn:     txn,
		GroupId: groupId,
		SpaceId: spaceId,
	}
	mock.lockCreateLinkGroupSpace.Lock()
	mock.calls.CreateLinkGroupSpace = append(mock.calls.CreateLinkGroupSpace, callInfo)
	mock.lockCreateLinkGroupSpace.Unlock()
	return mock.CreateLinkGroupSpaceFunc(txn, groupId, spaceId)
}

// CreateLinkGroupSpaceCalls gets all the calls that were made to CreateLinkGroupSpace.
// Check the length with:
//
//	len(mockedStoreService.CreateLinkGroupSpaceCalls())
func (mock *StoreServiceMock) CreateLinkGroupSpaceCalls() []struct {
	Txn     *badger.Txn
	GroupId string
	SpaceId string
} {
	var calls []struct {
		Txn     *badger.Txn
		GroupId string
		SpaceId string
	}
	mock.lockCreateLinkGroupSpace.RLock()
	calls = mock.calls.CreateLinkGroupSpace
	mock.lockCreateLinkGroupSpace.RUnlock()
	return calls
}

// GetBlock calls GetBlockFunc.
func (mock *StoreServiceMock) GetBlock(txn *badger.Txn, k cid.Cid) (*BlockObj, error) {
	if mock.GetBlockFunc == nil {
		panic("StoreServiceMock.GetBlockFunc: method is nil but StoreService.GetBlock was just called")
	}
	callInfo := struct {
		Txn *badger.Txn
		K   cid.Cid
	}{
		Txn: txn,
		K:   k,
	}
	mock.lockGetBlock.Lock()
	mock.calls.GetBlock = append(mock.calls.GetBlock, callInfo)
	mock.lockGetBlock.Unlock()
	return mock.GetBlockFunc(txn, k)
}

// GetBlockCalls gets all the calls that were made to GetBlock.
// Check the length with:
//
//	len(mockedStoreService.GetBlockCalls())
func (mock *StoreServiceMock) GetBlockCalls() []struct {
	Txn *badger.Txn
	K   cid.Cid
} {
	var calls []struct {
		Txn *badger.Txn
		K   cid.Cid
	}
	mock.lockGetBlock.RLock()
	calls = mock.calls.GetBlock
	mock.lockGetBlock.RUnlock()
	return calls
}

// GetFile calls GetFileFunc.
func (mock *StoreServiceMock) GetFile(txn *badger.Txn, spaceId string, fileId string) (*FileObj, error) {
	if mock.GetFileFunc == nil {
		panic("StoreServiceMock.GetFileFunc: method is nil but StoreService.GetFile was just called")
	}
	callInfo := struct {
		Txn     *badger.Txn
		SpaceId string
		FileId  string
	}{
		Txn:     txn,
		SpaceId: spaceId,
		FileId:  fileId,
	}
	mock.lockGetFile.Lock()
	mock.calls.GetFile = append(mock.calls.GetFile, callInfo)
	mock.lockGetFile.Unlock()
	return mock.GetFileFunc(txn, spaceId, fileId)
}

// GetFileCalls gets all the calls that were made to GetFile.
// Check the length with:
//
//	len(mockedStoreService.GetFileCalls())
func (mock *StoreServiceMock) GetFileCalls() []struct {
	Txn     *badger.Txn
	SpaceId string
	FileId  string
} {
	var calls []struct {
		Txn     *badger.Txn
		SpaceId string
		FileId  string
	}
	mock.lockGetFile.RLock()
	calls = mock.calls.GetFile
	mock.lockGetFile.RUnlock()
	return calls
}

// GetGroup calls GetGroupFunc.
func (mock *StoreServiceMock) GetGroup(txn *badger.Txn, groupId string) (*GroupObj, error) {
	if mock.GetGroupFunc == nil {
		panic("StoreServiceMock.GetGroupFunc: method is nil but StoreService.GetGroup was just called")
	}
	callInfo := struct {
		Txn     *badger.Txn
		GroupId string
	}{
		Txn:     txn,
		GroupId: groupId,
	}
	mock.lockGetGroup.Lock()
	mock.calls.GetGroup = append(mock.calls.GetGroup, callInfo)
	mock.lockGetGroup.Unlock()
	return mock.GetGroupFunc(txn, groupId)
}

// GetGroupCalls gets all the calls that were made to GetGroup.
// Check the length with:
//
//	len(mockedStoreService.GetGroupCalls())
func (mock *StoreServiceMock) GetGroupCalls() []struct {
	Txn     *badger.Txn
	GroupId string
} {
	var calls []struct {
		Txn     *badger.Txn
		GroupId string
	}
	mock.lockGetGroup.RLock()
	calls = mock.calls.GetGroup
	mock.lockGetGroup.RUnlock()
	return calls
}

// GetSpace calls GetSpaceFunc.
func (mock *StoreServiceMock) GetSpace(txn *badger.Txn, spaceId string) (*SpaceObj, error) {
	if mock.GetSpaceFunc == nil {
		panic("StoreServiceMock.GetSpaceFunc: method is nil but StoreService.GetSpace was just called")
	}
	callInfo := struct {
		Txn     *badger.Txn
		SpaceId string
	}{
		Txn:     txn,
		SpaceId: spaceId,
	}
	mock.lockGetSpace.Lock()
	mock.calls.GetSpace = append(mock.calls.GetSpace, callInfo)
	mock.lockGetSpace.Unlock()
	return mock.GetSpaceFunc(txn, spaceId)
}

// GetSpaceCalls gets all the calls that were made to GetSpace.
// Check the length with:
//
//	len(mockedStoreService.GetSpaceCalls())
func (mock *StoreServiceMock) GetSpaceCalls() []struct {
	Txn     *badger.Txn
	SpaceId string
} {
	var calls []struct {
		Txn     *badger.Txn
		SpaceId string
	}
	mock.lockGetSpace.RLock()
	calls = mock.calls.GetSpace
	mock.lockGetSpace.RUnlock()
	return calls
}

// HadCID calls HadCIDFunc.
func (mock *StoreServiceMock) HadCID(txn *badger.Txn, k cid.Cid) (bool, error) {
	if mock.HadCIDFunc == nil {
		panic("StoreServiceMock.HadCIDFunc: method is nil but StoreService.HadCID was just called")
	}
	callInfo := struct {
		Txn *badger.Txn
		K   cid.Cid
	}{
		Txn: txn,
		K:   k,
	}
	mock.lockHadCID.Lock()
	mock.calls.HadCID = append(mock.calls.HadCID, callInfo)
	mock.lockHadCID.Unlock()
	return mock.HadCIDFunc(txn, k)
}

// HadCIDCalls gets all the calls that were made to HadCID.
// Check the length with:
//
//	len(mockedStoreService.HadCIDCalls())
func (mock *StoreServiceMock) HadCIDCalls() []struct {
	Txn *badger.Txn
	K   cid.Cid
} {
	var calls []struct {
		Txn *badger.Txn
		K   cid.Cid
	}
	mock.lockHadCID.RLock()
	calls = mock.calls.HadCID
	mock.lockHadCID.RUnlock()
	return calls
}

// HasCIDInSpace calls HasCIDInSpaceFunc.
func (mock *StoreServiceMock) HasCIDInSpace(txn *badger.Txn, spaceId string, k cid.Cid) (bool, error) {
	if mock.HasCIDInSpaceFunc == nil {
		panic("StoreServiceMock.HasCIDInSpaceFunc: method is nil but StoreService.HasCIDInSpace was just called")
	}
	callInfo := struct {
		Txn     *badger.Txn
		SpaceId string
		K       cid.Cid
	}{
		Txn:     txn,
		SpaceId: spaceId,
		K:       k,
	}
	mock.lockHasCIDInSpace.Lock()
	mock.calls.HasCIDInSpace = append(mock.calls.HasCIDInSpace, callInfo)
	mock.lockHasCIDInSpace.Unlock()
	return mock.HasCIDInSpaceFunc(txn, spaceId, k)
}

// HasCIDInSpaceCalls gets all the calls that were made to HasCIDInSpace.
// Check the length with:
//
//	len(mockedStoreService.HasCIDInSpaceCalls())
func (mock *StoreServiceMock) HasCIDInSpaceCalls() []struct {
	Txn     *badger.Txn
	SpaceId string
	K       cid.Cid
} {
	var calls []struct {
		Txn     *badger.Txn
		SpaceId string
		K       cid.Cid
	}
	mock.lockHasCIDInSpace.RLock()
	calls = mock.calls.HasCIDInSpace
	mock.lockHasCIDInSpace.RUnlock()
	return calls
}

// HasLinkFileBlock calls HasLinkFileBlockFunc.
func (mock *StoreServiceMock) HasLinkFileBlock(txn *badger.Txn, spaceId string, fileId string, k cid.Cid) (bool, error) {
	if mock.HasLinkFileBlockFunc == nil {
		panic("StoreServiceMock.HasLinkFileBlockFunc: method is nil but StoreService.HasLinkFileBlock was just called")
	}
	callInfo := struct {
		Txn     *badger.Txn
		SpaceId string
		FileId  string
		K       cid.Cid
	}{
		Txn:     txn,
		SpaceId: spaceId,
		FileId:  fileId,
		K:       k,
	}
	mock.lockHasLinkFileBlock.Lock()
	mock.calls.HasLinkFileBlock = append(mock.calls.HasLinkFileBlock, callInfo)
	mock.lockHasLinkFileBlock.Unlock()
	return mock.HasLinkFileBlockFunc(txn, spaceId, fileId, k)
}

// HasLinkFileBlockCalls gets all the calls that were made to HasLinkFileBlock.
// Check the length with:
//
//	len(mockedStoreService.HasLinkFileBlockCalls())
func (mock *StoreServiceMock) HasLinkFileBlockCalls() []struct {
	Txn     *badger.Txn
	SpaceId string
	FileId  string
	K       cid.Cid
} {
	var calls []struct {
		Txn     *badger.Txn
		SpaceId string
		FileId  string
		K       cid.Cid
	}
	mock.lockHasLinkFileBlock.RLock()
	calls = mock.calls.HasLinkFileBlock
	mock.lockHasLinkFileBlock.RUnlock()
	return calls
}

// Init calls InitFunc.
func (mock *StoreServiceMock) Init(a *app.App) error {
	if mock.InitFunc == nil {
		panic("StoreServiceMock.InitFunc: method is nil but StoreService.Init was just called")
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
//	len(mockedStoreService.InitCalls())
func (mock *StoreServiceMock) InitCalls() []struct {
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
func (mock *StoreServiceMock) Name() string {
	if mock.NameFunc == nil {
		panic("StoreServiceMock.NameFunc: method is nil but StoreService.Name was just called")
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
//	len(mockedStoreService.NameCalls())
func (mock *StoreServiceMock) NameCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockName.RLock()
	calls = mock.calls.Name
	mock.lockName.RUnlock()
	return calls
}

// PushBlock calls PushBlockFunc.
func (mock *StoreServiceMock) PushBlock(txn *badger.Txn, spaceId string, block blocks.Block) error {
	if mock.PushBlockFunc == nil {
		panic("StoreServiceMock.PushBlockFunc: method is nil but StoreService.PushBlock was just called")
	}
	callInfo := struct {
		Txn     *badger.Txn
		SpaceId string
		Block   blocks.Block
	}{
		Txn:     txn,
		SpaceId: spaceId,
		Block:   block,
	}
	mock.lockPushBlock.Lock()
	mock.calls.PushBlock = append(mock.calls.PushBlock, callInfo)
	mock.lockPushBlock.Unlock()
	return mock.PushBlockFunc(txn, spaceId, block)
}

// PushBlockCalls gets all the calls that were made to PushBlock.
// Check the length with:
//
//	len(mockedStoreService.PushBlockCalls())
func (mock *StoreServiceMock) PushBlockCalls() []struct {
	Txn     *badger.Txn
	SpaceId string
	Block   blocks.Block
} {
	var calls []struct {
		Txn     *badger.Txn
		SpaceId string
		Block   blocks.Block
	}
	mock.lockPushBlock.RLock()
	calls = mock.calls.PushBlock
	mock.lockPushBlock.RUnlock()
	return calls
}

// TxUpdate calls TxUpdateFunc.
func (mock *StoreServiceMock) TxUpdate(f func(txn *badger.Txn) error) error {
	if mock.TxUpdateFunc == nil {
		panic("StoreServiceMock.TxUpdateFunc: method is nil but StoreService.TxUpdate was just called")
	}
	callInfo := struct {
		F func(txn *badger.Txn) error
	}{
		F: f,
	}
	mock.lockTxUpdate.Lock()
	mock.calls.TxUpdate = append(mock.calls.TxUpdate, callInfo)
	mock.lockTxUpdate.Unlock()
	return mock.TxUpdateFunc(f)
}

// TxUpdateCalls gets all the calls that were made to TxUpdate.
// Check the length with:
//
//	len(mockedStoreService.TxUpdateCalls())
func (mock *StoreServiceMock) TxUpdateCalls() []struct {
	F func(txn *badger.Txn) error
} {
	var calls []struct {
		F func(txn *badger.Txn) error
	}
	mock.lockTxUpdate.RLock()
	calls = mock.calls.TxUpdate
	mock.lockTxUpdate.RUnlock()
	return calls
}

// TxView calls TxViewFunc.
func (mock *StoreServiceMock) TxView(f func(txn *badger.Txn) error) error {
	if mock.TxViewFunc == nil {
		panic("StoreServiceMock.TxViewFunc: method is nil but StoreService.TxView was just called")
	}
	callInfo := struct {
		F func(txn *badger.Txn) error
	}{
		F: f,
	}
	mock.lockTxView.Lock()
	mock.calls.TxView = append(mock.calls.TxView, callInfo)
	mock.lockTxView.Unlock()
	return mock.TxViewFunc(f)
}

// TxViewCalls gets all the calls that were made to TxView.
// Check the length with:
//
//	len(mockedStoreService.TxViewCalls())
func (mock *StoreServiceMock) TxViewCalls() []struct {
	F func(txn *badger.Txn) error
} {
	var calls []struct {
		F func(txn *badger.Txn) error
	}
	mock.lockTxView.RLock()
	calls = mock.calls.TxView
	mock.lockTxView.RUnlock()
	return calls
}

// WriteFile calls WriteFileFunc.
func (mock *StoreServiceMock) WriteFile(txn *badger.Txn, file *FileObj) error {
	if mock.WriteFileFunc == nil {
		panic("StoreServiceMock.WriteFileFunc: method is nil but StoreService.WriteFile was just called")
	}
	callInfo := struct {
		Txn  *badger.Txn
		File *FileObj
	}{
		Txn:  txn,
		File: file,
	}
	mock.lockWriteFile.Lock()
	mock.calls.WriteFile = append(mock.calls.WriteFile, callInfo)
	mock.lockWriteFile.Unlock()
	return mock.WriteFileFunc(txn, file)
}

// WriteFileCalls gets all the calls that were made to WriteFile.
// Check the length with:
//
//	len(mockedStoreService.WriteFileCalls())
func (mock *StoreServiceMock) WriteFileCalls() []struct {
	Txn  *badger.Txn
	File *FileObj
} {
	var calls []struct {
		Txn  *badger.Txn
		File *FileObj
	}
	mock.lockWriteFile.RLock()
	calls = mock.calls.WriteFile
	mock.lockWriteFile.RUnlock()
	return calls
}

// WriteGroup calls WriteGroupFunc.
func (mock *StoreServiceMock) WriteGroup(txn *badger.Txn, obj *GroupObj) error {
	if mock.WriteGroupFunc == nil {
		panic("StoreServiceMock.WriteGroupFunc: method is nil but StoreService.WriteGroup was just called")
	}
	callInfo := struct {
		Txn *badger.Txn
		Obj *GroupObj
	}{
		Txn: txn,
		Obj: obj,
	}
	mock.lockWriteGroup.Lock()
	mock.calls.WriteGroup = append(mock.calls.WriteGroup, callInfo)
	mock.lockWriteGroup.Unlock()
	return mock.WriteGroupFunc(txn, obj)
}

// WriteGroupCalls gets all the calls that were made to WriteGroup.
// Check the length with:
//
//	len(mockedStoreService.WriteGroupCalls())
func (mock *StoreServiceMock) WriteGroupCalls() []struct {
	Txn *badger.Txn
	Obj *GroupObj
} {
	var calls []struct {
		Txn *badger.Txn
		Obj *GroupObj
	}
	mock.lockWriteGroup.RLock()
	calls = mock.calls.WriteGroup
	mock.lockWriteGroup.RUnlock()
	return calls
}

// WriteSpace calls WriteSpaceFunc.
func (mock *StoreServiceMock) WriteSpace(txn *badger.Txn, spaceObj *SpaceObj) error {
	if mock.WriteSpaceFunc == nil {
		panic("StoreServiceMock.WriteSpaceFunc: method is nil but StoreService.WriteSpace was just called")
	}
	callInfo := struct {
		Txn      *badger.Txn
		SpaceObj *SpaceObj
	}{
		Txn:      txn,
		SpaceObj: spaceObj,
	}
	mock.lockWriteSpace.Lock()
	mock.calls.WriteSpace = append(mock.calls.WriteSpace, callInfo)
	mock.lockWriteSpace.Unlock()
	return mock.WriteSpaceFunc(txn, spaceObj)
}

// WriteSpaceCalls gets all the calls that were made to WriteSpace.
// Check the length with:
//
//	len(mockedStoreService.WriteSpaceCalls())
func (mock *StoreServiceMock) WriteSpaceCalls() []struct {
	Txn      *badger.Txn
	SpaceObj *SpaceObj
} {
	var calls []struct {
		Txn      *badger.Txn
		SpaceObj *SpaceObj
	}
	mock.lockWriteSpace.RLock()
	calls = mock.calls.WriteSpace
	mock.lockWriteSpace.RUnlock()
	return calls
}
