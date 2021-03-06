package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
)

const (
	QueryValidators = "validators"
	QueryValidator  = "validator"
	QueryMinMax     = "minmax"
)

// Client request for validator by address.
type ValidatorReq struct {
	Address types.AccAddress `json:"address" yaml:"address"`
}

// Client response for getValidators.
type ValidatorsConfirmationsResp struct {
	// Registered validators list
	Validators Validators `json:"validators" yaml:"validators"`
	// Minimum number of confirmations needed to approve Call
	Confirmations uint16 `json:"confirmations" yaml:"confirmations" example:"3"`
}

func (r ValidatorsConfirmationsResp) String() string {
	return fmt.Sprintf("%s\nConfirmations: %d", r.Validators.String(), r.Confirmations)
}
