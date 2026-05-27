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
}

type ProjectConfig struct {
	ID            string              `yaml:"id"`
	Name          string              `yaml:"name"`
	Repo          string              `yaml:"repo"`
	WebhookSecret string              `yaml:"webhook_secret"`
	Branch        string              `yaml:"branch"`
	Exclude       []string            `yaml:"exclude"`
	SharedFiles   []string            `yaml:"shared_files"`
	SharedDirs    []string            `yaml:"shared_dirs"`
	Build         BuildConfig         `yaml:"build"`
	Environments  []EnvironmentConfig `yaml:"environments"`
}

type BuildConfig struct {
	BeforeSync []string `yaml:"before_sync"`
}

type EnvironmentConfig struct {
	ID            string         `yaml:"id"`
	Name          string         `yaml:"name"`
	DefaultBranch string         `yaml:"default_branch"`
	KeepReleases  int            `yaml:"keep_releases"`
	Servers       []ServerConfig `yaml:"servers"`
	BeforeSymlink []string       `yaml:"before_symlink"`
	AfterSymlink  []string       `yaml:"after_symlink"`
}

type ServerConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	User       string `yaml:"user"`
	DeployTo   string `yaml:"deploy_to"`
	SSHKeyPath string `yaml:"ssh_key_path"`
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
						config.Projects[projConfig.ID] = projConfig
					}
				}
			}
		}
	}

	return &config, nil
}
