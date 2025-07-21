#!/bin/bash

# Change to the project directory
cd /home/gobotuser/go/src/mygotelegrambot || { echo "Failed to change directory"; exit 1; }

# Check if there are any changes to commit
if [[ -z $(git status --porcelain) ]]; then
  echo "No changes to commit"
  exit 0
fi

# Add all changes first
echo "Adding changes to git..."
git add .

# Commit with timestamp
message="Auto-commit on $(date '+%Y-%m-%d %H:%M:%S')"
echo "Committing: $message"
if ! git commit -m "$message"; then
  echo "Failed to commit changes"
  exit 1
fi

# Pull latest changes with rebase to avoid merge commits
echo "Pulling latest changes from GitHub..."
if ! git pull --rebase origin main; then
  echo "Failed to pull with rebase. Trying to resolve conflicts..."
  # If rebase fails, try to abort it and continue with force push
  git rebase --abort
fi

# Push to GitHub with force if needed
echo "Pushing to GitHub..."
if ! git push origin main; then
  echo "Regular push failed, trying with --force-with-lease..."
  if ! git push --force-with-lease origin main; then
    echo "Failed to push to GitHub. Check your credentials."
    echo "You might need to set up a GitHub token or SSH key."
    exit 1
  fi
fi

echo "Successfully pushed to GitHub!"
