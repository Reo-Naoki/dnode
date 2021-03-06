package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkErrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/dfinance/dvm-proto/go/vm_grpc"
	"github.com/dfinance/glav"
	"github.com/dfinance/lcs"

	"github.com/dfinance/dnode/x/ccstorage/internal/types"
	"github.com/dfinance/dnode/x/common_vm"
)

// GetResStdCurrencyInfo returns VM currencyInfo for stdlib currencies (non-token).
func (k Keeper) GetResStdCurrencyInfo(ctx sdk.Context, denom string) (types.ResCurrencyInfo, error) {
	k.modulePerms.AutoCheck(types.PermRead)

	accessPath := &vm_grpc.VMAccessPath{
		Address: common_vm.StdLibAddress,
		Path:    glav.CurrencyInfoVector(denom),
	}

	if !k.vmKeeper.HasValue(ctx, accessPath) {
		return types.ResCurrencyInfo{}, sdkErrors.Wrapf(types.ErrInternal, "currencyInfo for %q: nof found in VM storage", denom)
	}

	currencyInfo := types.ResCurrencyInfo{}
	bz := k.vmKeeper.GetValue(ctx, accessPath)
	if err := lcs.Unmarshal(bz, &currencyInfo); err != nil {
		return types.ResCurrencyInfo{}, sdkErrors.Wrapf(types.ErrInternal, "currencyInfo for %q: lcs unmarshal: %v", denom, err)
	}

	return currencyInfo, nil
}

// storeResStdCurrencyInfo sets currencyInfo to the VM storage.
func (k Keeper) storeResStdCurrencyInfo(ctx sdk.Context, currency types.Currency) {
	currencyInfo, err := types.NewResCurrencyInfo(currency, common_vm.StdLibAddress)
	if err != nil {
		panic(fmt.Errorf("currency %q: %v", currency.Denom, err))
	}

	bz, err := lcs.Marshal(currencyInfo)
	if err != nil {
		panic(fmt.Errorf("currency %q: lcs marshal: %v", currency.Denom, err))
	}

	accessPath := &vm_grpc.VMAccessPath{
		Address: common_vm.StdLibAddress,
		Path:    currency.InfoPath(),
	}

	k.vmKeeper.SetValue(ctx, accessPath, bz)
}
