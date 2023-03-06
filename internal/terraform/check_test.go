package terraform

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/adnankobir/concourse-terraform-resource/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	cases := []struct {
		desc    string
		payload []byte
		args    []string
		assert  func([]types.Version, error)
	}{
		{
			desc:    "no version",
			payload: []byte("{}"),
			args:    []string{"check", "/tmp"},
			assert: func(res []types.Version, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 0)
			},
		},
		{
			desc:    "null version",
			payload: []byte(`{"version":null}`),
			args:    []string{"check", "/tmp"},
			assert: func(res []types.Version, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 0)
			},
		},
		{
			desc:    "non null version",
			payload: []byte(`{"version":{"key":"my-version.tgz","version_id":"iQTUjehl1EsngSfrax_L.4wL4qcsHTYx"}}`),
			args:    []string{"check", "/tmp"},
			assert: func(res []types.Version, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 1)
				assert.Equal(t, res[0].Key, "my-version.tgz")
				assert.Equal(t, res[0].VersionID, "iQTUjehl1EsngSfrax_L.4wL4qcsHTYx")
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			stderr := &bytes.Buffer{}
			stdout := &bytes.Buffer{}
			rerr := NewCheck(bytes.NewBuffer(c.payload), stderr, stdout, c.args).Execute()
			var res []types.Version
			if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
				c.assert(nil, err)
			} else {
				c.assert(res, rerr)
			}
		})
	}
}
