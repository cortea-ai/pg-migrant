package config

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/jackc/pgx/v4"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type Variable struct {
	Name    string `hcl:"name,label"`
	Default string `hcl:"default,optional"`
}

type Locals struct {
	Values map[string]cty.Value `hcl:",remain"`
}

type GitHubConfig struct {
	Owner        string `hcl:"owner" cty:"owner"`
	Repo         string `hcl:"repo" cty:"repo"`
	TargetBranch string `hcl:"target_branch" cty:"target_branch"`
}

type Env struct {
	Name           string       `hcl:"name,label"`
	DBUrl          string       `hcl:"db_url"`
	MigrationDir   string       `hcl:"migration_dir,optional" default:"./migrations"`
	SchemaFiles    []string     `hcl:"schema_files"`
	GitHubConfig   GitHubConfig `hcl:"github_config"`
	ExcludeSchemas []string     `hcl:"exclude_schemas"`
}

type Config struct {
	Variables   []Variable `hcl:"variable,block"`
	Locals      *Locals    `hcl:"locals,block"`
	Envs        []Env      `hcl:"env,block"`
	SelectedEnv Env
}

func GetConfig(filePath string, env string, vars Vars) (*Config, error) {
	var config Config

	parser := hclparse.NewParser()
	hclFile, diags := parser.ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file: %w", diags)
	}
	variables := make(map[string]cty.Value)
	evalCtx := &hcl.EvalContext{
		Functions: map[string]function.Function{
			"getenv": getEnvFunc,
		},
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(variables),
		},
	}
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "variable", LabelNames: []string{"name"}},
			{Type: "locals", LabelNames: []string{}},
			{Type: "env", LabelNames: []string{"name"}},
		},
	}
	content, diags := hclFile.Body.Content(schema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse body content: %w", diags)
	}

	// Collect variables
	for _, block := range content.Blocks.OfType("variable") {
		varName := block.Labels[0]
		attrs, diags := block.Body.JustAttributes()
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse attributes of variable '%s': %w", varName, diags)
		}
		if val, ok := vars[varName]; ok {
			variables[varName] = cty.StringVal(val)
			continue
		}
		defaultAttr, ok := attrs["default"]
		if !ok {
			continue
		}
		defaultVal, diags := defaultAttr.Expr.Value(evalCtx)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to evaluate default value of '%s': %w", varName, diags)
		}
		variables[varName] = defaultVal
	}
	evalCtx.Variables["var"] = cty.ObjectVal(variables)

	// Collect locals
	if len(content.Blocks.OfType("locals")) > 0 {
		block := content.Blocks.OfType("locals")[0]
		attrs, diags := block.Body.JustAttributes()
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse locals block: %w", diags)
		}
		localVals := make(map[string]cty.Value)
		for name, attr := range attrs {
			val, diags := attr.Expr.Value(evalCtx)
			if diags.HasErrors() {
				return nil, fmt.Errorf("failed to evaluate local value '%s': %w", name, diags)
			}
			localVals[name] = val
		}
		evalCtx.Variables["local"] = cty.ObjectVal(localVals)
	}

	err := hclsimple.DecodeFile(filePath, evalCtx, &config)
	if err != nil {
		return nil, err
	}

	if env != "" {
		for _, e := range config.Envs {
			if e.Name == env {
				return &Config{
					Variables:   config.Variables,
					Envs:        []Env{e},
					SelectedEnv: e,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("environment %q not found in config", env)
}

func (conf *Config) GetDBUrl() string {
	return conf.SelectedEnv.DBUrl
}

func (conf *Config) GetDBConfig() (*pgx.ConnConfig, error) {
	connConfig, err := pgx.ParseConfig(conf.SelectedEnv.DBUrl)
	if err != nil {
		return nil, err
	}
	return connConfig, nil
}

func (conf *Config) GetMigrationDir() string {
	return conf.SelectedEnv.MigrationDir
}

func (conf *Config) GetSchemaFiles() []string {
	return conf.SelectedEnv.SchemaFiles
}

func (conf *Config) GetGitHubConfig() GitHubConfig {
	return conf.SelectedEnv.GitHubConfig
}

func (conf *Config) GetExcludeSchemas() []string {
	return conf.SelectedEnv.ExcludeSchemas
}

var getEnvFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "key",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		val, ok := os.LookupEnv(args[0].AsString())
		if !ok {
			return cty.StringVal(""), fmt.Errorf("environment variable %q not found", args[0].AsString())
		}
		return cty.StringVal(val), nil
	},
})
