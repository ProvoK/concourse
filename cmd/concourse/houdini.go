package main

import (
	"fmt"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/ifrit"
	"github.com/vito/houdini"
)

func (cmd *WorkerCommand) houdiniRunner(logger lager.Logger) (ifrit.Runner, error) {
	depotDir := filepath.Join(cmd.WorkDir.Path(), "containers")

	err := os.MkdirAll(depotDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create depot dir: %s", err)
	}

	return cmd.backendRunner(logger, houdini.NewBackend(depotDir)), nil
}
