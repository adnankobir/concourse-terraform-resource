package terraform

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/adnankobir/concourse-terraform-resource/internal/types"
)

// In describes an in command executor
type In struct {
	stdin  io.Reader
	stderr io.Writer
	stdout io.Writer
	args   []string
	env    types.Environment
}

// NewIn instantiates a new in command executor
func NewIn(stdin io.Reader, stderr io.Writer, stdout io.Writer, args []string) *In {
	return &In{
		stdin:  stdin,
		stderr: stderr,
		stdout: stdout,
		args:   args,
	}
}

/// Execute handles a get operation
func (cmd *In) Execute() error {
	setupLogging(cmd.stderr)
	if err := parseEnvironment(&cmd.env); err != nil {
		return err
	}

	var req types.InRequest
	decoder := json.NewDecoder(cmd.stdin)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	resp := types.InResponse{
		Version:  req.Version,
		Metadata: []types.Metadata{},
	}

	if err := json.NewEncoder(cmd.stdout).Encode(&resp); err != nil {
		return fmt.Errorf("error marshalling response: %v", err)
	}

	return nil
}

func (cmd *In) ansiblePlaybookCmd(req *types.InRequest) (*Ansible, error) {
	// validate put request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid get request: %v", err)
	}

	ansible := NewAnsible(&req.Source, cmd.stderr, &cmd.env, "/opt/ansible/in.yml", cmd.args[1])
	ansible.extraVars.Set(req.Version, "version")
	return ansible, nil
}
