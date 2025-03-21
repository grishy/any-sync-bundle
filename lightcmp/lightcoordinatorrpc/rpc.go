package lightcoordinatorrpc

import (
	"context"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/coordinator/coordinatorproto"
	"github.com/anyproto/any-sync/net/rpc/server"
)

const (
	CName = "light.coordinator.rpc"
)

var (
	// Type assertion
	_ coordinatorproto.DRPCCoordinatorServer = (*lightcoordinatorrpc)(nil)

	log = logger.NewNamed(CName)
)

type lightcoordinatorrpc struct {
	srvDRPC server.DRPCServer
}

func New() *lightcoordinatorrpc {
	return new(lightcoordinatorrpc)
}

//
// App Component
//

func (r *lightcoordinatorrpc) Init(a *app.App) error {
	log.Info("call Init")

	r.srvDRPC = app.MustComponent[server.DRPCServer](a)

	return nil
}

func (r *lightcoordinatorrpc) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (r *lightcoordinatorrpc) Run(ctx context.Context) error {
	return coordinatorproto.DRPCRegisterCoordinator(r.srvDRPC, r)
}

func (r *lightcoordinatorrpc) Close(ctx context.Context) error {
	return nil
}

//
// Component methods
//

func (r *lightcoordinatorrpc) SpaceSign(ctx context.Context, request *coordinatorproto.SpaceSignRequest) (*coordinatorproto.SpaceSignResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) SpaceStatusCheck(ctx context.Context, request *coordinatorproto.SpaceStatusCheckRequest) (*coordinatorproto.SpaceStatusCheckResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) SpaceStatusCheckMany(ctx context.Context, request *coordinatorproto.SpaceStatusCheckManyRequest) (*coordinatorproto.SpaceStatusCheckManyResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) SpaceStatusChange(ctx context.Context, request *coordinatorproto.SpaceStatusChangeRequest) (*coordinatorproto.SpaceStatusChangeResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) SpaceMakeShareable(ctx context.Context, request *coordinatorproto.SpaceMakeShareableRequest) (*coordinatorproto.SpaceMakeShareableResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) SpaceMakeUnshareable(ctx context.Context, request *coordinatorproto.SpaceMakeUnshareableRequest) (*coordinatorproto.SpaceMakeUnshareableResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) NetworkConfiguration(ctx context.Context, request *coordinatorproto.NetworkConfigurationRequest) (*coordinatorproto.NetworkConfigurationResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) DeletionLog(ctx context.Context, request *coordinatorproto.DeletionLogRequest) (*coordinatorproto.DeletionLogResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) SpaceDelete(ctx context.Context, request *coordinatorproto.SpaceDeleteRequest) (*coordinatorproto.SpaceDeleteResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) AccountDelete(ctx context.Context, request *coordinatorproto.AccountDeleteRequest) (*coordinatorproto.AccountDeleteResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) AccountRevertDeletion(ctx context.Context, request *coordinatorproto.AccountRevertDeletionRequest) (*coordinatorproto.AccountRevertDeletionResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) AclAddRecord(ctx context.Context, request *coordinatorproto.AclAddRecordRequest) (*coordinatorproto.AclAddRecordResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) AclGetRecords(ctx context.Context, request *coordinatorproto.AclGetRecordsRequest) (*coordinatorproto.AclGetRecordsResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) AccountLimitsSet(ctx context.Context, request *coordinatorproto.AccountLimitsSetRequest) (*coordinatorproto.AccountLimitsSetResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorrpc) AclEventLog(ctx context.Context, request *coordinatorproto.AclEventLogRequest) (*coordinatorproto.AclEventLogResponse, error) {
	// TODO implement me
	panic("implement me")
}
