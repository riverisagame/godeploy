import re, glob

# Fix getCacheDir
for f in ['d:/claudeprj/deploy/godeployer/infrastructure/git/git_cache_test.go', 'd:/claudeprj/deploy/godeployer/git_ref_test.go']:
    with open(f, 'r', encoding='utf-8') as file:
        c = file.read()
        c = c.replace('getCacheDir(', 'GetCacheDir(')
        c = c.replace('GetCommits(', 'git.GetCommits(')
    with open(f, 'w', encoding='utf-8') as file:
        file.write(c)

# Fix GlobalConfig, NewDeployEngine
for f in glob.glob('d:/claudeprj/deploy/godeployer/*_test.go'):
    with open(f, 'r', encoding='utf-8') as file:
        c = file.read()
    c = c.replace('GlobalConfig{', 'domain.GlobalConfig{')
    c = c.replace('NewDeployEngine(', 'application.NewDeployEngine(')
    with open(f, 'w', encoding='utf-8') as file:
        file.write(c)

# Fix ssh_test.go variable redeclaration
with open('d:/claudeprj/deploy/godeployer/ssh_test.go', 'r', encoding='utf-8') as f:
    c = f.read()
    c = c.replace('ssh := ', 'sshClient := ') # assuming it was ssh := something
    c = c.replace('ssh.NewSignerFromKey(', 'ssh.NewSignerFromKey(') # Wait, it said "undefined: ssh.NewSignerFromKey" which might be golang.org/x/crypto/ssh
    # If the imported package is golang.org/x/crypto/ssh but we also imported deploy/godeployer/infrastructure/ssh, they clash!
    c = c.replace('"deploy/godeployer/infrastructure/ssh"', 'godeployerssh "deploy/godeployer/infrastructure/ssh"')
    c = c.replace('ssh.SSHPool', 'godeployerssh.SSHPool')
    c = c.replace('ssh.NewSSHPool', 'godeployerssh.NewSSHPool')
    c = c.replace('ssh.RemoteExecutor', 'godeployerssh.RemoteExecutor')
    c = c.replace('ssh.SSHExecutor', 'godeployerssh.SSHExecutor')
    c = c.replace('ssh.NewSSHExecutor', 'godeployerssh.NewSSHExecutor')
with open('d:/claudeprj/deploy/godeployer/ssh_test.go', 'w', encoding='utf-8') as f:
    f.write(c)
