with open('d:/claudeprj/deploy/godeployer/interfaces/api/api.go', 'r', encoding='utf-8') as f:
    c = f.read()

c = c.replace('func api.SetupRoutes(', 'func SetupRoutes(')
c = c.replace('func api.SetupRoutesWithExecutor(', 'func SetupRoutesWithExecutor(')
c = c.replace('return api.SetupRoutesWithExecutor(', 'return SetupRoutesWithExecutor(')

if '"deploy/godeployer/application"' not in c:
    c = c.replace('import (', 'import (\n\t"deploy/godeployer/application"\n\t"deploy/godeployer/infrastructure/ssh"\n')

with open('d:/claudeprj/deploy/godeployer/interfaces/api/api.go', 'w', encoding='utf-8') as f:
    f.write(c)

