import glob, re

for fpath in glob.glob('d:/claudeprj/deploy/godeployer/*_test.go'):
    with open(fpath, 'r', encoding='utf-8') as f:
        c = f.read()
    
    # Fix godeployer.domain. etc.
    c = c.replace('godeployer.domain.', 'domain.')
    c = c.replace('godeployer.application.', 'application.')
    c = c.replace('godeployer.ssh.', 'ssh.')
    c = c.replace('godeployer.git.', 'git.')
    c = c.replace('godeployer.sys.', 'sys.')
    
    # Also add imports if needed for test files
    imports = [
        '"deploy/godeployer/domain"',
        '"deploy/godeployer/application"',
        '"deploy/godeployer/infrastructure/ssh"',
        '"deploy/godeployer/infrastructure/git"'
    ]
    for imp in imports:
        if imp not in c and (imp.split('/')[-1].strip('"') + '.' in c):
            c = c.replace('import (', f'import (\n\t{imp}')

    with open(fpath, 'w', encoding='utf-8') as f:
        f.write(c)

# also fix application/deploy_service.go undefined: setProcessGroup
with open('d:/claudeprj/deploy/godeployer/application/deploy_service.go', 'r', encoding='utf-8') as f:
    c = f.read()
    c = c.replace('killProcessGroup(', 'sys.KillProcessGroup(')
with open('d:/claudeprj/deploy/godeployer/application/deploy_service.go', 'w', encoding='utf-8') as f:
    f.write(c)

# also fix sys exports
for f in ['sys_unix.go', 'sys_windows.go']:
    fpath = f'd:/claudeprj/deploy/godeployer/infrastructure/sys/{f}'
    with open(fpath, 'r', encoding='utf-8') as file:
        c = file.read()
        c = c.replace('func killProcessGroup', 'func KillProcessGroup')
    with open(fpath, 'w', encoding='utf-8') as file:
        file.write(c)

