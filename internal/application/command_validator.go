package application

import (
	"errors"
	"regexp"
	"strings"
)

var allowedCommands = map[string]bool{
	"npm":       true,
	"yarn":      true,
	"pnpm":      true,
	"make":      true,
	"docker":    true,
	"pm2":       true,
	"systemctl": true,
	"go":        true,
	"python":    true,
	"python3":   true,
	"pip":       true,
	"echo":      true,
}

// ValidateAndFormat 验证并格式化命令，防止注入
func ValidateAndFormat(baseCmd string, args map[string]string) (string, error) {
	cmd := strings.TrimSpace(baseCmd)
	if cmd == "" {
		return "", errors.New("empty command")
	}

	parts := strings.Fields(cmd)
	if !allowedCommands[parts[0]] {
		return "", errors.New("command not in whitelist: " + parts[0])
	}

	// 检查是否有非法注入字符 (; & | $ < > `)
	// 但允许必要的字符，比如短划线，下划线，斜杠等
	injectionPattern := regexp.MustCompile(`[;&|$\<\>\x60]`)
	if injectionPattern.MatchString(cmd) {
		return "", errors.New("command contains illegal characters")
	}

	// 模板替换 {{key}}
	for k, v := range args {
		if injectionPattern.MatchString(v) {
			return "", errors.New("argument contains illegal characters")
		}
		cmd = strings.ReplaceAll(cmd, "{{"+k+"}}", v)
	}

	return cmd, nil
}
