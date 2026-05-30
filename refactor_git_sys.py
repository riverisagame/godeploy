import os, glob

# Create dirs
os.makedirs('d:/claudeprj/deploy/godeployer/infrastructure/git', exist_ok=True)
os.makedirs('d:/claudeprj/deploy/godeployer/infrastructure/sys', exist_ok=True)

# Move git files
for f in ['git.go', 'git_cache_test.go']:
    src = f'd:/claudeprj/deploy/godeployer/{f}'
    dst = f'd:/claudeprj/deploy/godeployer/infrastructure/git/{f}'
    if os.path.exists(src):
        os.rename(src, dst)
        with open(dst, 'r', encoding='utf-8') as file:
            c = file.read()
        c = c.replace('package godeployer', 'package git')
        # Capitalize exports
        c = c.replace('func getCacheDir', 'func GetCacheDir')
        c = c.replace('func findGitRepo', 'func FindGitRepo')
        c = c.replace('func filterFilesForTruncatedDiff', 'func FilterFilesForTruncatedDiff')
        with open(dst, 'w', encoding='utf-8') as file:
            file.write(c)

# Move sys files
sys_files = ['disk_linux.go', 'disk_windows.go', 'sys_unix.go', 'sys_windows.go']
for f in sys_files:
    src = f'd:/claudeprj/deploy/godeployer/{f}'
    dst = f'd:/claudeprj/deploy/godeployer/infrastructure/sys/{f}'
    if os.path.exists(src):
        os.rename(src, dst)
        with open(dst, 'r', encoding='utf-8') as file:
            c = file.read()
        c = c.replace('package godeployer', 'package sys')
        c = c.replace('func getFreeDiskSpaceMB', 'func GetFreeDiskSpaceMB')
        c = c.replace('func setProcessGroup', 'func SetProcessGroup')
        with open(dst, 'w', encoding='utf-8') as file:
            file.write(c)
