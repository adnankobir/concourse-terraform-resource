package terraform

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/adnankobir/concourse-terraform-resource/internal/types"
	"github.com/Jeffail/benthos/v3/lib/bloblang"
	"github.com/Jeffail/benthos/v3/lib/message"
	"github.com/Jeffail/gabs/v2"
)

// Out describes an out command executor
type Out struct {
	stdin  io.Reader
	stderr io.Writer
	stdout io.Writer
	args   []string
	env    types.Environment
	input  bloblang.Message
}

// NewOut instantiates a new out command executor
func NewOut(stdin io.Reader, stderr io.Writer, stdout io.Writer, args []string) *Out {
	return &Out{
		stdin:  stdin,
		stderr: stderr,
		stdout: stdout,
		args:   args,
	}
}

// Execute handles a put operation
func (cmd *Out) Execute() (err error) {
	setupLogging(cmd.stderr)
	if err := parseEnvironment(&cmd.env); err != nil {
		return err
	}

	// parse put operation request
	var req types.OutRequest
	decoder := json.NewDecoder(cmd.stdin)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// configure ssh
	if keyField, ok := req.PrivateKey(); ok {
		key, err := cmd.parseField(keyField)
		if err != nil {
			return fmt.Errorf("error parsing private key: %v", err)
		}
		agent, err := setupSSH(key)
		defer agent.Shutdown()
		if err != nil {
			return fmt.Errorf("error configuring ssh agent: %v", err)
		}
	}

	// change into target working directory
	if err := os.Chdir(cmd.args[1]); err != nil {
		return fmt.Errorf("error changing into out working directory: %v", err)
	}

	// execute ansible-playbook
	ansible, err := cmd.ansiblePlaybookCmd(&req)
	if err != nil {
		return fmt.Errorf("Failed to build ansible playbook command: %v", err)
	}
	if err := ansible.Run(); err != nil {
		return fmt.Errorf("error executing ansible-playbook: %v", err)
	}

	versionID, err := ioutil.ReadFile("version_id")
	if err != nil {
		versionID = []byte(time.Now().Format(time.RFC3339Nano))
		// return fmt.Errorf("error reading version_id from file system after playbook: %v", err)
	}

	version := types.Version{
		Key:       req.Source.Storage.Key,
		VersionID: string(versionID),
	}
	resp := types.OutResponse{
		Version: version,
	}

	if err := json.NewEncoder(cmd.stdout).Encode(&resp); err != nil {
		return fmt.Errorf("error marshalling response: %v", err)
	}

	return nil
}

// prepare ansible-playbook command
func (cmd *Out) ansiblePlaybookCmd(req *types.OutRequest) (*Ansible, error) {
	// validate put request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid put request: %v", err)
	}

	req.Source.AssignFallbackValues(cmd.env.Team, cmd.env.Pipeline)

	// compute operation input
	var err error
	if req.Params.InputMapping != "" {
		cmd.input, err = cmd.parseMapping(req.Params.InputMapping)
		if err != nil {
			return nil, fmt.Errorf("error parsing input context: %v", err)
		}
	}

	ansible := NewAnsible(&req.Source, cmd.stderr, &cmd.env, "/opt/ansible/out.yml", cmd.args[1])

	// merge user provided environment variables
	for k, v := range req.Envs() {
		parsed, err := cmd.parseField(v)
		if err != nil {
			return nil, fmt.Errorf("error parsing env (%s): %v", k, err)
		}
		ansible.envs = append(ansible.envs, fmt.Sprintf("%s=%s", k, parsed))
	}

	// write ansible playbook extra vars file and set arg
	if err := cmd.injectExtraVars(ansible.extraVars, req); err != nil {
		return nil, fmt.Errorf("error writing ansible extra vars: %v", err)
	}

	return ansible, nil
}

// inject playbook-specific extra variables
func (cmd *Out) injectExtraVars(extraVars *gabs.Container, req *types.OutRequest) error {
	// set context
	context, err := cmd.parseField(req.Params.Context)
	if err != nil {
		return err
	}
	extraVars.Set(context, "context")

	// set terraform_workspace
	workspace, err := cmd.parseField(req.Params.Workspace)
	if err != nil {
		return fmt.Errorf("error parsing request param: %v", err)
	}
	if workspace == "" {
		workspace = context
	}
	extraVars.Set(workspace, "terraform_workspace")

	// parse release version
	if req.Params.ReleaseVersion != "" {
		releaseVersion, err := cmd.parseField(req.Params.Context)
		if err != nil {
			return fmt.Errorf("error parsing release version: %v", err)
		}
		extraVars.Set(releaseVersion, "release_version")
	}

	// parse vars mapping
	if req.Params.VarsMapping != "" {
		tfvars, err := cmd.parseMapping(req.Params.VarsMapping)
		if err != nil {
			return fmt.Errorf("error executing vars mapping: %v", err)
		}
		if b := tfvars.Get(0).Get(); len(b) > 0 {
			tfvarsJSON, err := gabs.ParseJSON(b)
			if err != nil {
				return fmt.Errorf("error parsing mapped vars: %v", err)
			}
			extraVars.Set(tfvarsJSON, "terraform_vars")
		}
	}

	if n := len(req.Params.VarFiles); n > 0 {
		varFiles := make([]string, n)
		for i, f := range req.Params.VarFiles {
			varFile, err := cmd.parseField(f)
			if err != nil {
				return fmt.Errorf("error parsing var file: %v", err)
			}
			if !strings.HasPrefix(varFile, "/") {
				varFile = path.Join(cmd.args[1], varFile)
			}
			varFiles[i] = varFile
		}
		extraVars.Set(varFiles, "terraform_var_files")
	}

	extraVars.Set(path.Join(cmd.args[1], req.Params.Dir), "terraform_path")

	extraVars.Set(req.Params.PlanOnly, "plan_only")

	extraVars.Set(req.Params.Destroy, "destroy")

	return nil
}

// parse text as bloblang field (ie string with embedded bloblang expressions wrapped in '${!...}')
func (cmd *Out) parseField(text string) (string, error) {
	input := cmd.input
	if input == nil {
		input = message.New([][]byte{[]byte("{}")})
	}
	f, err := bloblang.NewField(text)
	if err != nil {
		return "", err
	}
	return f.String(0, input), nil
}

// parse text as bloblang mapping
func (cmd *Out) parseMapping(text string) (bloblang.Message, error) {
	msg := message.New(nil)
	input := cmd.input
	if input == nil {
		input = message.New([][]byte{[]byte("{}")})
	}
	m, err := bloblang.NewMapping(text)
	if err != nil {
		return nil, err
	}
	p, err := m.MapPart(0, input)
	if err != nil {
		return nil, fmt.Errorf("error executing mapping: %v", err)
	}
	msg.Append(p)
	return msg, nil
}
