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

list, rm use menu selection

rm trigger list of repos to remove

list trigger list of repos to do more things like nerds, status, rm, info, open etc

when open is selected it opens the repo asking for the editor saved in configure
