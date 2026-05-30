import os
import glob
import re

test_files = glob.glob('d:/claudeprj/deploy/godeployer/interfaces/api/*_test.go')

for f in test_files:
    with open(f, 'r', encoding='utf-8') as file:
        c = file.read()
    
    # Replace godeployer prefixes
    c = c.replace('godeployer.sqlite', 'sqlite')
    c = c.replace('godeployer.domain', 'domain')
    c = c.replace('godeployer.application', 'application')
    c = c.replace('godeployer.SetupRoutes', 'SetupRoutes')
    c = c.replace('godeployer.SetupRoutesWithExecutor', 'SetupRoutesWithExecutor')
    c = c.replace('godeployer.ComputeGithubSignature', 'ComputeGithubSignature')
    
    with open(f, 'w', encoding='utf-8') as file:
        file.write(c)

