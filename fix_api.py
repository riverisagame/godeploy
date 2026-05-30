import glob, re

for f in glob.glob('d:/claudeprj/deploy/godeployer/interfaces/api/*.go'):
    with open(f, 'r', encoding='utf-8') as file:
        c = file.read()
    c = c.replace('package godeployer', 'package api')
    
    # Needs to import domain, application, infrastructure etc if they are used
    # We will just let goimports handle imports if possible, but we don't have it easily scriptable here.
    # We can try to rely on go fmt / govet, or we can just replace known types:
    
    with open(f, 'w', encoding='utf-8') as file:
        file.write(c)

# Fix ssh_test.go
with open('d:/claudeprj/deploy/godeployer/infrastructure/ssh/ssh_test.go', 'r', encoding='utf-8') as f:
    c = f.read()
    c = c.replace('ssh.domain', 'domain')
with open('d:/claudeprj/deploy/godeployer/infrastructure/ssh/ssh_test.go', 'w', encoding='utf-8') as f:
    f.write(c)
