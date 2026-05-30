with open('d:/claudeprj/deploy/godeployer/main.go', 'r', encoding='utf-8') as f:
    c = f.read()

c = c.replace('SetupRoutes(', 'api.SetupRoutes(')
if '"deploy/godeployer/interfaces/api"' not in c:
    c = c.replace('"deploy/godeployer/infrastructure/notifier"\n', '"deploy/godeployer/infrastructure/notifier"\n\t"deploy/godeployer/interfaces/api"\n')

with open('d:/claudeprj/deploy/godeployer/main.go', 'w', encoding='utf-8') as f:
    f.write(c)

