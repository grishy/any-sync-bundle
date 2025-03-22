package lightcoordinatorrpc

import (
	"context"
	"fmt"

	"github.com/anyproto/any-sync-coordinator/coordinator"
	"github.com/anyproto/any-sync-coordinator/spacestatus"
	commonaccount "github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonspace"
	"github.com/anyproto/any-sync/commonspace/object/accountdata"
	"github.com/anyproto/any-sync/commonspace/spacesyncproto"
	"github.com/anyproto/any-sync/coordinator/coordinatorproto"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/util/crypto"
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
	account *accountdata.AccountKeys

	srvDRPC server.DRPCServer
	srvCfg  configService
}

type configService interface {
	app.Component
	GetNodeConf() nodeconf.Configuration
	GetAccount() commonaccount.Config
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
	r.srvCfg = app.MustComponent[configService](a)

	return nil
}

func (r *lightcoordinatorrpc) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (r *lightcoordinatorrpc) Run(ctx context.Context) error {
	log.Info("call Run")

	acc := r.srvCfg.GetAccount()
	decodedSigningKey, err := crypto.DecodeKeyFromString(acc.SigningKey, crypto.UnmarshalEd25519PrivateKey, nil)
	if err != nil {
		return fmt.Errorf("decode signing key: %w", err)
	}

	decodedPeerKey, err := crypto.DecodeKeyFromString(acc.PeerKey, crypto.UnmarshalEd25519PrivateKey, nil)
	if err != nil {
		return fmt.Errorf("decode peer key: %w", err)
	}

	r.account = accountdata.New(decodedPeerKey, decodedSigningKey)

	return coordinatorproto.DRPCRegisterCoordinator(r.srvDRPC, r)
}

func (r *lightcoordinatorrpc) Close(ctx context.Context) error {
	return nil
}

//
// Component methods
//

// SpaceSign signs a space creation request and returns a SpaceReceiptWithSignature.
//
// This is the entry point for clients to create new spaces. The coordinator
// validates the request, generates a cryptographically signed receipt (proof
// of creation), and initializes the space's status. The client then uses
// this receipt to prove to sync nodes that the space is legitimate.
//
// Critical for:
//   - Space creation initialization.
//   - Client authentication to sync nodes (receipt verification).
//   - Preventing unauthorized space creation (coordinator signature).
//   - Maintaining a consistent view of space existence across the network.
func (r *lightcoordinatorrpc) SpaceSign(ctx context.Context, req *coordinatorproto.SpaceSignRequest) (*coordinatorproto.SpaceSignResponse, error) {
	panic("implement me")

	// TODO: Think about how to make it more evident that account.SignKey is actually a network key on a coordinator level
	networkKey := r.account.SignKey
	accountPubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("get public key from context: %w", err)
	}

	peerId, err := peer.CtxPeerId(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peer id from context: %w", err)
	}

	oldPubKey, err := crypto.UnmarshalEd25519PublicKeyProto(req.GetOldIdentity())
	if err != nil {
		return nil, fmt.Errorf("unmarshal old identity: %w", err)
	}

	if err := r.verifyOldAccount(accountPubKey, oldPubKey, req.NewIdentitySignature); err != nil {
		return nil, fmt.Errorf("verify old account: %w", err)
	}

	spaceHeader := &spacesyncproto.RawSpaceHeaderWithId{
		RawHeader: req.GetHeader(),
		Id:        req.GetSpaceId(),
	}

	if err := commonspace.ValidateSpaceHeader(spaceHeader, accountPubKey); err != nil {
		return nil, fmt.Errorf("validate space header: %w", err)
	}

	spaceType, err := spacestatus.VerifySpaceHeader(accountPubKey, req.GetHeader())
	if err != nil {
		return nil, fmt.Errorf("verify space header: %w", err)
	}

	// TODO: Implement here
	// if err := c.spaceStatus.NewStatus(ctx, req.GetSpaceId(), accountPubKey, oldPubKey, spaceType, req.GetForceRequest()); err != nil {
	// 	return nil, err
	// }

	_ = networkKey
	_ = peerId
	_ = spaceType

	return &coordinatorproto.SpaceSignResponse{
		// Receipt: receipt,
	}, nil
}

func (r *lightcoordinatorrpc) SpaceStatusCheck(ctx context.Context, request *coordinatorproto.SpaceStatusCheckRequest) (*coordinatorproto.SpaceStatusCheckResponse, error) {
	panic("implement me")
	return &coordinatorproto.SpaceStatusCheckResponse{}, nil
}

func (r *lightcoordinatorrpc) SpaceStatusCheckMany(ctx context.Context, request *coordinatorproto.SpaceStatusCheckManyRequest) (*coordinatorproto.SpaceStatusCheckManyResponse, error) {
	panic("implement me")
	return &coordinatorproto.SpaceStatusCheckManyResponse{}, nil
}

func (r *lightcoordinatorrpc) SpaceStatusChange(ctx context.Context, request *coordinatorproto.SpaceStatusChangeRequest) (*coordinatorproto.SpaceStatusChangeResponse, error) {
	panic("implement me")
	return &coordinatorproto.SpaceStatusChangeResponse{}, nil
}

func (r *lightcoordinatorrpc) SpaceMakeShareable(ctx context.Context, request *coordinatorproto.SpaceMakeShareableRequest) (*coordinatorproto.SpaceMakeShareableResponse, error) {
	panic("implement me")
	return &coordinatorproto.SpaceMakeShareableResponse{}, nil
}

func (r *lightcoordinatorrpc) SpaceMakeUnshareable(ctx context.Context, request *coordinatorproto.SpaceMakeUnshareableRequest) (*coordinatorproto.SpaceMakeUnshareableResponse, error) {
	panic("implement me")
	return &coordinatorproto.SpaceMakeUnshareableResponse{}, nil
}

func (r *lightcoordinatorrpc) NetworkConfiguration(ctx context.Context, request *coordinatorproto.NetworkConfigurationRequest) (*coordinatorproto.NetworkConfigurationResponse, error) {
	network := r.srvCfg.GetNodeConf()
	nodes := make([]*coordinatorproto.Node, 0, len(network.Nodes))
	for _, n := range network.Nodes {
		types := make([]coordinatorproto.NodeType, 0, len(n.Types))
		for _, t := range n.Types {
			switch t {
			case nodeconf.NodeTypeCoordinator:
				types = append(types, coordinatorproto.NodeType_CoordinatorAPI)
			case nodeconf.NodeTypeFile:
				types = append(types, coordinatorproto.NodeType_FileAPI)
			case nodeconf.NodeTypeTree:
				types = append(types, coordinatorproto.NodeType_TreeAPI)
			case nodeconf.NodeTypeConsensus:
				types = append(types, coordinatorproto.NodeType_ConsensusAPI)
			default:
				panic("unknown node type " + t)
			}
		}

		nodes = append(nodes, &coordinatorproto.Node{
			PeerId:    n.PeerId,
			Addresses: n.Addresses,
			Types:     types,
		})
	}

	return &coordinatorproto.NetworkConfigurationResponse{
		ConfigurationId: network.Id,
		NetworkId:       network.NetworkId,
		Nodes:           nodes,
		// TODO: Check on real system what we have here
		CreationTimeUnix: uint64(network.CreationTime.Unix()),
	}, nil
}

func (r *lightcoordinatorrpc) DeletionLog(ctx context.Context, request *coordinatorproto.DeletionLogRequest) (*coordinatorproto.DeletionLogResponse, error) {
	panic("implement me")
	return &coordinatorproto.DeletionLogResponse{}, nil
}

func (r *lightcoordinatorrpc) SpaceDelete(ctx context.Context, request *coordinatorproto.SpaceDeleteRequest) (*coordinatorproto.SpaceDeleteResponse, error) {
	panic("implement me")
	return &coordinatorproto.SpaceDeleteResponse{}, nil
}

func (r *lightcoordinatorrpc) AccountDelete(ctx context.Context, request *coordinatorproto.AccountDeleteRequest) (*coordinatorproto.AccountDeleteResponse, error) {
	panic("implement me")
	return &coordinatorproto.AccountDeleteResponse{}, nil
}

func (r *lightcoordinatorrpc) AccountRevertDeletion(ctx context.Context, request *coordinatorproto.AccountRevertDeletionRequest) (*coordinatorproto.AccountRevertDeletionResponse, error) {
	panic("implement me")
	return &coordinatorproto.AccountRevertDeletionResponse{}, nil
}

func (r *lightcoordinatorrpc) AclAddRecord(ctx context.Context, request *coordinatorproto.AclAddRecordRequest) (*coordinatorproto.AclAddRecordResponse, error) {
	panic("implement me")
	return &coordinatorproto.AclAddRecordResponse{}, nil
}

func (r *lightcoordinatorrpc) AclGetRecords(ctx context.Context, request *coordinatorproto.AclGetRecordsRequest) (*coordinatorproto.AclGetRecordsResponse, error) {
	panic("implement me")
	return &coordinatorproto.AclGetRecordsResponse{}, nil
}

func (r *lightcoordinatorrpc) AccountLimitsSet(ctx context.Context, request *coordinatorproto.AccountLimitsSetRequest) (*coordinatorproto.AccountLimitsSetResponse, error) {
	panic("implement me")
	return &coordinatorproto.AccountLimitsSetResponse{}, nil
}

func (r *lightcoordinatorrpc) AclEventLog(ctx context.Context, request *coordinatorproto.AclEventLogRequest) (*coordinatorproto.AclEventLogResponse, error) {
	panic("implement me")
	return &coordinatorproto.AclEventLogResponse{}, nil
}

func (r *lightcoordinatorrpc) verifyOldAccount(newAccountKey, oldAccountKey crypto.PubKey, signature []byte) error {
	rawPub, err := newAccountKey.Raw()
	if err != nil {
		return fmt.Errorf("get raw public key: %w", err)
	}
	verify, err := oldAccountKey.Verify(rawPub, signature)
	if err != nil {
		return fmt.Errorf("verify old account key: %w", err)
	}

	if !verify {
		return coordinator.ErrIncorrectAccountSignature
	}

	return nil
}
