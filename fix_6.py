import os

# fix engine_test.go
f_eng = 'd:/claudeprj/deploy/godeployer/application/engine_test.go'
if os.path.exists(f_eng):
    with open(f_eng, 'r', encoding='utf-8') as f:
        c = f.read()
    c = c.replace('package application_test', 'package application')
    c = c.replace('application.', '')
    with open(f_eng, 'w', encoding='utf-8') as f:
        f.write(c)

# fix auth_test.go
f_auth = 'd:/claudeprj/deploy/godeployer/application/auth_test.go'
if os.path.exists(f_auth):
    with open(f_auth, 'r', encoding='utf-8') as f:
        c = f.read()
    c = c.replace('package application_test', 'package application')
    c = c.replace('application.', '')
    with open(f_auth, 'w', encoding='utf-8') as f:
        f.write(c)

