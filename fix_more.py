import glob, re

# Fix notifier_test.go
with open('d:/claudeprj/deploy/godeployer/infrastructure/notifier/notifier_test.go', 'r', encoding='utf-8') as f:
    c = f.read()
c = c.replace('package godeployer_test', 'package notifier_test')
c = c.replace('godeployer.DeployEvent', 'notifier.DeployEvent')
c = c.replace('godeployer.NewEventBus', 'notifier.NewEventBus')
with open('d:/claudeprj/deploy/godeployer/infrastructure/notifier/notifier_test.go', 'w', encoding='utf-8') as f:
    f.write(c)

# Fix ssh_local_bypass_test.go
with open('d:/claudeprj/deploy/godeployer/infrastructure/ssh/ssh_local_bypass_test.go', 'r', encoding='utf-8') as f:
    c = f.read()
c = c.replace('package godeployer_test', 'package ssh_test')
c = c.replace('godeployer.ServerConfig', 'domain.ServerConfig')
c = c.replace('godeployer.NewSSHExecutor', 'ssh.NewSSHExecutor')
c = c.replace('godeployer.NewSSHPool', 'ssh.NewSSHPool')
with open('d:/claudeprj/deploy/godeployer/infrastructure/ssh/ssh_local_bypass_test.go', 'w', encoding='utf-8') as f:
    f.write(c)

# Fix sqlite package
for f in ['d:/claudeprj/deploy/godeployer/infrastructure/sqlite/sqlite.go', 'd:/claudeprj/deploy/godeployer/infrastructure/sqlite/sqlite_test.go']:
    with open(f, 'r', encoding='utf-8') as file:
        c = file.read()
    c = c.replace('package godeployer', 'package sqlite')
    if '"deploy/godeployer/domain"' not in c:
        c = c.replace('import (', 'import (\n\t"deploy/godeployer/domain"\n')
    with open(f, 'w', encoding='utf-8') as file:
        file.write(c)

# Clean up interfaces/api imports and InitDB
for f in glob.glob('d:/claudeprj/deploy/godeployer/interfaces/api/*.go'):
    with open(f, 'r', encoding='utf-8') as file:
        c = file.read()
    
    # InitDB -> sqlite.InitDB
    c = c.replace('InitDB(', 'sqlite.InitDB(')
    c = c.replace('SetupRoutes(', 'api.SetupRoutes(')
    
    # Remove unused imports naively if possible
    c = re.sub(r'\"deploy/godeployer/infrastructure/sys\"\n', '', c)
    c = re.sub(r'\"deploy/godeployer/infrastructure/ssh\"\n', '', c)
    c = re.sub(r'\"deploy/godeployer/infrastructure/git\"\n', '', c)
    c = re.sub(r'\"deploy/godeployer/application\"\n', '', c)
    
    # we might actually need domain, sqlite, application in api.go, so we won't remove them from all files blindly
    # let's just rely on goimports or gofmt.
    # Oh wait, go fmt doesn't remove unused imports. We need goimports. 
    # But since I can't be sure goimports is installed, I'll use regular expressions to clean up.
    
    # Instead of deleting lines, let's just run go test and see. Wait, "imported and not used" is a hard error in Go tests.
    # I should just delete the blocks I just regex'd.
    
    with open(f, 'w', encoding='utf-8') as file:
        file.write(c)

