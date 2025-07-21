#!/bin/bash

# Change to the project directory
cd /home/gobotuser/go/src/mygotelegrambot || { echo "Failed to change directory"; exit 1; }

# Pull latest changes first to avoid non-fast-forward errors
echo "Pulling latest changes from GitHub..."
git pull --rebase origin main || {
  echo "Failed to pull changes. There might be conflicts."
  echo "Continuing with local changes..."
}

# Check if there are any changes to commit
if [[ -z $(git status --porcelain) ]]; then
  echo "No changes to commit"
  exit 0
fi

# Add all changes
echo "Adding changes to git..."
git add .

# Commit with timestamp
message="Auto-commit on $(date '+%Y-%m-%d %H:%M:%S')"
echo "Committing: $message"
if ! git commit -m "$message"; then
  echo "Failed to commit changes"
  exit 1
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
