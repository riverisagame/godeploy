import os
import re

f = 'd:/claudeprj/deploy/godeployer/interfaces/api/api_enhance_test.go'
with open(f, 'r', encoding='utf-8') as file:
    content = file.read()

content = content.replace('"os"', '"os"\n\t"os/exec"')

with open(f, 'w', encoding='utf-8') as file:
    file.write(content)
