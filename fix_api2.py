import sys, glob, re

def fix(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        c = f.read()
    
    # Fix the literal \n\t that got inserted
    c = c.replace(r'\n\t', '\n\t')

    with open(file_path, 'w', encoding='utf-8') as f:
        f.write(c)

for f in glob.glob('d:/claudeprj/deploy/godeployer/api*.go'):
    fix(f)
