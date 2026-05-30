package godeployer

import (
	"fmt"
	"os"
	"path/filepath"

	"deploy/godeployer/domain"

	"gopkg.in/yaml.v3"
)

// LoadConfig 读取主配置文件并扫描加载所有项目配置，同时替换环境变量。
func LoadConfig(path string) (*domain.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read main config: %w", err)
	}

	// 替换环境变量
	expandedData := []byte(os.ExpandEnv(string(data)))

	var config domain.Config
	if err := yaml.Unmarshal(expandedData, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal main config: %w", err)
	}

	config.Projects = make(map[string]domain.ProjectConfig)

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
					var projConfig domain.ProjectConfig
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
