package web3

import (
	"fmt"

	"github.com/bacalhau-project/lilypad/pkg/data"
	"github.com/bacalhau-project/lilypad/pkg/system"
	"github.com/bacalhau-project/lilypad/pkg/web3/bindings/users"
	"github.com/ethereum/go-ethereum/common"
)

func (sdk *Web3SDK) GetServiceAddresses(serviceType string) ([]common.Address, error) {
	solverType, err := data.GetServiceType(serviceType)
	if err != nil {
		return nil, err
	}
	return sdk.Contracts.Users.ShowUsersInList(
		sdk.CallOpts,
		solverType,
	)
}

func (sdk *Web3SDK) GetSolverAddresses() ([]common.Address, error) {
	return sdk.GetServiceAddresses("Solver")
}

func (sdk *Web3SDK) GetUser(
	address common.Address,
) (users.SharedStructsUser, error) {
	return sdk.Contracts.Users.GetUser(
		sdk.CallOpts,
		address,
	)
}

func (sdk *Web3SDK) UpdateUser(
	metadataCID string,
	url string,
	roles []uint8,
) error {
	tx, err := sdk.Contracts.Users.UpdateUser(
		sdk.TransactOpts,
		metadataCID,
		url,
		roles,
	)
	if err != nil {
		system.Error(sdk.Service, "error submitting Users.UpdateUser", err)
		return err
	} else {
		system.Info(sdk.Service, "submitted users.UpdateUser", tx)
	}
	_, err = sdk.waitTx(tx)
	if err != nil {
		return err
	}
	return nil
}

func (sdk *Web3SDK) AddUserToList(
	serviceType uint8,
) error {
	tx, err := sdk.Contracts.Users.AddUserToList(
		sdk.TransactOpts,
		serviceType,
	)
	if err != nil {
		system.Error(sdk.Service, "error submitting Users.AddUserToList", err)
		return err
	} else {
		system.Info(sdk.Service, "submitted users.AddUserToList", tx)
	}
	_, err = sdk.waitTx(tx)
	if err != nil {
		return err
	}
	return nil
}

func (sdk *Web3SDK) GetSolverUrl(address string) (string, error) {
	solver, err := sdk.Contracts.Users.GetUser(
		sdk.CallOpts,
		common.HexToAddress(address),
	)
	if err != nil {
		return "", err
	}

	if solver.UserAddress == common.HexToAddress("0x0") {
		return "", fmt.Errorf("no solver found for address: %s", address)
	}
	return solver.Url, nil
}

func (sdk *Web3SDK) Agree(
	deal data.Deal,
) (string, error) {
	mediators := []common.Address{}
	for _, mediator := range deal.Members.Mediators {
		mediators = append(mediators, common.HexToAddress(mediator))
	}
	tx, err := sdk.Contracts.Controller.Agree(
		sdk.TransactOpts,
		deal.ID,
		data.ConvertDealMembers(deal.Members),
		data.ConvertDealTimeouts(deal.Timeouts),
		data.ConvertDealPricing(deal.Pricing),
	)
	if err != nil {
		system.Error(sdk.Service, "error submitting agree tx", err)
		return "", err
	} else {
		system.Info(sdk.Service, "submitted agree tx", tx)
	}
	_, err = sdk.waitTx(tx)
	if err != nil {
		return "", err
	}
	return tx.Hash().String(), nil
}
