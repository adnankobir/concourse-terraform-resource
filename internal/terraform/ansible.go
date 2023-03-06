package terraform

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/adnankobir/concourse-terraform-resource/internal/types"
	"github.com/Jeffail/gabs/v2"
)

// Ansible manages state for an ansible-playbook invocation
type Ansible struct {
	args      []string
	envs      []string
	extraVars *gabs.Container
	stdout    io.Writer
	playbook  string
}

// NewAnsible initializes a new ansible playbook command
func NewAnsible(src *types.Source, out io.Writer, env *types.Environment, playbook string, workdir string) *Ansible {
	ansible := Ansible{
		stdout:   out,
		playbook: playbook,
	}

	// define extra ansible playbook environment variables
	ansible.envs = append(os.Environ(), toList(map[string]string{
		"ANSIBLE_FORCE_COLOR": "True",
		//"PY_COLORS":           "1",
		//"ANSIBLE_CALLBACKS_ENABLED":       "selective",
		//"ANSIBLE_STDOUT_CALLBACK":         "selective",
		//"ANSIBLE_LOAD_CALLBACK_PLUGINS":   "1",
		"ANSIBLE_STDOUT_CALLBACK":         "debug",
		"ANSIBLE_DISPLAY_SKIPPED_HOSTS":   "False",
		"ANSIBLE_HASHI_VAULT_ADDR":        src.Vault.Addr,
		"ANSIBLE_HASHI_VAULT_AUTH_METHOD": "approle",
		"ANSIBLE_HASHI_VAULT_ROLE_ID":     src.Vault.RoleID,
		"ANSIBLE_HASHI_VAULT_SECRET_ID":   src.Vault.SecretID,
		"ANSIBLE_COLOR_OK":                "white",
	})...)

	// enable ansible debug logs if debug flag is set
	//if src.Debug {
	//	ansible.args = append(ansible.args, "-v")
	//}

	extraVars := gabs.New()

	extraVars.Set(env.ATCExternalURL, "concourse_atc_external_url")
	extraVars.Set(env.ID, "concourse_build_id")
	extraVars.Set(env.Job, "concourse_build_job")
	extraVars.Set(env.Name, "concourse_build_name")
	extraVars.Set(env.Pipeline, "concourse_build_pipeline")
	extraVars.Set(env.Team, "concourse_build_team")
	extraVars.Set(src.Component, "component")
	extraVars.Set(src.Storage, "storage")
	extraVars.Set(workdir, "workdir")
	ansible.extraVars = extraVars

	return &ansible
}

// Run wraps the underlying command run function invocation and handles cleanup
func (a *Ansible) Run() error {
	extraVars, err := a.prepareRun()
	defer os.Remove(extraVars.Name())
	if err != nil {
		return fmt.Errorf("error writing extra vars: %v", err)
	}

	cmd := exec.Command("ansible-playbook", append(a.args, a.playbook)...)
	cmd.Stdout = a.stdout
	cmd.Env = append(os.Environ(), a.envs...)

	return cmd.Run()
}

// prepare ansible-playbook run
func (a *Ansible) prepareRun() (*os.File, error) {
	// create temporary file for playbook variables
	tmpFile, err := ioutil.TempFile("", "vars")
	if err != nil {
		return nil, fmt.Errorf("error creating file: %v", err)
	}

	// write playbook variables to temp file
	if err := ioutil.WriteFile(tmpFile.Name(), a.extraVars.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("error writing file: %v", err)
	}
	a.args = append(a.args, "-e", fmt.Sprintf("@%s", tmpFile.Name()))

	return tmpFile, nil
}

func toList(in map[string]string) []string {
	var out []string
	for k, v := range in {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}
