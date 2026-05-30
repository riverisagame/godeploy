with open('d:/claudeprj/deploy/godeployer/interfaces/api/api.go', 'r', encoding='utf-8') as f:
    c = f.read()

c = c.replace('func api.SetupRoutes(', 'func SetupRoutes(')
c = c.replace('func api.SetupRoutesWithExecutor(', 'func SetupRoutesWithExecutor(')
c = c.replace('return api.SetupRoutesWithExecutor(', 'return SetupRoutesWithExecutor(')

with open('d:/claudeprj/deploy/godeployer/interfaces/api/api.go', 'w', encoding='utf-8') as f:
    f.write(c)

with open('d:/claudeprj/deploy/godeployer/infrastructure/sqlite/sqlite.go', 'r', encoding='utf-8') as f:
    c = f.read()
c = c.replace('"deploy/godeployer/domain"\n', '')
with open('d:/claudeprj/deploy/godeployer/infrastructure/sqlite/sqlite.go', 'w', encoding='utf-8') as f:
    f.write(c)
