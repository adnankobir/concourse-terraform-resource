package types

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Environment describes the runtime environment provided by concourse
type Environment struct {
	ID             string `envconfig:"BUILD_ID" required:"true"`
	Name           string `envconfig:"BUILD_NAME" required:"true"`
	Job            string `envconfig:"BUILD_JOB_NAME" required:"true"`
	Pipeline       string `envconfig:"BUILD_PIPELINE_NAME" required:"true"`
	Team           string `envconfig:"BUILD_TEAM_NAME" required:"true"`
	ATCExternalURL string `envconfig:"ATC_EXTERNAL_URL" required:"true"`
}

// Source describes the resource configuration
type Source struct {
	Component string `json:"component"`
	//Debug      bool              `json:"debug"`
	Envs       map[string]string `json:"envs"`
	PrivateKey string            `json:"private_key,omitempty"`
	Storage    Storage           `json:"storage,omitempty"`
	Vault      VaultSource       `json:"vault"`
}

// Validate resource runtime configuration
func (s *Source) Validate() error {
	if err := s.Vault.Validate(); err != nil {
		return fmt.Errorf("invalid vault config: %v", err)
	}
	if err := s.Storage.Validate(); err != nil {
		return fmt.Errorf("invalid storage config: %v", err)
	}
	return nil
}

// AssignFallbackValues assigns fallback values for Component and Storage.Key based on runtime configuration
func (s *Source) AssignFallbackValues(team, component string) {
	if s.Component == "" {
		s.Component = component
	}
	if s.Storage.Key == "" {
		s.Storage.Key = fmt.Sprintf("%s/%s/concourse-terraform-resource/version.tgz", team, s.Component)
	}
}

// EncryptField allows this value to be either bool or string,
// but coerces the value into a string type as that is used by
// our Ansible playbook
type EncryptField string

// UnmarshalJSON implements the interface expected of a nested
// JSON field and allows a string value to be parsed from bool or string
func (f *EncryptField) UnmarshalJSON(data []byte) (err error) {
	if enc, err := strconv.ParseBool(string(data)); err == nil {
		str := fmt.Sprintf("%t", enc)
		*f = EncryptField(str)
		return nil
	}
	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(str), f)
}

// Storage describes resource storage configuration
type Storage struct {
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
	Bucket             string `json:"bucket"`
	Key                string `json:"key"`
	Region             string `json:"region"`
}

// Validate storage configuration
func (s *Storage) Validate() error {
	if s.AWSAccessKeyID == "" {
		return fmt.Errorf("missing aws_access_key_id")
	}
	if s.AWSSecretAccessKey == "" {
		return fmt.Errorf("missing aws_secret_access_key")
	}
	if s.Bucket == "" {
		return fmt.Errorf("missing bucket")
	}
	if s.Region == "" {
		return fmt.Errorf("missing region")
	}
	return nil
}

// VaultSource describes requried vault runtime configuration
type VaultSource struct {
	Addr     string `json:"addr"`
	RoleID   string `json:"role_id"`
	SecretID string `json:"secret_id"`
}

// Validate resource runtime configuration
func (s *VaultSource) Validate() error {
	if s.Addr == "" {
		return fmt.Errorf("missing vault addr")
	}
	if s.RoleID == "" {
		return fmt.Errorf("missing vault role_id")
	}
	if s.SecretID == "" {
		return fmt.Errorf("missing vault secret_id")
	}
	return nil
}

// Version describes a Concourse version
type Version struct {
	Key       string `json:"key"`
	VersionID string `json:"version_id"`
}

// Validate a version value
func (v *Version) Validate() error {
	if v.Key == "" {
		return fmt.Errorf("missing key")
	}
	if v.VersionID == "" {
		return fmt.Errorf("missing version_id")
	}
	return nil
}

// Metadata describes an individual metadata entry
type Metadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CheckRequest describes the input to a check operation
type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version,omitempty"`
}

// InRequest describes the input to a get operation
type InRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

// Validate in request
func (r *InRequest) Validate() error {
	if err := r.Source.Validate(); err != nil {
		return fmt.Errorf("invalid source: %v", err)
	}
	if err := r.Version.Validate(); err != nil {
		return fmt.Errorf("invalid version: %v", err)
	}
	return nil
}

// InResponse describes the output from a successful get operation
type InResponse struct {
	Version  Version    `json:"version"`
	Metadata []Metadata `json:"metadata"`
}

// OutRequest describes the input to a put operation
type OutRequest struct {
	Source Source    `json:"source"`
	Params OutParams `json:"params"`
}

// Envs returns the combined usm of env vars
func (r *OutRequest) Envs() map[string]string {
	envs := r.Source.Envs
	if envs == nil {
		envs = map[string]string{}
	}
	if r.Params.Envs != nil && len(r.Params.Envs) > 0 {
		for k, v := range r.Params.Envs {
			envs[k] = v
		}
	}
	return envs
}

// PrivateKey extracts private key
func (r *OutRequest) PrivateKey() (string, bool) {
	if r.Params.PrivateKey != "" {
		return r.Params.PrivateKey, true
	}
	if r.Source.PrivateKey != "" {
		return r.Source.PrivateKey, true
	}
	return "", false
}

// Validate out request
func (r *OutRequest) Validate() error {
	if err := r.Source.Validate(); err != nil {
		return fmt.Errorf("invalid source: %v", err)
	}
	if err := r.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %v", err)
	}
	return nil
}

// OutParams describes job-level configuration for a put operation
type OutParams struct {
	Context        string            `json:"context"`
	Dir            string            `json:"dir"`
	Envs           map[string]string `json:"envs"`
	InputMapping   string            `json:"input_mapping"`
	PlanOnly       bool              `json:"plan_only,omitempty"`
	PrivateKey     string            `json:"private_key,omitempty"`
	ReleaseVersion string            `json:"release_version"`
	VarFiles       []string          `json:"var_files"`
	VarsMapping    string            `json:"vars_mapping"`
	Workspace      string            `json:"workspace"`
}

// Validate out parameters
func (p *OutParams) Validate() error {
	if p.Context == "" {
		return fmt.Errorf("missing required parameter (context)")
	}
	if p.Dir == "" {
		return fmt.Errorf("missing required parameter (dir)")
	}
	return nil
}

// OutResponse describes the output from a successful put operation
type OutResponse struct {
	Version  Version    `json:"version"`
	Metadata []Metadata `json:"metadata"`
}
