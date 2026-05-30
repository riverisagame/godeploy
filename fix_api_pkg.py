import glob

for f in glob.glob('d:/claudeprj/deploy/godeployer/interfaces/api/*.go'):
    with open(f, 'r', encoding='utf-8') as file:
        c = file.read()
    c = c.replace('package godeployer\n', 'package api\n')
    c = c.replace('package godeployer\r\n', 'package api\r\n')
    with open(f, 'w', encoding='utf-8') as file:
        file.write(c)

