#!/bin/bash

TARGET=$1
COMMIT_MSG=$2

if [ -z "$TARGET" ]; then
  echo "target required"
  exit 1
fi

if [ -z "$COMMIT_MSG" ]; then
  echo "commit message required"
  exit 1
fi

./vuln-list-update -vuln-list-dir "vuln-list-k8s" -target "k8s"

cd "$VULN_LIST_DIR" || exit 1

if [[ -n $(git status --porcelain) ]]; then
  # List changed files
CHANGED_FILES=$(git ls-files . --exclude-standard --others)

# Loop through changed files and create PRs
for FILE in $CHANGED_FILES; do
  REPO="github.com/aquasecurity/vuln-list-k8s"
  BRANCH_NAME=$(echo $FILE | tr / -)
  PR_TITLE="Update $FILE"
  PR_BODY="This PR updates $FILE"

  # Check if a PR with the same branch name already exists
 ## if gh pr list --state open --repo "github.com/aquasecurity/vuln-list-k8s" | grep -q "No open pull requests"; then
    open_pr_count=$(gh pr list --state open --base main --repo  github.com/aquasecurity/vuln-list-k8s  | grep $FILE | wc -l)


  if [ "$open_pr_count" -eq 0 ]; then
    # Create a new branch and push it
    git checkout -b $BRANCH_NAME
    git add $FILE
    git commit -m "Update $FILE"
    git push origin $BRANCH_NAME

    # Create a new pull request using gh
    gh pr create --base "$BASE_BRANCH" --head "$BRANCH_NAME" --title "$PR_TITLE" --body "$PR_BODY" --repo "$REPO"
  else
    echo "PR for $FILE already exists, skipping."
  fi
done
fi
