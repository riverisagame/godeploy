with open('d:/claudeprj/deploy/godeployer/infrastructure/ssh/ssh_test.go', 'r', encoding='utf-8') as f:
    c = f.read()
c = c.replace('package godeployer', 'package ssh')
c = c.replace('godeployerssh "deploy/godeployer/infrastructure/ssh"\n', '')
c = c.replace('godeployerssh.', '')
c = c.replace('domain.ServerConfig', 'domain.ServerConfig') # Keep it if it used domain.ServerConfig
# But wait, it previously said "undefined: ServerConfig". It needs to import domain and use domain.ServerConfig.
if '"deploy/godeployer/domain"' not in c:
    c = c.replace('import (', 'import (\n\t"deploy/godeployer/domain"\n')
c = c.replace('ServerConfig{', 'domain.ServerConfig{')
c = c.replace('domain.domain.ServerConfig', 'domain.ServerConfig')
with open('d:/claudeprj/deploy/godeployer/infrastructure/ssh/ssh_test.go', 'w', encoding='utf-8') as f:
    f.write(c)
