import os

f = 'd:/claudeprj/deploy/godeployer/interfaces/api/api_test.go'
with open(f, 'r', encoding='utf-8') as file:
    content = file.read()

# Replace the previous mock implementation with a complete one.
# It was added at the end of the file:
idx = content.find('type MockRemoteExecutor struct {')
if idx != -1:
    content = content[:idx]

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
func (m *MockRemoteExecutor) Close() error {
    return nil
}
func (m *MockRemoteExecutor) GetCommands() []string {
    m.mu.Lock()
    defer m.mu.Unlock()
    res := make([]string, len(m.commandsRun))
    copy(res, m.commandsRun)
    return res
}
'''
with open(f, 'w', encoding='utf-8') as file:
    file.write(content)
