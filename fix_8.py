import os
import glob
import re

test_files = glob.glob('d:/claudeprj/deploy/godeployer/interfaces/api/*_test.go')

for f in test_files:
    with open(f, 'r', encoding='utf-8') as file:
        c = file.read()
    
    # Remove import
    c = re.sub(r'\n\s*"deploy/godeployer/interfaces/api"\n', '\n', c)
    c = re.sub(r'\n\s*api\s+"deploy/godeployer/interfaces/api"\n', '\n', c)
    
    # Remove api. prefix for known symbols
    c = c.replace('api.SetupRoutes', 'SetupRoutes')
    c = c.replace('api.APIHandler', 'APIHandler')
    c = c.replace('api.RequireAuth', 'RequireAuth')
    
    with open(f, 'w', encoding='utf-8') as file:
        file.write(c)

