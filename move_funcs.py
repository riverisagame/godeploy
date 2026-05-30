import re

with open('d:/claudeprj/deploy/godeployer/api.go', 'r', encoding='utf-8') as f:
    c = f.read()

# Find findGitRepo
m1 = re.search(r'// git\.FindGitRepo[^\n]*\nfunc findGitRepo.*?return result, err\n}', c, re.DOTALL)
if m1:
    fn1 = m1.group(0).replace('func findGitRepo', 'func FindGitRepo')
    c = c.replace(m1.group(0), '')
else:
    print("Could not find findGitRepo")
    fn1 = ""

m2 = re.search(r'// git\.FilterFilesForTruncatedDiff[^\n]*\n//[^\n]*\nfunc filterFilesForTruncatedDiff.*?return strings\.Join\(filtered, "\\n"\)\n}', c, re.DOTALL)
if m2:
    fn2 = m2.group(0).replace('func filterFilesForTruncatedDiff', 'func FilterFilesForTruncatedDiff')
    c = c.replace(m2.group(0), '')
else:
    print("Could not find filterFilesForTruncatedDiff")
    fn2 = ""

with open('d:/claudeprj/deploy/godeployer/api.go', 'w', encoding='utf-8') as f:
    f.write(c)

with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'a', encoding='utf-8') as f:
    f.write('\n\n' + fn1 + '\n\n' + fn2 + '\n')

