import re
with open('d:/claudeprj/deploy/godeployer/api.go', 'r', encoding='utf-8') as f:
    c = f.read()

m = re.search(r'func (git\.)?isCommitHash[^\n]*\n.*?\n}', c, re.DOTALL)
if m:
    print("Found isCommitHash in api.go!")
    func_text = m.group(0).replace('func git.isCommitHash', 'func IsCommitHash').replace('func isCommitHash', 'func IsCommitHash')
    c = c.replace(m.group(0), '')
    with open('d:/claudeprj/deploy/godeployer/api.go', 'w', encoding='utf-8') as f:
        f.write(c)
    with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'a', encoding='utf-8') as f:
        f.write('\n\n' + func_text + '\n')
else:
    print("Could not find isCommitHash in api.go")
