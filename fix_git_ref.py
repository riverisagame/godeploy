with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git_ref_test.go', 'r', encoding='utf-8') as f:
    c = f.read()
c = c.replace('package godeployer', 'package git')
c = c.replace('git.GetCommits(', 'GetCommits(')
with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git_ref_test.go', 'w', encoding='utf-8') as f:
    f.write(c)
