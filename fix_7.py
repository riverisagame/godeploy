import os
import re

# fix api_permissions_test.go
f_api = 'd:/claudeprj/deploy/godeployer/interfaces/api/api_permissions_test.go'
if os.path.exists(f_api):
    with open(f_api, 'r', encoding='utf-8') as f:
        c = f.read()
    c = c.replace('	"deploy/godeployer/interfaces/api"\n', '')
    c = c.replace('api.SetupRoutes', 'SetupRoutes')
    with open(f_api, 'w', encoding='utf-8') as f:
        f.write(c)

# fix engine_test.go
f_eng = 'd:/claudeprj/deploy/godeployer/application/engine_test.go'
if os.path.exists(f_eng):
    with open(f_eng, 'r', encoding='utf-8') as f:
        c = f.read()
    
    replacements = {
        r'\bProjectConfig\b': 'domain.ProjectConfig',
        r'\bBuildConfig\b': 'domain.BuildConfig',
        r'\bServerConfig\b': 'domain.ServerConfig',
        r'\bEnvironmentConfig\b': 'domain.EnvironmentConfig',
        r'\bConfig\b': 'domain.Config',
        r'\bDeployJob\b': 'domain.DeployJob',
        r'\bInitDB\b': 'sqlite.InitDB'
    }
    
    for k, v in replacements.items():
        # don't replace if already domain.XXX
        c = re.sub(r'(?<!domain\.)' + k, v, c)
        
    with open(f_eng, 'w', encoding='utf-8') as f:
        f.write(c)

