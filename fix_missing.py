import os

with open('d:/claudeprj/deploy/godeployer/application/deploy_service.go', 'r', encoding='utf-8') as f:
    c = f.read()

c = c.replace('go func(srv ServerConfig)', 'go func(srv domain.ServerConfig)')
c = c.replace('executor.(*SSHExecutor)', 'executor.(*ssh.SSHExecutor)')

with open('d:/claudeprj/deploy/godeployer/application/deploy_service.go', 'w', encoding='utf-8') as f:
    f.write(c)

# What about FindGitRepo?
with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'r', encoding='utf-8') as f:
    c = f.read()
    c = c.replace('func findGitRepo', 'func FindGitRepo')
    c = c.replace('func filterFilesForTruncatedDiff', 'func FilterFilesForTruncatedDiff')
with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'w', encoding='utf-8') as f:
    f.write(c)

