import os
import re

f = 'd:/claudeprj/deploy/godeployer/interfaces/api/api_enhance_test.go'
with open(f, 'r', encoding='utf-8') as file:
    content = file.read()

# Instead of doing a full tempdir setup, since this test is only testing API routing and mocking git response,
# wait, it actually hits `engine.RunDeploy` or `git.UpdateRepoCache` which does real git clone!
# So I should create a real temp repo.

# Find `TestAPI_PreviewDiffWithFileList`
# It has:
# 	wd, _ := os.Getwd()
#	mockConfig := &domain.Config{ ... Repo: wd ... }

# Replace wd with a temp dir
replacement = """
	repoDir := t.TempDir()
	_ = exec.Command("git", "init", repoDir).Run()
	func() {
		cmd := exec.Command("git", "config", "user.email", "test@test.com")
		cmd.Dir = repoDir
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = repoDir
		_ = cmd.Run()
		cmd = exec.Command("git", "commit", "--allow-empty", "-m", "init")
		cmd.Dir = repoDir
		_ = cmd.Run()
	}()
"""

content = content.replace('wd, _ := os.Getwd()', replacement)
content = content.replace('Repo:          wd,', 'Repo:          repoDir,')

with open(f, 'w', encoding='utf-8') as file:
    file.write(content)
