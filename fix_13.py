import os

f = 'd:/claudeprj/deploy/godeployer/interfaces/api/api_test.go'
with open(f, 'r', encoding='utf-8') as file:
    content = file.read()

idx = content.find('\ntype MockRemoteExecutor struct')
if idx != -1:
    content = content[:idx]

mock = '''
import "sync"

type MockRemoteExecutor struct {
    mu sync.Mutex
    commandsRun []string
}
func (m *MockRemoteExecutor) RunCommand(cmd string) (string, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.commandsRun = append(m.commandsRun, cmd)
    return "", nil
}
func (m *MockRemoteExecutor) RunRsync(src string, user string, host string, port int, dest string, exclude []string) error {
    return nil
}
'''

# Wait, adding `import "sync"` in the middle of a go file is illegal.
# So I should just append the struct and make sure "sync" is imported at the top!
# I will use a regex to inject "sync" into imports if not there.

if "sync" not in content:
    content = content.replace('import (', 'import (\n\t"sync"\n', 1)

content += '''
type MockRemoteExecutor struct {
    mu sync.Mutex
    commandsRun []string
}
func (m *MockRemoteExecutor) RunCommand(cmd string) (string, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.commandsRun = append(m.commandsRun, cmd)
    return "", nil
}
func (m *MockRemoteExecutor) RunRsync(src string, user string, host string, port int, dest string, exclude []string) error {
    return nil
}
'''
with open(f, 'w', encoding='utf-8') as file:
    file.write(content)
