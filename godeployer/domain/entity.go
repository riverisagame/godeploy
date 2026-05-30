package domain

import (
	"context"
	"time"
)

// DeployJob 部署任务实体
type DeployJob struct {
	Ctx         context.Context
	Cancel      context.CancelFunc
	TaskID      int64
	Config      *Config
	LogFilePath string
}

// Config 配置相关的实体
type Config struct {
	Global           GlobalConfig             `yaml:"global"`
	ProjectConfigDir string                   `yaml:"project_config_dir"`
	Projects         map[string]ProjectConfig `yaml:"-"`
}

type GlobalConfig struct {
	SQLitePath     string `yaml:"sqlite_path"`
	LogPath        string `yaml:"log_path"`
	WorkspacePath  string `yaml:"workspace_path"`
	SSHKeyPath     string `yaml:"ssh_key_path"`
	ServerPort     int    `yaml:"server_port"`
	JWTSecret      string `yaml:"jwt_secret"`
	DiffMaxSizeKB  int    `yaml:"diff_max_size_kb"`
	DiskMinSpaceMB int    `yaml:"disk_min_space_mb"`
	TaskRetainMax  int    `yaml:"task_retain_max"`
	TaskRetainDays int    `yaml:"task_retain_days"`
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

// UserResponse 及相关 DTO
type UserResponse struct {
	ID                 int       `json:"id"`
	Username           string    `json:"username"`
	Role               string    `json:"role"`
	CreatedAt          time.Time `json:"created_at"`
	BoundGitAuthors    string    `json:"bound_git_authors"`
	RestrictGitAuthors bool      `json:"restrict_git_authors"`
	PermittedProjects  string    `json:"permitted_projects"`
}

// GitCommit 实体
type GitCommit struct {
	Hash      string `json:"hash"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}
