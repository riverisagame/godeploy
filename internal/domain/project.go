package domain

import "errors"

// @Ref: docs/sps/plans/20260720_env_vars_ir.md | @Date: 2026-07-20
type EnvVar struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
}

// CommitInfo 描述 Git 提交的信息
type CommitInfo struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

type Environment struct {
	ID         uint     `json:"id"`
	Name       string   `json:"name"`
	Branch     string   `json:"branch"`
	DeployType string   `json:"deploy_type"`
	PreDeploy  string   `json:"pre_deploy"`
	PostDeploy string   `json:"post_deploy"`
	ServerIDs  []uint   `json:"server_ids"`
	DeployPath string   `json:"deploy_path"`
	EnvVars    []EnvVar `json:"env_vars"`
}

type Project struct {
	ID           uint           `json:"id"`
	Name         string         `json:"name"`
	RepoURL      string         `json:"repo_url"`
	KeepReleases int            `json:"keep_releases"`
	Environments []*Environment `json:"environments"`
}

func NewProject(name, repoURL string) (*Project, error) {
	if name == "" {
		return nil, errors.New("project name cannot be empty")
	}
	if repoURL == "" {
		return nil, errors.New("repo URL cannot be empty")
	}

	return &Project{
		Name:         name,
		RepoURL:      repoURL,
		KeepReleases: 5,
		Environments: make([]*Environment, 0),
	}, nil
}

func (p *Project) AddEnvironment(name, branch, deployType string) error {
	if name == "" {
		return errors.New("environment name cannot be empty")
	}
	// @Ref: docs/sps/plans/20260721_production_fix_ir.md Task 4.6 | @Date: 2026-07-21
	for _, e := range p.Environments {
		if e.Name == name {
			return errors.New("environment with this name already exists")
		}
	}
	if branch == "" {
		branch = "main"
	}
	if deployType == "" {
		deployType = "symlink"
	}

	env := &Environment{
		Name:       name,
		Branch:     branch,
		DeployType: deployType,
		ServerIDs:  make([]uint, 0),
		DeployPath: "/var/www/pdeploy",
		EnvVars:    make([]EnvVar, 0),
	}
	p.Environments = append(p.Environments, env)
	return nil
}

func (e *Environment) AddEnvVar(key, value string, isSecret bool) {
	e.EnvVars = append(e.EnvVars, EnvVar{
		Key:      key,
		Value:    value,
		IsSecret: isSecret,
	})
}
