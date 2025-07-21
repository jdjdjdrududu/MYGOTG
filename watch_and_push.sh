#!/bin/bash

WATCH_DIR="/home/gobotuser/go/src/mygotelegrambot"

inotifywait -m -r -e modify,create,delete "$WATCH_DIR" --format '%w%f' | while read FILE
do
  echo "Изменён файл: $FILE"
  ./auto_git_push.sh
done
