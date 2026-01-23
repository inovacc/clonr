clonr https,http,git,ssh,ftp,sftp > repo to clone
clonr list                        > list all repos
clonr rm                          > remove repo
clonr add                         > add local repo
clonr configure                   > configure monitor interval, port, default repo main directory, editor
clonr map                         > map local directory to search for repos
clonr server                      > start server to expose api
clonr nerds                       > show nerd stats of all repos
clonr status                      > show status of all repos
clonr open                        > list favorited repos to open
clonr help                        > show help

## GitHub CLI Integration (clonr gh)

clonr gh issues list              > list issues for a repository
clonr gh issues create            > create a new issue
clonr gh pr status                > check pull request status
clonr gh pr status 123            > detailed status of specific PR
clonr gh actions status           > list recent workflow runs
clonr gh actions status 123456    > detailed status of specific run
clonr gh release list             > list releases
clonr gh release create           > create a new release
clonr gh release download         > download release assets

All gh commands:
- Auto-detect repository from current directory
- Support --repo owner/repo flag for explicit repo
- Support --json flag for JSON output
- Support --token flag for explicit token (auto-detects from gh CLI config)

## UI Behavior

list, rm use menu selection

rm trigger list of repos to remove

list trigger list of repos to do more things like nerds, status, rm, info, open etc

when open is selected it opens the repo asking for the editor saved in configure
