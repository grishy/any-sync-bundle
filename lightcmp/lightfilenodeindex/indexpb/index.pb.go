// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.29.3
// source: index.proto

package indexpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Composite key for identifying the space within a group.
type Key struct {
	state                  protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_GroupId     *string                `protobuf:"bytes,1,opt,name=group_id,json=groupId"`
	xxx_hidden_SpaceId     *string                `protobuf:"bytes,2,opt,name=space_id,json=spaceId"`
	XXX_raceDetectHookData protoimpl.RaceDetectHookData
	XXX_presence           [1]uint32
	unknownFields          protoimpl.UnknownFields
	sizeCache              protoimpl.SizeCache
}

func (x *Key) Reset() {
	*x = Key{}
	mi := &file_index_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Key) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Key) ProtoMessage() {}

func (x *Key) ProtoReflect() protoreflect.Message {
	mi := &file_index_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Key) GetGroupId() string {
	if x != nil {
		if x.xxx_hidden_GroupId != nil {
			return *x.xxx_hidden_GroupId
		}
		return ""
	}
	return ""
}

func (x *Key) GetSpaceId() string {
	if x != nil {
		if x.xxx_hidden_SpaceId != nil {
			return *x.xxx_hidden_SpaceId
		}
		return ""
	}
	return ""
}

func (x *Key) SetGroupId(v string) {
	x.xxx_hidden_GroupId = &v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 0, 2)
}

func (x *Key) SetSpaceId(v string) {
	x.xxx_hidden_SpaceId = &v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 1, 2)
}

func (x *Key) HasGroupId() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 0)
}

func (x *Key) HasSpaceId() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 1)
}

func (x *Key) ClearGroupId() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 0)
	x.xxx_hidden_GroupId = nil
}

func (x *Key) ClearSpaceId() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 1)
	x.xxx_hidden_SpaceId = nil
}

type Key_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	GroupId *string
	SpaceId *string
}

func (b0 Key_builder) Build() *Key {
	m0 := &Key{}
	b, x := &b0, m0
	_, _ = b, x
	if b.GroupId != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 0, 2)
		x.xxx_hidden_GroupId = b.GroupId
	}
	if b.SpaceId != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 1, 2)
		x.xxx_hidden_SpaceId = b.SpaceId
	}
	return m0
}

// Define parameters for binding a file.
type FileBindOperation struct {
	state                  protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_FileId      *string                `protobuf:"bytes,1,opt,name=file_id,json=fileId"`
	xxx_hidden_BlockCids   []string               `protobuf:"bytes,2,rep,name=block_cids,json=blockCids"`
	xxx_hidden_DataSizes   []int32                `protobuf:"varint,3,rep,packed,name=data_sizes,json=dataSizes"`
	XXX_raceDetectHookData protoimpl.RaceDetectHookData
	XXX_presence           [1]uint32
	unknownFields          protoimpl.UnknownFields
	sizeCache              protoimpl.SizeCache
}

func (x *FileBindOperation) Reset() {
	*x = FileBindOperation{}
	mi := &file_index_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FileBindOperation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileBindOperation) ProtoMessage() {}

func (x *FileBindOperation) ProtoReflect() protoreflect.Message {
	mi := &file_index_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *FileBindOperation) GetFileId() string {
	if x != nil {
		if x.xxx_hidden_FileId != nil {
			return *x.xxx_hidden_FileId
		}
		return ""
	}
	return ""
}

func (x *FileBindOperation) GetBlockCids() []string {
	if x != nil {
		return x.xxx_hidden_BlockCids
	}
	return nil
}

func (x *FileBindOperation) GetDataSizes() []int32 {
	if x != nil {
		return x.xxx_hidden_DataSizes
	}
	return nil
}

func (x *FileBindOperation) SetFileId(v string) {
	x.xxx_hidden_FileId = &v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 0, 3)
}

func (x *FileBindOperation) SetBlockCids(v []string) {
	x.xxx_hidden_BlockCids = v
}

func (x *FileBindOperation) SetDataSizes(v []int32) {
	x.xxx_hidden_DataSizes = v
}

func (x *FileBindOperation) HasFileId() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 0)
}

func (x *FileBindOperation) ClearFileId() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 0)
	x.xxx_hidden_FileId = nil
}

type FileBindOperation_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	FileId *string
	// List of block CIDs associated with this file.
	BlockCids []string
	// Data size for each CID, corresponding to the block_cids array
	DataSizes []int32
}

func (b0 FileBindOperation_builder) Build() *FileBindOperation {
	m0 := &FileBindOperation{}
	b, x := &b0, m0
	_, _ = b, x
	if b.FileId != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 0, 3)
		x.xxx_hidden_FileId = b.FileId
	}
	x.xxx_hidden_BlockCids = b.BlockCids
	x.xxx_hidden_DataSizes = b.DataSizes
	return m0
}

// Define parameters for unbinding a file.
type FileUnbindOperation struct {
	state              protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_FileIds []string               `protobuf:"bytes,1,rep,name=file_ids,json=fileIds"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *FileUnbindOperation) Reset() {
	*x = FileUnbindOperation{}
	mi := &file_index_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FileUnbindOperation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileUnbindOperation) ProtoMessage() {}

func (x *FileUnbindOperation) ProtoReflect() protoreflect.Message {
	mi := &file_index_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *FileUnbindOperation) GetFileIds() []string {
	if x != nil {
		return x.xxx_hidden_FileIds
	}
	return nil
}

func (x *FileUnbindOperation) SetFileIds(v []string) {
	x.xxx_hidden_FileIds = v
}

type FileUnbindOperation_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	FileIds []string
}

func (b0 FileUnbindOperation_builder) Build() *FileUnbindOperation {
	m0 := &FileUnbindOperation{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_FileIds = b.FileIds
	return m0
}

// Set the storage limit for an account
type AccountLimitSetOperation struct {
	state                  protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Limit       uint64                 `protobuf:"varint,2,opt,name=limit"`
	XXX_raceDetectHookData protoimpl.RaceDetectHookData
	XXX_presence           [1]uint32
	unknownFields          protoimpl.UnknownFields
	sizeCache              protoimpl.SizeCache
}

func (x *AccountLimitSetOperation) Reset() {
	*x = AccountLimitSetOperation{}
	mi := &file_index_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AccountLimitSetOperation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AccountLimitSetOperation) ProtoMessage() {}

func (x *AccountLimitSetOperation) ProtoReflect() protoreflect.Message {
	mi := &file_index_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *AccountLimitSetOperation) GetLimit() uint64 {
	if x != nil {
		return x.xxx_hidden_Limit
	}
	return 0
}

func (x *AccountLimitSetOperation) SetLimit(v uint64) {
	x.xxx_hidden_Limit = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 0, 1)
}

func (x *AccountLimitSetOperation) HasLimit() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 0)
}

func (x *AccountLimitSetOperation) ClearLimit() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 0)
	x.xxx_hidden_Limit = 0
}

type AccountLimitSetOperation_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Limit *uint64
}

func (b0 AccountLimitSetOperation_builder) Build() *AccountLimitSetOperation {
	m0 := &AccountLimitSetOperation{}
	b, x := &b0, m0
	_, _ = b, x
	if b.Limit != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 0, 1)
		x.xxx_hidden_Limit = *b.Limit
	}
	return m0
}

// Set the storage limit for a space
type SpaceLimitSetOperation struct {
	state                  protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Limit       uint64                 `protobuf:"varint,2,opt,name=limit"`
	XXX_raceDetectHookData protoimpl.RaceDetectHookData
	XXX_presence           [1]uint32
	unknownFields          protoimpl.UnknownFields
	sizeCache              protoimpl.SizeCache
}

func (x *SpaceLimitSetOperation) Reset() {
	*x = SpaceLimitSetOperation{}
	mi := &file_index_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SpaceLimitSetOperation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SpaceLimitSetOperation) ProtoMessage() {}

func (x *SpaceLimitSetOperation) ProtoReflect() protoreflect.Message {
	mi := &file_index_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *SpaceLimitSetOperation) GetLimit() uint64 {
	if x != nil {
		return x.xxx_hidden_Limit
	}
	return 0
}

func (x *SpaceLimitSetOperation) SetLimit(v uint64) {
	x.xxx_hidden_Limit = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 0, 1)
}

func (x *SpaceLimitSetOperation) HasLimit() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 0)
}

func (x *SpaceLimitSetOperation) ClearLimit() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 0)
	x.xxx_hidden_Limit = 0
}

type SpaceLimitSetOperation_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Limit *uint64
}

func (b0 SpaceLimitSetOperation_builder) Build() *SpaceLimitSetOperation {
	m0 := &SpaceLimitSetOperation{}
	b, x := &b0, m0
	_, _ = b, x
	if b.Limit != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 0, 1)
		x.xxx_hidden_Limit = *b.Limit
	}
	return m0
}

// Operation encapsulates all possible index modification operations.
type Operation struct {
	state         protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Op isOperation_Op         `protobuf_oneof:"op"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Operation) Reset() {
	*x = Operation{}
	mi := &file_index_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Operation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Operation) ProtoMessage() {}

func (x *Operation) ProtoReflect() protoreflect.Message {
	mi := &file_index_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Operation) GetBindFile() *FileBindOperation {
	if x != nil {
		if x, ok := x.xxx_hidden_Op.(*operation_BindFile); ok {
			return x.BindFile
		}
	}
	return nil
}

func (x *Operation) GetUnbindFile() *FileUnbindOperation {
	if x != nil {
		if x, ok := x.xxx_hidden_Op.(*operation_UnbindFile); ok {
			return x.UnbindFile
		}
	}
	return nil
}

func (x *Operation) GetAccountLimitSet() *AccountLimitSetOperation {
	if x != nil {
		if x, ok := x.xxx_hidden_Op.(*operation_AccountLimitSet); ok {
			return x.AccountLimitSet
		}
	}
	return nil
}

func (x *Operation) GetSpaceLimitSet() *SpaceLimitSetOperation {
	if x != nil {
		if x, ok := x.xxx_hidden_Op.(*operation_SpaceLimitSet); ok {
			return x.SpaceLimitSet
		}
	}
	return nil
}

func (x *Operation) SetBindFile(v *FileBindOperation) {
	if v == nil {
		x.xxx_hidden_Op = nil
		return
	}
	x.xxx_hidden_Op = &operation_BindFile{v}
}

func (x *Operation) SetUnbindFile(v *FileUnbindOperation) {
	if v == nil {
		x.xxx_hidden_Op = nil
		return
	}
	x.xxx_hidden_Op = &operation_UnbindFile{v}
}

func (x *Operation) SetAccountLimitSet(v *AccountLimitSetOperation) {
	if v == nil {
		x.xxx_hidden_Op = nil
		return
	}
	x.xxx_hidden_Op = &operation_AccountLimitSet{v}
}

func (x *Operation) SetSpaceLimitSet(v *SpaceLimitSetOperation) {
	if v == nil {
		x.xxx_hidden_Op = nil
		return
	}
	x.xxx_hidden_Op = &operation_SpaceLimitSet{v}
}

func (x *Operation) HasOp() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Op != nil
}

func (x *Operation) HasBindFile() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Op.(*operation_BindFile)
	return ok
}

func (x *Operation) HasUnbindFile() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Op.(*operation_UnbindFile)
	return ok
}

func (x *Operation) HasAccountLimitSet() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Op.(*operation_AccountLimitSet)
	return ok
}

func (x *Operation) HasSpaceLimitSet() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Op.(*operation_SpaceLimitSet)
	return ok
}

func (x *Operation) ClearOp() {
	x.xxx_hidden_Op = nil
}

func (x *Operation) ClearBindFile() {
	if _, ok := x.xxx_hidden_Op.(*operation_BindFile); ok {
		x.xxx_hidden_Op = nil
	}
}

func (x *Operation) ClearUnbindFile() {
	if _, ok := x.xxx_hidden_Op.(*operation_UnbindFile); ok {
		x.xxx_hidden_Op = nil
	}
}

func (x *Operation) ClearAccountLimitSet() {
	if _, ok := x.xxx_hidden_Op.(*operation_AccountLimitSet); ok {
		x.xxx_hidden_Op = nil
	}
}

func (x *Operation) ClearSpaceLimitSet() {
	if _, ok := x.xxx_hidden_Op.(*operation_SpaceLimitSet); ok {
		x.xxx_hidden_Op = nil
	}
}

const Operation_Op_not_set_case case_Operation_Op = 0
const Operation_BindFile_case case_Operation_Op = 1
const Operation_UnbindFile_case case_Operation_Op = 2
const Operation_AccountLimitSet_case case_Operation_Op = 3
const Operation_SpaceLimitSet_case case_Operation_Op = 4

func (x *Operation) WhichOp() case_Operation_Op {
	if x == nil {
		return Operation_Op_not_set_case
	}
	switch x.xxx_hidden_Op.(type) {
	case *operation_BindFile:
		return Operation_BindFile_case
	case *operation_UnbindFile:
		return Operation_UnbindFile_case
	case *operation_AccountLimitSet:
		return Operation_AccountLimitSet_case
	case *operation_SpaceLimitSet:
		return Operation_SpaceLimitSet_case
	default:
		return Operation_Op_not_set_case
	}
}

type Operation_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Fields of oneof xxx_hidden_Op:
	BindFile        *FileBindOperation
	UnbindFile      *FileUnbindOperation
	AccountLimitSet *AccountLimitSetOperation
	SpaceLimitSet   *SpaceLimitSetOperation
	// -- end of xxx_hidden_Op
}

func (b0 Operation_builder) Build() *Operation {
	m0 := &Operation{}
	b, x := &b0, m0
	_, _ = b, x
	if b.BindFile != nil {
		x.xxx_hidden_Op = &operation_BindFile{b.BindFile}
	}
	if b.UnbindFile != nil {
		x.xxx_hidden_Op = &operation_UnbindFile{b.UnbindFile}
	}
	if b.AccountLimitSet != nil {
		x.xxx_hidden_Op = &operation_AccountLimitSet{b.AccountLimitSet}
	}
	if b.SpaceLimitSet != nil {
		x.xxx_hidden_Op = &operation_SpaceLimitSet{b.SpaceLimitSet}
	}
	return m0
}

type case_Operation_Op protoreflect.FieldNumber

func (x case_Operation_Op) String() string {
	md := file_index_proto_msgTypes[5].Descriptor()
	if x == 0 {
		return "not set"
	}
	return protoimpl.X.MessageFieldStringOf(md, protoreflect.FieldNumber(x))
}

type isOperation_Op interface {
	isOperation_Op()
}

type operation_BindFile struct {
	BindFile *FileBindOperation `protobuf:"bytes,1,opt,name=bind_file,json=bindFile,oneof"`
}

type operation_UnbindFile struct {
	UnbindFile *FileUnbindOperation `protobuf:"bytes,2,opt,name=unbind_file,json=unbindFile,oneof"`
}

type operation_AccountLimitSet struct {
	AccountLimitSet *AccountLimitSetOperation `protobuf:"bytes,3,opt,name=account_limit_set,json=accountLimitSet,oneof"`
}

type operation_SpaceLimitSet struct {
	SpaceLimitSet *SpaceLimitSetOperation `protobuf:"bytes,4,opt,name=space_limit_set,json=spaceLimitSet,oneof"` // Add other operations here as needed.
}

func (*operation_BindFile) isOperation_Op() {}

func (*operation_UnbindFile) isOperation_Op() {}

func (*operation_AccountLimitSet) isOperation_Op() {}

func (*operation_SpaceLimitSet) isOperation_Op() {}

// WALRecord represents a single write-ahead log record.
type WALRecord struct {
	state                  protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Key         *Key                   `protobuf:"bytes,1,opt,name=key"`
	xxx_hidden_Op          *Operation             `protobuf:"bytes,2,opt,name=op"`
	xxx_hidden_Timestamp   int64                  `protobuf:"varint,3,opt,name=timestamp"`
	XXX_raceDetectHookData protoimpl.RaceDetectHookData
	XXX_presence           [1]uint32
	unknownFields          protoimpl.UnknownFields
	sizeCache              protoimpl.SizeCache
}

func (x *WALRecord) Reset() {
	*x = WALRecord{}
	mi := &file_index_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *WALRecord) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WALRecord) ProtoMessage() {}

func (x *WALRecord) ProtoReflect() protoreflect.Message {
	mi := &file_index_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *WALRecord) GetKey() *Key {
	if x != nil {
		return x.xxx_hidden_Key
	}
	return nil
}

func (x *WALRecord) GetOp() *Operation {
	if x != nil {
		return x.xxx_hidden_Op
	}
	return nil
}

func (x *WALRecord) GetTimestamp() int64 {
	if x != nil {
		return x.xxx_hidden_Timestamp
	}
	return 0
}

func (x *WALRecord) SetKey(v *Key) {
	x.xxx_hidden_Key = v
}

func (x *WALRecord) SetOp(v *Operation) {
	x.xxx_hidden_Op = v
}

func (x *WALRecord) SetTimestamp(v int64) {
	x.xxx_hidden_Timestamp = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 2, 3)
}

func (x *WALRecord) HasKey() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Key != nil
}

func (x *WALRecord) HasOp() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Op != nil
}

func (x *WALRecord) HasTimestamp() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 2)
}

func (x *WALRecord) ClearKey() {
	x.xxx_hidden_Key = nil
}

func (x *WALRecord) ClearOp() {
	x.xxx_hidden_Op = nil
}

func (x *WALRecord) ClearTimestamp() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 2)
	x.xxx_hidden_Timestamp = 0
}

type WALRecord_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Key       *Key
	Op        *Operation
	Timestamp *int64
}

func (b0 WALRecord_builder) Build() *WALRecord {
	m0 := &WALRecord{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Key = b.Key
	x.xxx_hidden_Op = b.Op
	if b.Timestamp != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 2, 3)
		x.xxx_hidden_Timestamp = *b.Timestamp
	}
	return m0
}

var File_index_proto protoreflect.FileDescriptor

var file_index_proto_rawDesc = string([]byte{
	0x0a, 0x0b, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x69,
	0x6e, 0x64, 0x65, 0x78, 0x70, 0x62, 0x22, 0x3b, 0x0a, 0x03, 0x4b, 0x65, 0x79, 0x12, 0x19, 0x0a,
	0x08, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x49, 0x64, 0x12, 0x19, 0x0a, 0x08, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x49, 0x64, 0x22, 0x6a, 0x0a, 0x11, 0x46, 0x69, 0x6c, 0x65, 0x42, 0x69, 0x6e, 0x64, 0x4f,
	0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x17, 0x0a, 0x07, 0x66, 0x69, 0x6c, 0x65,
	0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x66, 0x69, 0x6c, 0x65, 0x49,
	0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x63, 0x69, 0x64, 0x73, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x43, 0x69, 0x64, 0x73,
	0x12, 0x1d, 0x0a, 0x0a, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x73, 0x18, 0x03,
	0x20, 0x03, 0x28, 0x05, 0x52, 0x09, 0x64, 0x61, 0x74, 0x61, 0x53, 0x69, 0x7a, 0x65, 0x73, 0x22,
	0x30, 0x0a, 0x13, 0x46, 0x69, 0x6c, 0x65, 0x55, 0x6e, 0x62, 0x69, 0x6e, 0x64, 0x4f, 0x70, 0x65,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x19, 0x0a, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x69,
	0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x66, 0x69, 0x6c, 0x65, 0x49, 0x64,
	0x73, 0x22, 0x30, 0x0a, 0x18, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x4c, 0x69, 0x6d, 0x69,
	0x74, 0x53, 0x65, 0x74, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x0a,
	0x05, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x6c, 0x69,
	0x6d, 0x69, 0x74, 0x22, 0x2e, 0x0a, 0x16, 0x53, 0x70, 0x61, 0x63, 0x65, 0x4c, 0x69, 0x6d, 0x69,
	0x74, 0x53, 0x65, 0x74, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x0a,
	0x05, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x6c, 0x69,
	0x6d, 0x69, 0x74, 0x22, 0xa9, 0x02, 0x0a, 0x09, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x39, 0x0a, 0x09, 0x62, 0x69, 0x6e, 0x64, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x70, 0x62, 0x2e, 0x46,
	0x69, 0x6c, 0x65, 0x42, 0x69, 0x6e, 0x64, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x48, 0x00, 0x52, 0x08, 0x62, 0x69, 0x6e, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x3f, 0x0a, 0x0b,
	0x75, 0x6e, 0x62, 0x69, 0x6e, 0x64, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1c, 0x2e, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x70, 0x62, 0x2e, 0x46, 0x69, 0x6c, 0x65,
	0x55, 0x6e, 0x62, 0x69, 0x6e, 0x64, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x48,
	0x00, 0x52, 0x0a, 0x75, 0x6e, 0x62, 0x69, 0x6e, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x4f, 0x0a,
	0x11, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x5f, 0x73,
	0x65, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x69, 0x6e, 0x64, 0x65, 0x78,
	0x70, 0x62, 0x2e, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x4c, 0x69, 0x6d, 0x69, 0x74, 0x53,
	0x65, 0x74, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x48, 0x00, 0x52, 0x0f, 0x61,
	0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x4c, 0x69, 0x6d, 0x69, 0x74, 0x53, 0x65, 0x74, 0x12, 0x49,
	0x0a, 0x0f, 0x73, 0x70, 0x61, 0x63, 0x65, 0x5f, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x5f, 0x73, 0x65,
	0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x70,
	0x62, 0x2e, 0x53, 0x70, 0x61, 0x63, 0x65, 0x4c, 0x69, 0x6d, 0x69, 0x74, 0x53, 0x65, 0x74, 0x4f,
	0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x48, 0x00, 0x52, 0x0d, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x4c, 0x69, 0x6d, 0x69, 0x74, 0x53, 0x65, 0x74, 0x42, 0x04, 0x0a, 0x02, 0x6f, 0x70, 0x22,
	0x6d, 0x0a, 0x09, 0x57, 0x41, 0x4c, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x12, 0x1e, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x69, 0x6e, 0x64, 0x65,
	0x78, 0x70, 0x62, 0x2e, 0x4b, 0x65, 0x79, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x22, 0x0a, 0x02,
	0x6f, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x69, 0x6e, 0x64, 0x65, 0x78,
	0x70, 0x62, 0x2e, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x02, 0x6f, 0x70,
	0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x42, 0x0f,
	0x5a, 0x0d, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x2f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x70, 0x62, 0x62,
	0x08, 0x65, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x70, 0xe8, 0x07,
})

var file_index_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_index_proto_goTypes = []any{
	(*Key)(nil),                      // 0: indexpb.Key
	(*FileBindOperation)(nil),        // 1: indexpb.FileBindOperation
	(*FileUnbindOperation)(nil),      // 2: indexpb.FileUnbindOperation
	(*AccountLimitSetOperation)(nil), // 3: indexpb.AccountLimitSetOperation
	(*SpaceLimitSetOperation)(nil),   // 4: indexpb.SpaceLimitSetOperation
	(*Operation)(nil),                // 5: indexpb.Operation
	(*WALRecord)(nil),                // 6: indexpb.WALRecord
}
var file_index_proto_depIdxs = []int32{
	1, // 0: indexpb.Operation.bind_file:type_name -> indexpb.FileBindOperation
	2, // 1: indexpb.Operation.unbind_file:type_name -> indexpb.FileUnbindOperation
	3, // 2: indexpb.Operation.account_limit_set:type_name -> indexpb.AccountLimitSetOperation
	4, // 3: indexpb.Operation.space_limit_set:type_name -> indexpb.SpaceLimitSetOperation
	0, // 4: indexpb.WALRecord.key:type_name -> indexpb.Key
	5, // 5: indexpb.WALRecord.op:type_name -> indexpb.Operation
	6, // [6:6] is the sub-list for method output_type
	6, // [6:6] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_index_proto_init() }
func file_index_proto_init() {
	if File_index_proto != nil {
		return
	}
	file_index_proto_msgTypes[5].OneofWrappers = []any{
		(*operation_BindFile)(nil),
		(*operation_UnbindFile)(nil),
		(*operation_AccountLimitSet)(nil),
		(*operation_SpaceLimitSet)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_index_proto_rawDesc), len(file_index_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_index_proto_goTypes,
		DependencyIndexes: file_index_proto_depIdxs,
		MessageInfos:      file_index_proto_msgTypes,
	}.Build()
	File_index_proto = out.File
	file_index_proto_goTypes = nil
	file_index_proto_depIdxs = nil
}
