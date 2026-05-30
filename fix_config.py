import re

with open('d:/claudeprj/deploy/godeployer/config.go', 'r', encoding='utf-8') as f:
    c = f.read()

# Remove struct definitions
c = re.sub(r'type Config struct.*?yaml:"-"\n}', '', c, flags=re.DOTALL)
c = re.sub(r'type GlobalConfig struct.*?\n}', '', c, flags=re.DOTALL)
c = re.sub(r'type ProjectConfig struct.*?\n}', '', c, flags=re.DOTALL)
c = re.sub(r'type BuildConfig struct.*?\n}', '', c, flags=re.DOTALL)
c = re.sub(r'type EnvironmentConfig struct.*?\n}', '', c, flags=re.DOTALL)
c = re.sub(r'type ServerConfig struct.*?\n}', '', c, flags=re.DOTALL)

# Now config.go uses domain.*
c = c.replace('func LoadConfig', 'func LoadConfig')
c = c.replace('*Config', '*domain.Config')
c = c.replace('ProjectConfig', 'domain.ProjectConfig')
c = c.replace('GlobalConfig', 'domain.GlobalConfig')

if '"deploy/godeployer/domain"' not in c:
    c = c.replace('import (', 'import (\n\t"deploy/godeployer/domain"\n')

with open('d:/claudeprj/deploy/godeployer/config.go', 'w', encoding='utf-8') as f:
    f.write(c)
