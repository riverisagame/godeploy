import os
import glob

test_files = glob.glob('d:/claudeprj/deploy/godeployer/interfaces/api/*_test.go')

for f in test_files:
    with open(f, 'r', encoding='utf-8') as file:
        c = file.read()
    
    # Change package to api
    c = c.replace('package api_test', 'package api')
    
    with open(f, 'w', encoding='utf-8') as file:
        file.write(c)

