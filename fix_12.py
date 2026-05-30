import os
import re

f = 'd:/claudeprj/deploy/godeployer/interfaces/api/api_test.go'
with open(f, 'r', encoding='utf-8') as file:
    content = file.read()

# I appended the mock code at the end of the file. I can find 'type MockRemoteExecutor struct' and truncate the file there.
idx = content.find('\ntype MockRemoteExecutor struct')
if idx != -1:
    content = content[:idx]

mock = '''
import "sync"
import "strings"

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
    m.mu.Lock()
    defer m.mu.Unlock()
    return nil
}
'''
# Actually we can just fix the undefined symbols in the previous appending:
# NewDeployEngine -> pplication.NewDeployEngine, etc.
# Wait, if I just truncate it and add a simple mock, that's better! But wait, the simple mock above doesn't have context or untime issues!
