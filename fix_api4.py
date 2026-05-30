import re

with open('d:/claudeprj/deploy/godeployer/api.go', 'r', encoding='utf-8') as f:
    c = f.read()

# Fix application prefixes
replacements = {
    'HashPassword(': 'application.HashPassword(',
    'ParseToken(': 'application.ParseToken(',
    'ErrQueueFull': 'application.ErrQueueFull',
    'NewDeployEngine(': 'application.NewDeployEngine(',
    'domain.Config:': 'Config:',
    'git.IsCommitHash': 'git.IsCommitHash', # Wait, why was IsCommitHash undefined? Because it's not exported in git.go? Let me check
}

for k, v in replacements.items():
    c = c.replace(k, v)

# Fix double application just in case
c = c.replace('application.application.', 'application.')

with open('d:/claudeprj/deploy/godeployer/api.go', 'w', encoding='utf-8') as f:
    f.write(c)

# Check IsCommitHash in git.go
with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'r', encoding='utf-8') as f:
    git_c = f.read()
    git_c = git_c.replace('func isCommitHash', 'func IsCommitHash')
with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'w', encoding='utf-8') as f:
    f.write(git_c)
