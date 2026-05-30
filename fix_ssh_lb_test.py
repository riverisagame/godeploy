with open('d:/claudeprj/deploy/godeployer/infrastructure/ssh/ssh_local_bypass_test.go', 'r', encoding='utf-8') as f:
    c = f.read()
c = c.replace('package godeployer', 'package ssh')
with open('d:/claudeprj/deploy/godeployer/infrastructure/ssh/ssh_local_bypass_test.go', 'w', encoding='utf-8') as f:
    f.write(c)
