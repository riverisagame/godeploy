import os
import re

with open('d:/claudeprj/deploy/godeployer/application/engine_test.go', 'r', encoding='utf-8') as f:
    engine_content = f.read()

# Extract MockRemoteExecutor
start = engine_content.find('type MockRemoteExecutor struct')
end = engine_content.find('func TestDeployEngine_MultiNodeDeploy')

mock_code = engine_content[start:end]

with open('d:/claudeprj/deploy/godeployer/interfaces/api/api_test.go', 'a', encoding='utf-8') as f:
    f.write('\n' + mock_code)

