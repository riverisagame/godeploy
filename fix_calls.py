import os, glob

def fix_file(file_path):
    with open(file_path, 'r', encoding='utf-8') as file:
        c = file.read()

    # Add imports
    if 'deploy/godeployer/infrastructure/git' not in c:
        c = c.replace('import (', 'import (\n\t"deploy/godeployer/infrastructure/git"\n\t"deploy/godeployer/infrastructure/sys"')
    
    replacements = {
        'EnsureRepoCache': 'git.EnsureRepoCache',
        'getCacheDir': 'git.GetCacheDir',
        'findGitRepo': 'git.FindGitRepo',
        'filterFilesForTruncatedDiff': 'git.FilterFilesForTruncatedDiff',
        'getFreeDiskSpaceMB': 'sys.GetFreeDiskSpaceMB',
        'setProcessGroup': 'sys.SetProcessGroup',
        'GetCommits': 'git.GetCommits',
        'GetDiffForFile': 'git.GetDiffForFile',
        'isCommitHash': 'git.IsCommitHash',
        'GetCommitAuthor': 'git.GetCommitAuthor',
        'GitCommit': 'git.GitCommit'
    }
    
    for k, v in replacements.items():
        # A simple replacement, being careful about double prefixes
        c = c.replace(k, v)
    
    c = c.replace('git.git.', 'git.')
    c = c.replace('sys.sys.', 'sys.')
    c = c.replace('sys.git.', 'sys.') # Just in case

    with open(file_path, 'w', encoding='utf-8') as file:
        file.write(c)

fix_file('d:/claudeprj/deploy/godeployer/application/deploy_service.go')
for f in glob.glob('d:/claudeprj/deploy/godeployer/api*.go'):
    fix_file(f)

# Also fix git.go to make isCommitHash exported
with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'r', encoding='utf-8') as file:
    c = file.read()
    c = c.replace('func isCommitHash', 'func IsCommitHash')
with open('d:/claudeprj/deploy/godeployer/infrastructure/git/git.go', 'w', encoding='utf-8') as file:
    file.write(c)

