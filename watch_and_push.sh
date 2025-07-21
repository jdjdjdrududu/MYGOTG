#!/bin/bash

WATCH_DIR="/home/gobotuser/go/src/mygotelegrambot"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Check if inotify-tools is installed
if ! command -v inotifywait &> /dev/null; then
    echo "Error: inotifywait not found. Please install inotify-tools package."
    echo "Run: sudo apt-get install inotify-tools"
    exit 1
fi

echo "Starting file watch on $WATCH_DIR"
echo "Press Ctrl+C to stop watching"

# Watch for file changes but ignore certain directories and file types
inotifywait -m -r -e modify,create,delete \
  --exclude '(/\.git/|/\.idea/|\.swp$|~$|\.tmp$|nohup\.out$)' \
  "$WATCH_DIR" --format '%w%f' | while read FILE
do
  echo "Changed file: $FILE"
  
  # Wait a short time for all changes to complete
  sleep 2
  
  # Execute the auto git push script
  "$SCRIPT_DIR/auto_git_push.sh"
done
