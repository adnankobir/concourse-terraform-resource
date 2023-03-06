# Terraform Resource
A [concourse](https://concourse-ci.org/) resource for working with terraform.

```yaml
resource_types:
  - name: terraform
    type: registry-image
    source:
      repository: concourse-terraform-resource
      tag: 1.3.9
      aws_access_key_id: ((ecr.aws_access_key_id))
      aws_secret_access_key: ((ecr.aws_secret_access_key))
      aws_region: ((ecr.region))
```

## Source Configuration

**Basic**
```yaml
resources:
  - name: terraform
    type: terraform
    icon: terraform
    source:
      envs:
        FOO: bar
        TF_VAR_foo: ((my-team-secret))
      private_key: ((git-private-key))
      storage: ((s3))
      vault: ((vault))
```

### `component`

Component name, if provided, should match workspace prefix in your Terraform code.

Type: `string`
Default: _name of Concourse pipeline_

### `envs`

Map of environment variables to pass to terraform. Values support [interpolation functions](https://www.benthos.dev/docs/configuration/interpolation#bloblang-queries)

Type: `map(string)`
Default: `{}`

```yaml
source:
  envs:
    MY_VAR: FOO
    GIT_SHA: ${!file("source/.git/short_ref").string().trim()}
```

### `private_key`

SSH private key, if provided, a new SSH agent will be spawned and used by terraform for cloning private modules. This field supports [interpolation functions](https://www.benthos.dev/docs/configuration/interpolation#bloblang-queries)

Type: `string`
Default: `""`


### `storage`

Amazon S3 configuration for persistence of Terraform output between job steps.

Type: `object`
Required: `true`


### `vault`

vault configuration

Type: `object`
Required: `true`

## Behavior

### Check
`no-op`

### In
`no-op`

### Out

**Parameters**

### `context`

Deployment context. This field supports [interpolation functions](https://www.benthos.dev/docs/configuration/interpolation#bloblang-queries)

Type: `string`
Required: `true`

### `dir`

Relative path to terraform module root.

Type: `string`
Required: `true`

### `envs`

Map of environment variables to pass to terraform. If specified, overrides source level field of the same name. Values support [interpolation functions](https://www.benthos.dev/docs/configuration/interpolation#bloblang-queries)

Type: `map(string)`
Optional: `true`

### `input_mapping`

An optional [bloblang mapping](https://www.benthos.dev/docs/guides/bloblang/about#assignment) that serves as the context for all other resource and put parameters that support mapping/interpolation. Useful if other parameters share required data that must be computed/extracted from the file system.

Type: `string`
Default: `{}`

```yaml
put: terraform
params:
  # extract input context from filesystem
  input_mapping: file("gate/item.json").parse_json() 
  # lookup input.vars.context, default to use1-prod-1
  context: ${!json("vars.context).or("use1-prod-1")}
  # will use input.vars or default to {}
  vars_mapping: vars.or({}) 
```

### `plan_only`

An optional flag to disable Terraform `apply` steps, intended to be used for manual verification of a Terraform plan.

Type: `bool`
Default: `false`

### `private_key`

SSH private key, if provided, a new SSH agent will be spawned and used by terraform for cloning private modules. This field supports [interpolation functions](https://www.benthos.dev/docs/configuration/interpolation#bloblang-queries)

Type: `string`
Optional: `true`

### `var_files`

Path to Terraform variables file, relative to the resource working directory. Supports [interpolation functions](https://www.benthos.dev/docs/configuration/interpolation#bloblang-queries)

Type: `list(string)`
Optional: `true`

### `vars_mapping`

An optional [bloblang mapping](https://www.benthos.dev/docs/guides/bloblang/about#assignment) for specifying additional variables to pass to terraform.

Type: `string`
Optional: `true`

```yaml
put: terraform
params:
  # extract input context from filesystem
  input_mapping: file("gate/item.json").parse_json() 
  vars_mapping: |
    let workspaceVars = file("source/terraform/workspaces/%s.tfvars.json".format(workspace)).parse_json()
    root = $workspaceVars.merge(vars.or({}))
    region = "us-east-2"
```

### `workspace`

Terraform workspace (defaults to context). This field supports [interpolation functions](https://www.benthos.dev/docs/configuration/interpolation#bloblang-queries)

Type: `string`
Optional: `true`

## testing

### build docker image
- Make sure you have [goreleaser](https://goreleaser.com/) installed and run
```
goreleaser release --rm-dist --snapshot --skip-publish
```

- Fetch the github token from vault

- Build the docker image.
```
docker build -t concourse-terraform-resource . --build-arg GITHUB_TOKEN=<github_token_from_vault>
```

### test locally
- There are some variables that the test pipeline needs to run. To mimic concourse vars create `/tmp/extra_vars.json` file with the following content:
```
{
  "component": "concourse-terraform-resource-test",
  "context": "use1-prod-1",
  "concourse_atc_external_url": "https://127.0.0.1",
  "concourse_build_team": "sre",
  "terraform_path": "/tmp/terraform",
  "terraform_workspace": "use1-prod-1",
  "workdir": "/tmp"
}
```
- Run the docker image
```
docker run -it -v /<local_path>/concourse-terraform-resource/examples/test/terraform:/tmp/terraform -v ~/tmp/ansible/extra_vars.json:/tmp/extra_vars.json --workdir /tmp/terraform -e "VAULT_TOKEN=<vault_token>" -e "VAULT_ADDR=<vault_address>" concourse-terraform-resource
```

- Execute ansible playbook
```
ansible-playbook /opt/ansible/out.yaml -e @/tmp/extra_vars.json
```

### test on concourse
- Tag the image with unique tag
```
docker tag concourse-terraform-resource:latest <ecr>.dkr.ecr.<region>.amazonaws.com/concourse-terraform-resource:<my-unique-tag>
```
- Log in to ecr
```
aws ecr get-login-password --region <region> | docker login --username AWS --password-stdin <account>.dkr.ecr.<region>.amazonaws.com
```
- Push the image to ecr
```
docker push <account>.dkr.ecr.<region>.amazonaws.com/concourse-terraform-resource:<my-unique-tag>
```
- Change the image tag in the `examples/test/pipeline.yaml` on the `pipeline` branch under `resource_types.source.tag` to `<my-unique-tag>`
- Update pipeline with fly 
```
fly -t <concourse-target> crt -r concourse-terraform-resource-test/terraform
```
- Manually trigger the pipeline from the plus sign in the top right corner
