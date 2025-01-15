## Contribution Guidelines
* Minor changes can be done directly by editing code on GitHub. GitHub automatically creates a temporary branch and
  files a PR. This is only suitable for really small changes like: spelling fixes, variable name changes or error string
  change etc. For larger commits, following steps are recommended.
* (Optional) If you want to discuss your implementation with the users of GoFr CLI, use the GitHub discussions of this repo.
* Configure your editor to use goimport and golangci-lint on file changes. Any code which is not formatted using these
  tools, will fail on the pipeline.
* All code contributions should have associated tests and all new line additions should be covered in those testcases.
  No PR should ever decrease the overall code coverage.
* Once your code changes are done along with the testcases, submit a PR to development branch. Please note that all PRs
  are merged from feature branches to development first.
* PR should be raised only when development is complete and the code is ready for review. This approach helps reduce the number of open pull requests and facilitates a more efficient review process for the team.
* All PRs need to be reviewed by at least 2 GoFr developers. They might reach out to you for any clarification.
* Thank you for your contribution. :)
