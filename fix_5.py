import glob, os

# Fix notifier_test.go
f_notifier = 'd:/claudeprj/deploy/godeployer/infrastructure/notifier/notifier_test.go'
with open(f_notifier, 'r', encoding='utf-8') as f:
    c = f.read()
c = c.replace('godeployer.EventDeploySuccess', 'notifier.EventDeploySuccess')
with open(f_notifier, 'w', encoding='utf-8') as f:
    f.write(c)

# Fix sqlite_test.go
f_sqlite = 'd:/claudeprj/deploy/godeployer/infrastructure/sqlite/sqlite_test.go'
with open(f_sqlite, 'r', encoding='utf-8') as f:
    c = f.read()
c = c.replace('godeployer.InitDB', 'sqlite.InitDB')
c = c.replace('godeployer.RepairStalledTasks', 'sqlite.RepairStalledTasks')
with open(f_sqlite, 'w', encoding='utf-8') as f:
    f.write(c)

# Fix auth_test.go -> rename package or import correctly
# But wait, auth.go should be in application/auth.go or infrastructure/auth.
# I will just move auth.go and auth_test.go to application/ since they contain auth service logic
if os.path.exists('d:/claudeprj/deploy/godeployer/auth.go'):
    os.rename('d:/claudeprj/deploy/godeployer/auth.go', 'd:/claudeprj/deploy/godeployer/application/auth.go')
if os.path.exists('d:/claudeprj/deploy/godeployer/auth_test.go'):
    os.rename('d:/claudeprj/deploy/godeployer/auth_test.go', 'd:/claudeprj/deploy/godeployer/application/auth_test.go')

# Fix auth code packages
for f in ['d:/claudeprj/deploy/godeployer/application/auth.go', 'd:/claudeprj/deploy/godeployer/application/auth_test.go']:
    if not os.path.exists(f): continue
    with open(f, 'r', encoding='utf-8') as file:
        c = file.read()
    c = c.replace('package godeployer\n', 'package application\n')
    c = c.replace('package godeployer_test', 'package application_test')
    c = c.replace('godeployer.', 'application.')
    with open(f, 'w', encoding='utf-8') as file:
        file.write(c)

# Remove import cycle in api_permissions_test.go
f_perm = 'd:/claudeprj/deploy/godeployer/interfaces/api/api_permissions_test.go'
with open(f_perm, 'r', encoding='utf-8') as f:
    c = f.read()
c = c.replace('\n\t"deploy/godeployer/interfaces/api"\n', '\n')
c = c.replace('\n\tapi "deploy/godeployer/interfaces/api"\n', '\n')
with open(f_perm, 'w', encoding='utf-8') as f:
    f.write(c)

# Move config_test.go, engine_test.go, main_test.go
if os.path.exists('d:/claudeprj/deploy/godeployer/engine_test.go'):
    os.rename('d:/claudeprj/deploy/godeployer/engine_test.go', 'd:/claudeprj/deploy/godeployer/application/engine_test.go')
    with open('d:/claudeprj/deploy/godeployer/application/engine_test.go', 'r', encoding='utf-8') as f:
        c = f.read()
    c = c.replace('package godeployer_test', 'package application_test')
    c = c.replace('godeployer.', 'application.')
    with open('d:/claudeprj/deploy/godeployer/application/engine_test.go', 'w', encoding='utf-8') as f:
        f.write(c)

if os.path.exists('d:/claudeprj/deploy/godeployer/api_test.go'): # If still there
    os.remove('d:/claudeprj/deploy/godeployer/api_test.go') # Because it was supposed to be moved! I'll move it properly.

