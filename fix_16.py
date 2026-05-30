import os
import re

f = 'd:/claudeprj/deploy/godeployer/interfaces/api/api_test.go'
with open(f, 'r', encoding='utf-8') as file:
    content = file.read()

content = re.sub(r'func \(m \*MockRemoteExecutor\) Rsync\(.*?\)', 'func (m *MockRemoteExecutor) Rsync(a, b, c string)', content)

with open(f, 'w', encoding='utf-8') as file:
    file.write(content)
