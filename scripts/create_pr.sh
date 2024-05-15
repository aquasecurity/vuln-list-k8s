#!/bin/bash -eu
# Copyright 2024 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

./vuln-list-k8s -vuln-list-dir "$VULN_LIST_DIR"

cd "$VULN_LIST_DIR" || exit 1

if [[ -n $(git status --porcelain) ]]; then
  # List changed files
  CHANGED_FILES=$(git ls-files . --exclude-standard --others | grep "CVE")
  REPO="$REPOSITORY_OWNER/$VULN_LIST_DIR"
  BASE_BRANCH="main"
  # Loop through changed files and create PRs
  for FILE in $CHANGED_FILES; do

    BRANCH_NAME=$(echo "$FILE" | tr / -)
    PR_TITLE="Update $FILE"
    PR_BODY="This PR updates $FILE"

    # Check if a PR with the same branch name already exists
    OPEN_PR_COUNT=$(gh pr list --state open --base $BASE_BRANCH --repo "$REPO"  --limit 50| grep "$FILE" | wc -l)

    if [ "$OPEN_PR_COUNT" != 0 ]; then
      echo "PR for $FILE already exists, skipping."
      continue
    fi

    # Create a new branch and push it
    git checkout -b "$BRANCH_NAME"
    echo "$FILE"
    git add "$FILE"
    git commit -m "Update $FILE"

    git push origin "$BRANCH_NAME" --force
    # Create a new pull request using gh
    gh pr create --base "$BASE_BRANCH" --head "$BRANCH_NAME" --title "$PR_TITLE" --body "$PR_BODY" --repo "$REPO"

    git checkout $BASE_BRANCH

    sleep 30
  done
fi
