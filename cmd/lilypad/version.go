package lilypad

import (
	"fmt"

	"github.com/spf13/cobra"

	optionsfactory "github.com/bacalhau-project/lilypad/pkg/options"
	"github.com/bacalhau-project/lilypad/pkg/system"
)

// VERSION: use `go build -ldflags="-X lilypad.VERSION=x.y.z" `
const VERSION = ""

func newVersionCmd() *cobra.Command {
	options := optionsfactory.NewSolverOptions()

	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"-v"},
		Short:   "Get the lilypad version",
		Long:    "Get the lilypad version",
		Example: "lilypad version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runVersion(cmd)
		},
	}

	optionsfactory.AddSolverCliFlags(versionCmd, &options)

	return versionCmd
}

func runVersion(cmd *cobra.Command) error {
	commandCtx := system.NewCommandContext(cmd)
	defer commandCtx.Cleanup()

	fmt.Printf("Lilypad: %s\n", VERSION)

	return nil
}
