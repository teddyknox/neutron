package testutil

import (
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/simapp"
	icatypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	"github.com/neutron-org/neutron/app"
	ictxstypes "github.com/neutron-org/neutron/x/interchaintxs/types"
	"github.com/stretchr/testify/suite"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "neutron17dtl0mjt3t77kpuhg2edqzjpszulwhgzcdvagh"

	TestInterchainId = "owner_id"

	// TestVersion defines a reusable interchainaccounts version string for testing purposes
	TestVersion = string(icatypes.ModuleCdc.MustMarshalJSON(&icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: ibctesting.FirstConnectionID,
		HostConnectionId:       ibctesting.FirstConnectionID,
		Encoding:               icatypes.EncodingProtobuf,
		TxType:                 icatypes.TxTypeSDKMultiMsg,
	}))
)

func init() {
	ibctesting.DefaultTestingAppInit = SetupTestingApp
	config := app.GetDefaultConfig()
	config.Seal()
}

type IBCConnectionTestSuite struct {
	suite.Suite
	Coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	ChainA *ibctesting.TestChain
	ChainB *ibctesting.TestChain

	Path *ibctesting.Path
}

func (suite *IBCConnectionTestSuite) SetupTest() {
	suite.Coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.ChainA = suite.Coordinator.GetChain(ibctesting.GetChainID(1))
	suite.ChainB = suite.Coordinator.GetChain(ibctesting.GetChainID(2))

	suite.Path = NewICAPath(suite.ChainA, suite.ChainB)

	suite.Coordinator.SetupConnections(suite.Path)

	suite.NoError(SetupICAPath(suite.Path, TestOwnerAddress))
}

func (suite *IBCConnectionTestSuite) GetNeutronZoneApp(chain *ibctesting.TestChain) *app.App {
	testApp, ok := chain.App.(*app.App)
	if !ok {
		panic("not NeutronZone app")
	}

	return testApp
}

func NewICAPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = icatypes.PortID
	path.EndpointB.ChannelConfig.PortID = icatypes.PortID
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointA.ChannelConfig.Version = TestVersion
	path.EndpointB.ChannelConfig.Version = TestVersion

	return path
}

// SetupICAPath invokes the InterchainAccounts entrypoint and subsequent channel handshake handlers
func SetupICAPath(path *ibctesting.Path, owner string) error {
	if err := RegisterInterchainAccount(path.EndpointA, owner); err != nil {
		return err
	}

	if err := path.EndpointB.ChanOpenTry(); err != nil {
		return err
	}

	if err := path.EndpointA.ChanOpenAck(); err != nil {
		return err
	}

	if err := path.EndpointB.ChanOpenConfirm(); err != nil {
		return err
	}

	return nil
}

// RegisterInterchainAccount is a helper function for starting the channel handshake
func RegisterInterchainAccount(endpoint *ibctesting.Endpoint, owner string) error {
	icaOwner, _ := ictxstypes.NewICAOwner(TestOwnerAddress, TestInterchainId)
	portID, err := icatypes.NewControllerPortID(icaOwner.String())
	if err != nil {
		return err
	}

	channelSequence := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(endpoint.Chain.GetContext())

	a, ok := endpoint.Chain.App.(*app.App)
	if !ok {
		return fmt.Errorf("not NeutronZoneApp")
	}

	if err := a.ICAControllerKeeper.RegisterInterchainAccount(endpoint.Chain.GetContext(), endpoint.ConnectionID, icaOwner.String()); err != nil {
		return err
	}

	// commit state changes for proof verification
	endpoint.Chain.App.Commit()
	endpoint.Chain.NextBlock()

	// update port/channel ids
	endpoint.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)
	endpoint.ChannelConfig.PortID = portID

	return nil
}

// SetupTestingApp initializes the IBC-go testing application
func SetupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	encoding := app.MakeEncodingConfig()
	db := dbm.NewMemDB()
	testApp := app.New(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		encoding,
		app.GetEnabledProposals(),
		simapp.EmptyAppOptions{},
		nil,
	)
	return testApp, app.NewDefaultGenesisState(testApp.AppCodec())
}