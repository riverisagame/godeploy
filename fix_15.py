import os

f = 'd:/claudeprj/deploy/godeployer/interfaces/api/api_test.go'
with open(f, 'r', encoding='utf-8') as file:
    content = file.read()

content = content.replace('func (m *MockRemoteExecutor) RunRsync', 'func (m *MockRemoteExecutor) Rsync')

with open(f, 'w', encoding='utf-8') as file:
    file.write(content)
