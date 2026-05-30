package godeployer

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Global            GlobalConfig             `yaml:"global"`
	ProjectConfigDir  string                   `yaml:"project_config_dir"`
	Projects          map[string]ProjectConfig `yaml:"-"`
}

type GlobalConfig struct {
	SQLitePath    string `yaml:"sqlite_path"`
	LogPath       string `yaml:"log_path"`
	WorkspacePath string `yaml:"workspace_path"`
	SSHKeyPath    string `yaml:"ssh_key_path"`
	ServerPort    int    `yaml:"server_port"`
	JWTSecret     string `yaml:"jwt_secret"`
	// @Ref: docs/sps/plans/20260529_diff_ux_loading_plan.md | @Date: 2026-05-29
	DiffMaxSizeKB  int `yaml:"diff_max_size_kb"`
	DiskMinSpaceMB int `yaml:"disk_min_space_mb"`
	TaskRetainMax  int `yaml:"task_retain_max"`
	TaskRetainDays int `yaml:"task_retain_days"`
}

type ProjectConfig struct {
	ID            string              `yaml:"id" json:"id"`
	Name          string              `yaml:"name" json:"name"`
	Repo          string              `yaml:"repo" json:"repo"`
	WebhookSecret string              `yaml:"webhook_secret" json:"webhook_secret"`
	Branch        string              `yaml:"branch" json:"branch"`
	Exclude       []string            `yaml:"exclude" json:"exclude"`
	SharedFiles   []string            `yaml:"shared_files" json:"shared_files"`
	SharedDirs    []string            `yaml:"shared_dirs" json:"shared_dirs"`
	Build         BuildConfig         `yaml:"build" json:"build"`
	Environments  []EnvironmentConfig `yaml:"environments" json:"environments"`
}

type BuildConfig struct {
	BeforeSync []string `yaml:"before_sync" json:"before_sync"`
}

type EnvironmentConfig struct {
	ID            string         `yaml:"id" json:"id"`
	Name          string         `yaml:"name" json:"name"`
	DefaultBranch string         `yaml:"default_branch" json:"default_branch"`
	KeepReleases  int            `yaml:"keep_releases" json:"keep_releases"`
	Servers       []ServerConfig `yaml:"servers" json:"servers"`
	BeforeSymlink []string       `yaml:"before_symlink" json:"before_symlink"`
	AfterSymlink  []string       `yaml:"after_symlink" json:"after_symlink"`
}

type ServerConfig struct {
	Host       string `yaml:"host" json:"host"`
	Port       int    `yaml:"port" json:"port"`
	User       string `yaml:"user" json:"user"`
	DeployTo   string `yaml:"deploy_to" json:"deploy_to"`
	SSHKeyPath string `yaml:"ssh_key_path" json:"ssh_key_path"`
}

// LoadConfig 读取主配置文件并扫描加载所有项目配置，同时替换环境变量。
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read main config: %w", err)
	}

	// 替换环境变量
	expandedData := []byte(os.ExpandEnv(string(data)))

	var config Config
	if err := yaml.Unmarshal(expandedData, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal main config: %w", err)
	}

	config.Projects = make(map[string]ProjectConfig)

	// 扫描 project_config_dir 目录下的 yaml 文件
	if config.ProjectConfigDir != "" {
		files, err := os.ReadDir(config.ProjectConfigDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && (filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml") {
					projPath := filepath.Join(config.ProjectConfigDir, file.Name())
					projData, err := os.ReadFile(projPath)
					if err != nil {
						continue
					}
					expandedProjData := []byte(os.ExpandEnv(string(projData)))
					var projConfig ProjectConfig
					if err := yaml.Unmarshal(expandedProjData, &projConfig); err == nil && projConfig.ID != "" {
						for envIdx := range projConfig.Environments {
							for srvIdx := range projConfig.Environments[envIdx].Servers {
								if projConfig.Environments[envIdx].Servers[srvIdx].SSHKeyPath == "" {
									projConfig.Environments[envIdx].Servers[srvIdx].SSHKeyPath = config.Global.SSHKeyPath
								}
							}
						}
						config.Projects[projConfig.ID] = projConfig
					}
				}
			}
		}
	}

	return &config, nil
}
