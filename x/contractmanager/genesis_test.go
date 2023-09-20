package contractmanager_test

import (
	"github.com/neutron-org/neutron/x/contractmanager/keeper"
	"testing"

	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"

	"github.com/stretchr/testify/require"

	keepertest "github.com/neutron-org/neutron/testutil/contractmanager/keeper"
	"github.com/neutron-org/neutron/testutil/contractmanager/nullify"
	"github.com/neutron-org/neutron/x/contractmanager"
	"github.com/neutron-org/neutron/x/contractmanager/types"
)

func TestGenesis(t *testing.T) {
	payload1, err := keeper.PrepareSudoCallbackMessage(
		channeltypes.Packet{
			Sequence: 1,
		},
		&channeltypes.Acknowledgement{
			Response: &channeltypes.Acknowledgement_Result{
				Result: []byte("Result"),
			},
		})
	require.NoError(t, err)
	payload2, err := keeper.PrepareSudoCallbackMessage(
		channeltypes.Packet{
			Sequence: 2,
		}, nil)
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		FailuresList: []types.Failure{
			{
				Address:     "address1",
				Id:          1,
				SudoPayload: payload1,
			},
			{
				Address:     "address1",
				Id:          2,
				SudoPayload: payload2,
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ContractManagerKeeper(t, nil)
	contractmanager.InitGenesis(ctx, *k, genesisState)
	got := contractmanager.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.FailuresList, got.FailuresList)
	// this line is used by starport scaffolding # genesis/test/assert
}
