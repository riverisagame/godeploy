package domain

import "errors"

type Environment struct {
	Name       string `json:"name"`
	Branch     string `json:"branch"`
	DeployType string `json:"deploy_type"`
	PreDeploy  string `json:"pre_deploy"`
	PostDeploy string `json:"post_deploy"`
	ServerIDs  []uint `json:"server_ids"`
	DeployPath string `json:"deploy_path"`
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
	}
	p.Environments = append(p.Environments, env)
	return nil
}
