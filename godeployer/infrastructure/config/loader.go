// ============================================================
// 文件：loader.go
// 作用：⚙️ 配置加载器——把 YAML 配置文件变成 Go 数据！
//
// 配置文件就像"菜谱"——告诉程序该怎么运行。
// GoDeploy 使用 YAML 格式的配置文件（比 JSON 更好看、支持注释）。
//
// 加载流程：
// 1. 读取主配置文件（config.yaml）
// 2. 替换环境变量（比如 $HOME, ${PORT} 等）
// 3. 解析成 Config 结构体
// 4. 扫描项目配置目录（project_config_dir）下的所有 .yaml 文件
// 5. 把每个项目配置也加载进来，合并到 Config.Projects 中
// 6. 如果某个服务器没配 SSH 私钥，自动用全局的 SSH 私钥
//
// 为什么要把配置拆成多个文件？
// 每个项目一个文件，方便管理。就像每个菜谱一张卡片~
// ============================================================

// Package config 提供 YAML 配置文件的加载与解析能力。
package config

import (
	"fmt"       // ✏️ 格式化
	"os"        // 💻 文件读写
	"path/filepath" // 📁 路径处理

	"deploy/godeployer/domain" // 📋 使用领域层的数据结构

	"gopkg.in/yaml.v3" // 📄 YAML 解析器（把 YAML 转成 Go 结构体）
)

// LoadConfig 读取主配置文件并扫描加载所有项目配置，同时替换环境变量。
func LoadConfig(path string) (*domain.Config, error) {
	// 1️⃣ 读取主配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read main config: %w", err)
	}

	// 2️⃣ 替换环境变量
	// os.ExpandEnv 会把配置中的 $VAR 或 ${VAR} 替换成环境变量的值
	// 比如 DB_PATH: "$HOME/godeployer.db" → DB_PATH: "/home/user/godeployer.db"
	expandedData := []byte(os.ExpandEnv(string(data)))

	// 3️⃣ 解析 YAML → Go 结构体
	var config domain.Config
	if err := yaml.Unmarshal(expandedData, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal main config: %w", err)
	}

	config.Projects = make(map[string]domain.ProjectConfig)

	// 4️⃣ 扫描项目配置目录
	if config.ProjectConfigDir != "" {
		files, err := os.ReadDir(config.ProjectConfigDir)
		if err == nil {
			for _, file := range files {
				// 只处理 .yaml 和 .yml 文件
				if !file.IsDir() && (filepath.Ext(file.Name()) == ".yaml" ||
					filepath.Ext(file.Name()) == ".yml") {
					projPath := filepath.Join(config.ProjectConfigDir, file.Name())
					projData, err := os.ReadFile(projPath)
					if err != nil {
						continue // 读不到就跳过
					}
					expandedProjData := []byte(os.ExpandEnv(string(projData)))
					var projConfig domain.ProjectConfig
					if err := yaml.Unmarshal(expandedProjData, &projConfig);
						err == nil && projConfig.ID != "" {
						// 5️⃣ 如果服务器没配 SSHKey，就用全局的
						for envIdx := range projConfig.Environments {
							for srvIdx := range projConfig.Environments[envIdx].Servers {
								if projConfig.Environments[envIdx].Servers[srvIdx].SSHKeyPath == "" {
									projConfig.Environments[envIdx].Servers[srvIdx].SSHKeyPath =
										config.Global.SSHKeyPath
								}
							}
						}
						// 把项目配置加入 map
						config.Projects[projConfig.ID] = projConfig
					}
				}
			}
		}
	}

	return &config, nil
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: YAML 和 JSON 有什么区别？
//    A: YAML 更可读，支持注释（#），写配置更方便！
//       JSON 更严格，适合机器之间的数据传输~
//
// 2. Q: os.ExpandEnv 的作用？
//    A: 把 $VAR 替换成环境变量。比如 PORT=9090，
//       配置里的 :${PORT} 就变成 :9090~
//
// 中级：
// 3. Q: 为什么项目配置要放在单独的文件里？
//    A: 每个项目一个文件，方便增删改查。
//       如果你有 50 个项目，放在一个文件里会非常长~
//
// 4. Q: 为什么加载失败时不 return error，而是 continue？
//    A: 一个项目配置坏了不影响其他项目！
//       继续加载后面的，让坏的那个项目不可用就好。
//       这叫"部分失败容忍"~
//
// 高级：
// 5. Q: yaml.Unmarshal 和 json.Unmarshal 有什么异同？
//    A: 都是"反序列化"——把文本变成结构体。
//       YAML 支持更多特性（别名、锚点、多文档），
//       但 Go 的 yaml.v3 库对 struct 标签的支持跟 json 很相似~
// ============================================================
