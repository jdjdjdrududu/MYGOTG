#!/bin/bash

cd /home/gobotuser/go/src/mygotelegrambot || exit 1

git add .

message="Auto-commit on $(date '+%Y-%m-%d %H:%M:%S')"
git commit -m "$message" 2>/dev/null

git push origin main
