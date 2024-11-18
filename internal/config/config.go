package config

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/jackc/pgx/v4"
	"github.com/zclconf/go-cty/cty"
)

type Variable struct {
	Name    string `hcl:"name,label"`
	Default string `hcl:"default,optional"`
}

type Locals struct {
	Values map[string]cty.Value `hcl:",remain"`
}

type Env struct {
	Name         string   `hcl:"name,label"`
	DBUrl        string   `hcl:"db_url"`
	MigrationDir string   `hcl:"migration_dir,optional" default:"./migrations"`
	SchemaFiles  []string `hcl:"schema_files"`
	GitRepo      string   `hcl:"git_repo"`
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

func (conf *Config) GetGitRepo() string {
	return conf.SelectedEnv.GitRepo
}
