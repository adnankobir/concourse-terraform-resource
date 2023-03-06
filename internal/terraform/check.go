package terraform

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/adnankobir/concourse-terraform-resource/internal/types"
)

// CheckRequest describes the input to a check operation
type CheckRequest struct {
	Source  types.Source   `json:"source" yaml:"source"`
	Version *types.Version `json:"version,omitempty" yaml:"version,omitempty"`
}

// CheckResponse describes the output from a successful check operation
type CheckResponse []types.Version

// Check describes a check command executor
type Check struct {
	stdin  io.Reader
	stderr io.Writer
	stdout io.Writer
	args   []string
}

// NewCheck instantiates a new check command executor
func NewCheck(stdin io.Reader, stderr io.Writer, stdout io.Writer, args []string) *Check {
	return &Check{
		stdin:  stdin,
		stderr: stderr,
		stdout: stdout,
		args:   args,
	}
}

// Execute handles a check operation
func (cmd *Check) Execute() error {
	setupLogging(cmd.stderr)

	var req CheckRequest
	decoder := json.NewDecoder(cmd.stdin)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	versions := []types.Version{}
	if req.Version != nil {
		versions = append(versions, *req.Version)
	}

	if err := json.NewEncoder(cmd.stdout).Encode(versions); err != nil {
		return fmt.Errorf("error marshalling response: %v", err)
	}
	return nil
}
