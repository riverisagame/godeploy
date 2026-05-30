import re

with open('d:/claudeprj/deploy/godeployer/api.go', 'r', encoding='utf-8') as f:
    c = f.read()

# FilterFilesForTruncatedDiff
m2 = re.search(r'// git\.FilterFilesForTruncatedDiff.*?return strings\.Join\(filtered, "\\n"\)\n}', c, re.DOTALL)
if m2:
    fn2 = m2.group(0).replace('func filterFilesForTruncatedDiff', 'func FilterFilesForTruncatedDiff')
    c = c.replace(m2.group(0), '')
    with open('d:/claudeprj/deploy/godeployer/api.go', 'w', encoding='utf-8') as f:
        f.write(c)
    with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'a', encoding='utf-8') as f:
        f.write('\n\n' + fn2 + '\n')
else:
    print("Could not find filterFilesForTruncatedDiff again")

with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'r', encoding='utf-8') as f:
    c = f.read()
    if '"time"' not in c:
        c = c.replace('import (', 'import (\n\t"time"')
with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'w', encoding='utf-8') as f:
    f.write(c)

