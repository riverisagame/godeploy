with open('d:/claudeprj/deploy/godeployer/infrastructure/ssh/ssh_test.go', 'r', encoding='utf-8') as f:
    c = f.read()

c = c.replace('domain.ServerConfig{', 'ssh.ServerConfig{')
c = c.replace('*domain.ServerConfig', '*ssh.ServerConfig')

with open('d:/claudeprj/deploy/godeployer/infrastructure/ssh/ssh_test.go', 'w', encoding='utf-8') as f:
    f.write(c)

