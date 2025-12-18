package chain

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/ProjectsTask/EasySwapBase/evm/eip"
)

const (
	Eth      = "eth"
	Optimism = "optimism"
	Sepolia  = "sepolia"
)

const (
	EthChainID      = 1
	OptimismChainID = 10
	SepoliaChainID  = 11155111
)

func UniformAddress(chainName string, address string) (string, error) {
	addr, err := eip.ToCheckSumAddress(address)
	if err != nil {
		return "", errors.Wrap(err, "failed on uniform evm chain address")
	}
	return strings.ToLower(addr), nil

}
