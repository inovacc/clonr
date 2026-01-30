# Command Tree

```
clonr
+-- add                                      # Register an existing local Git reposi...
+-- branches                                 # List and manage git branches
+-- clone                                    # Clone a Git repository
+-- cmdtree                                  # Display command tree visualization
+-- config                                   # Manage clonr configuration
|   \-- editor                               # Manage custom editors
|       +-- add                              # Add a new custom editor
|       +-- list                             # List all editors
|       \-- remove                           # Remove a custom editor
+-- configure                                # Configure clonr settings
+-- diff                                     # Show git diff for a repository
+-- favorite                                 # Mark a repository as favorite
+-- gh                                       # GitHub operations for repositories
|   +-- actions                              # Check GitHub Actions workflow status
|   |   \-- status                           # Check workflow run status
|   +-- contributors                         # View repository contributors and thei...
|   |   +-- journey                          # View a contributor's activity journey
|   |   \-- list                             # List repository contributors
|   +-- issues                               # Manage GitHub issues
|   |   +-- close                            # Close an issue
|   |   +-- create                           # Create a new issue
|   |   \-- list                             # List issues for a repository
|   +-- pr                                   # Check pull request status
|   |   \-- status                           # Check pull request status
|   \-- release                              # Manage GitHub releases
|       +-- create                           # Create a new release
|       +-- download                         # Download release assets
|       \-- list                             # List releases for a repository
+-- list                                     # Interactively list all repositories
+-- map                                      # Scan directory for existing Git repos...
+-- mirror                                   # Mirror all repositories from a GitHub...
+-- nerds                                    # Display repository statistics
+-- open                                     # Open a repository in your configured ...
+-- org                                      # Manage GitHub organizations
|   +-- list                                 # List your GitHub organizations
|   \-- mirror                               # Mirror all repositories from a GitHub...
+-- pm                                       # Project management tool integrations
|   +-- jira                                 # Jira operations for project management
|   |   +-- auth                             # Open Jira/Atlassian token page in bro...
|   |   +-- boards                           # Manage Jira boards
|   |   |   \-- list                         # List Jira boards
|   |   +-- issues                           # Manage Jira issues
|   |   |   +-- create                       # Create a new Jira issue
|   |   |   +-- list                         # List issues for a Jira project
|   |   |   +-- transition                   # Move an issue to a new status
|   |   |   \-- view                         # View details of a Jira issue
|   |   \-- sprints                          # Manage Jira sprints
|   |       +-- current                      # Show the current active sprint
|   |       \-- list                         # List sprints for a board
|   \-- zenhub                               # ZenHub operations for project management
|       +-- auth                             # Open ZenHub and GitHub token pages in...
|       +-- board                            # View ZenHub board state
|       +-- epic                             # Show epic with children and progress
|       +-- epics                            # List ZenHub epics
|       +-- issue                            # View ZenHub issue details
|       +-- issues                           # List issues with ZenHub enrichment
|       +-- move                             # Move issue to a different pipeline
|       \-- workspaces                       # List ZenHub workspaces
+-- profile                                  # Manage GitHub authentication profiles
|   +-- add                                  # Create a new profile with GitHub OAuth
|   +-- list                                 # List all profiles
|   +-- remove                               # Delete a profile
|   +-- status                               # Show profile information
|   \-- use                                  # Set the active profile
+-- reauthor                                 # Rewrite git history to change author/...
+-- remove                                   # Remove repository from management
+-- repo                                     # Repository operations
|   +-- edit                                 # Open repository in selected editor
|   \-- open                                 # Open repository folder in file manager
+-- server                                   # Server management commands
|   \-- start                                # Start the gRPC server
+-- service                                  # Manage clonr server as a system service
+-- snapshot                                 # Export database to JSON snapshot
+-- stats                                    # Show git statistics for a repository
+-- status                                   # Show git status of repositories
+-- tpm                                      # TPM 2.0 + KeePass key management
|   +-- init                                 # Initialize TPM key + KeePass database
|   +-- migrate                              # Migrate existing KeePass DB to TPM
|   +-- migrate-profiles                     # Migrate profiles from keyring to KeePass
|   +-- reset                                # Remove TPM-sealed key
|   \-- status                               # Show TPM + KeePass status
+-- unfavorite                               # Remove favorite mark from a repository
+-- update                                   # Check for and install updates
\-- version                                  # Print version information
