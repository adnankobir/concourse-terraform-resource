package terraform

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/adnankobir/concourse-terraform-resource/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestAnsibleOut(t *testing.T) {
	src := types.Source{
		Storage: types.Storage{
			AWSAccessKeyID:     "foo",
			AWSSecretAccessKey: "bar",
			Bucket:             "foo",
			Region:             "us-east-1",
		},
		Vault: types.VaultSource{
			Addr:     "https://vault.com",
			RoleID:   "vault-role-d",
			SecretID: "vault-secret-id",
		},
	}

	cases := []struct {
		desc   string
		out    *Out
		req    *types.OutRequest
		assert func(*Out, *types.OutRequest, *Ansible, error)
	}{
		{
			desc: "invalid request",
			req:  &types.OutRequest{},
			assert: func(out *Out, req *types.OutRequest, ansible *Ansible, err error) {
				assert.Error(t, err)
			},
		},
		{
			desc: "basic",
			req: &types.OutRequest{
				Source: src,
				Params: types.OutParams{
					Context: "foo",
					Dir:     "source/terraform",
				},
			},
			assert: func(out *Out, req *types.OutRequest, ansible *Ansible, err error) {
				assert.NoError(t, err)

				extraVars, err := ansible.prepareRun()
				assert.NoError(t, err)
				defer os.Remove(extraVars.Name())
				assert.Contains(t, ansible.envs, "ANSIBLE_FORCE_COLOR=True")
				assert.Contains(t, ansible.envs, "ANSIBLE_STDOUT_CALLBACK=debug")
				assert.Contains(t, ansible.envs, fmt.Sprintf("ANSIBLE_HASHI_VAULT_ADDR=%s", req.Source.Vault.Addr))
				assert.Contains(t, ansible.envs, "ANSIBLE_HASHI_VAULT_AUTH_METHOD=approle")
				assert.Contains(t, ansible.envs, fmt.Sprintf("ANSIBLE_HASHI_VAULT_ROLE_ID=%s", req.Source.Vault.RoleID))
				assert.Contains(t, ansible.envs, fmt.Sprintf("ANSIBLE_HASHI_VAULT_SECRET_ID=%s", req.Source.Vault.SecretID))

				assert.Contains(t, ansible.args, fmt.Sprintf("@%s", extraVars.Name()))
				assert.NotContains(t, ansible.args, "-v")

				vars, err := ioutil.ReadFile(extraVars.Name())
				assert.NoError(t, err)
				assert.Equal(t, out.env.Pipeline, gjson.GetBytes(vars, "component").String(), "invalid extra var: component")
				assert.Equal(t, out.env.ATCExternalURL, gjson.GetBytes(vars, "concourse_atc_external_url").String(), "invalid extra var: concourse_atc_external_url")
				assert.Equal(t, out.env.ID, gjson.GetBytes(vars, "concourse_build_id").String(), "invalid extra var: concourse_build_id")
				assert.Equal(t, out.env.Job, gjson.GetBytes(vars, "concourse_build_job").String(), "invalid extra var: concourse_build_job")
				assert.Equal(t, out.env.Name, gjson.GetBytes(vars, "concourse_build_name").String(), "invalid extra var: concourse_build_name")
				assert.Equal(t, out.env.Pipeline, gjson.GetBytes(vars, "concourse_build_pipeline").String(), "invalid extra var: concourse_build_pipeline")
				assert.Equal(t, out.env.Team, gjson.GetBytes(vars, "concourse_build_team").String(), "invalid extra var: concourse_build_team")
				assert.Equal(t, req.Params.Context, gjson.GetBytes(vars, "context").String(), "invalid extra var: context")
				assert.False(t, gjson.GetBytes(vars, "release_version").Exists())
				assert.Equal(t, path.Join(out.args[1], req.Params.Dir), gjson.GetBytes(vars, "terraform_path").String(), "invalid extra var: terraform_path")
				assert.False(t, gjson.GetBytes(vars, "terraform_vars").Exists())
				assert.False(t, gjson.GetBytes(vars, "terraform_var_file").Exists())
				assert.Equal(t, req.Params.Context, gjson.GetBytes(vars, "terraform_workspace").String(), "invalid extra var: terraform_workspace")
				assert.Equal(t, out.args[1], gjson.GetBytes(vars, "workdir").String(), "invalid extra var: workdir")
			},
		},
		{
			desc: "component",
			req: &types.OutRequest{
				Source: types.Source{
					Component: "the-component",
					Storage:   src.Storage,
					Vault:     src.Vault,
				},
				Params: types.OutParams{
					Context: "foo",
					Dir:     "source/terraform",
				},
			},
			assert: func(out *Out, req *types.OutRequest, ansible *Ansible, err error) {
				assert.NoError(t, err)

				extraVars, err := ansible.prepareRun()
				assert.NoError(t, err)
				defer os.Remove(extraVars.Name())

				vars, err := ioutil.ReadFile(extraVars.Name())
				assert.NoError(t, err)
				assert.Equal(t, "the-component", gjson.GetBytes(vars, "component").String(), "invalid extra var: component")
				assert.Equal(t, fmt.Sprintf("%s/%s/concourse-terraform-resource/version.tgz", out.env.Team, "the-component"),
					gjson.GetBytes(vars, "storage.key").String(), "invalid extra var: storage.key")
			},
		},
		{
			desc: "component fallback to pipeline",
			req: &types.OutRequest{
				Source: types.Source{
					Storage: types.Storage{
						AWSAccessKeyID:     "foo",
						AWSSecretAccessKey: "bar",
						Bucket:             "foo",
						Region:             "us-east-1",
					},
					Vault: src.Vault,
				},
				Params: types.OutParams{
					Context: "foo",
					Dir:     "source/terraform",
				},
			},
			assert: func(out *Out, req *types.OutRequest, ansible *Ansible, err error) {
				assert.NoError(t, err)

				extraVars, err := ansible.prepareRun()
				assert.NoError(t, err)
				defer os.Remove(extraVars.Name())

				vars, err := ioutil.ReadFile(extraVars.Name())
				assert.NoError(t, err)
				assert.Equal(t, out.env.Pipeline, gjson.GetBytes(vars, "component").String(), "invalid extra var: component")
				assert.Equal(t, fmt.Sprintf("%s/%s/concourse-terraform-resource/version.tgz", out.env.Team, out.env.Pipeline),
					gjson.GetBytes(vars, "storage.key").String(), "invalid extra var: storage.key")
			},
		},
		{
			desc: "storage key config",
			req: &types.OutRequest{
				Source: types.Source{
					Storage: types.Storage{
						AWSAccessKeyID:     "foo",
						AWSSecretAccessKey: "bar",
						Bucket:             "foo",
						Key:                "agreatkeyforyou",
						Region:             "us-east-1",
					},
					Vault: src.Vault,
				},
				Params: types.OutParams{
					Context: "foo",
					Dir:     "source/terraform",
				},
			},
			assert: func(out *Out, req *types.OutRequest, ansible *Ansible, err error) {
				assert.NoError(t, err)

				extraVars, err := ansible.prepareRun()
				assert.NoError(t, err)
				defer os.Remove(extraVars.Name())

				vars, err := ioutil.ReadFile(extraVars.Name())
				assert.NoError(t, err)
				assert.Equal(t, "agreatkeyforyou", gjson.GetBytes(vars, "storage.key").String(), "invalid extra var: storage.key")
			},
		},
		{
			desc: "context",
			req: &types.OutRequest{
				Source: src,
				Params: types.OutParams{
					InputMapping: `
					context = "qa1-use2"
					workspace = "qa1-use2-monitoring"
					`,
					Context:   `${!json("context")}`,
					Workspace: `${!json("workspace")}`,
					Dir:       "source/terraform",
				},
			},
			assert: func(out *Out, req *types.OutRequest, ansible *Ansible, err error) {
				assert.NoError(t, err)

				extraVars, err := ansible.prepareRun()
				assert.NoError(t, err)
				defer os.Remove(extraVars.Name())

				vars, err := ioutil.ReadFile(extraVars.Name())
				assert.NoError(t, err)
				assert.Equal(t, "qa1-use2", gjson.GetBytes(vars, "context").String(), "invalid extra var: context")
				assert.Equal(t, "qa1-use2-monitoring", gjson.GetBytes(vars, "terraform_workspace").String(), "invalid extra var: terraform_workspace")
			},
		},
		{
			desc: "vars_mapping",
			req: &types.OutRequest{
				Source: src,
				Params: types.OutParams{
					InputMapping: `
					context = "qa1-use2"
					workspace = "qa1-use2-monitoring"
					vars.region = "us-east-2"
					vars.list = [1,2,3]
					`,
					Context: `${!json("context")}`,
					Dir:     "source/terraform",
					VarsMapping: `
					root = vars
					foo = "bar"
					test_bool = true
					`,
				},
			},
			assert: func(out *Out, req *types.OutRequest, ansible *Ansible, err error) {
				assert.NoError(t, err)

				extraVars, err := ansible.prepareRun()
				assert.NoError(t, err)
				defer os.Remove(extraVars.Name())

				vars, err := ioutil.ReadFile(extraVars.Name())
				assert.NoError(t, err)
				fmt.Println(string(vars))
				assert.Equal(t, "qa1-use2", gjson.GetBytes(vars, "context").String(), "invalid extra var: context")
				assert.True(t, gjson.GetBytes(vars, "terraform_vars").IsObject())
				assert.Equal(t, "us-east-2", gjson.GetBytes(vars, "terraform_vars.region").String())
				assert.Equal(t, "bar", gjson.GetBytes(vars, "terraform_vars.foo").String())
				assert.Equal(t, float64(1), gjson.GetBytes(vars, "terraform_vars.list.0").Float())
				assert.Equal(t, float64(2), gjson.GetBytes(vars, "terraform_vars.list.1").Float())
				assert.Equal(t, float64(3), gjson.GetBytes(vars, "terraform_vars.list.2").Float())
				assert.True(t, gjson.GetBytes(vars, "terraform_vars.test_bool").Bool())
			},
		},
		{
			desc: "vars_mapping only bool",
			req: &types.OutRequest{
				Source: src,
				Params: types.OutParams{
					Context: "foo",
					Dir:     "source/terraform",
					VarsMapping: `
					test_bool = true
					`,
				},
			},
			assert: func(out *Out, req *types.OutRequest, ansible *Ansible, err error) {
				assert.NoError(t, err)

				extraVars, err := ansible.prepareRun()
				assert.NoError(t, err)
				defer os.Remove(extraVars.Name())

				vars, err := ioutil.ReadFile(extraVars.Name())
				assert.NoError(t, err)
				fmt.Println(string(vars))
				assert.Equal(t, "foo", gjson.GetBytes(vars, "context").String(), "invalid extra var: context")
				assert.True(t, gjson.GetBytes(vars, "terraform_vars").IsObject())
				assert.True(t, gjson.GetBytes(vars, "terraform_vars.test_bool").Bool())
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			out := c.out
			if out == nil {
				out = &Out{
					args: []string{"/out", "/tmp/build/put"},
					env: types.Environment{
						ID:             "2199",
						Job:            "testing",
						Name:           "217",
						Pipeline:       "example-component",
						Team:           "sre",
						ATCExternalURL: "http://127.0.0.1:8080",
					},
				}
			}
			ansible, err := out.ansiblePlaybookCmd(c.req)
			c.assert(out, c.req, ansible, err)
		})
	}
}
