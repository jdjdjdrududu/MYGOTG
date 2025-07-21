#!/bin/bash

# Change to the project directory
cd /home/gobotuser/go/src/mygotelegrambot || { echo "Failed to change directory"; exit 1; }

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

# Push to GitHub
echo "Pushing to GitHub..."
if ! git push origin main; then
  echo "Failed to push to GitHub. Check your credentials."
  echo "You might need to set up a GitHub token or SSH key."
  exit 1
fi

echo "Successfully pushed to GitHub!"
