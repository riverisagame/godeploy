import re

with open('d:/claudeprj/deploy/godeployer/api.go', 'r', encoding='utf-8') as f:
    lines = f.readlines()

out_lines = []
fn_lines = []
in_fn = False

for line in lines:
    if 'func filterFilesForTruncatedDiff' in line or 'git.FilterFilesForTruncatedDiff' in line:
        in_fn = True
    
    if in_fn:
        fn_lines.append(line)
        if line.startswith('}'):
            in_fn = False
    else:
        out_lines.append(line)

with open('d:/claudeprj/deploy/godeployer/api.go', 'w', encoding='utf-8') as f:
    f.writelines(out_lines)

fn_code = ''.join(fn_lines).replace('func filterFilesForTruncatedDiff', 'func FilterFilesForTruncatedDiff')

with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'a', encoding='utf-8') as f:
    f.write('\n\n' + fn_code + '\n')

