# Git Auto-Push System

This system automatically commits and pushes changes to GitHub whenever files in the project are modified.

## Components

1. **auto_git_push.sh** - Script that commits and pushes changes to GitHub
2. **watch_and_push.sh** - Script that watches for file changes and triggers auto_git_push.sh
3. **git-auto-push.service** - Systemd service that runs the watch script as a background service

## How It Works

The system uses `inotifywait` to monitor file changes in the project directory. When a change is detected, it:

1. Waits 2 seconds for all changes to complete
2. Adds all changes to git
3. Commits with a timestamp message
4. Pulls the latest changes from GitHub with rebase
5. Pushes the changes to GitHub (with force if necessary)

## Using the Auto-Push System

### Automatic Method (Systemd Service)

If the systemd service is not working correctly, you can use the manual method below.

### Manual Method

You can run the auto-push script manually at any time:

```
./auto_git_push.sh
```

This is the most reliable method and will:
1. Add all changes
2. Commit with a timestamp
3. Pull latest changes from GitHub
4. Push your changes to GitHub

## Managing the Service

### Check Service Status
```
sudo systemctl status git-auto-push.service
```

### Start the Service
```
sudo systemctl start git-auto-push.service
```

### Stop the Service
```
sudo systemctl stop git-auto-push.service
```

### Restart the Service
```
sudo systemctl restart git-auto-push.service
```

### Disable Auto-Start on Boot
```
sudo systemctl disable git-auto-push.service
```

### Enable Auto-Start on Boot
```
sudo systemctl enable git-auto-push.service
```

### View Service Logs
```
sudo journalctl -u git-auto-push.service
```

## Troubleshooting

If you encounter issues with the auto-push system:

1. Check if the service is running with `sudo systemctl status git-auto-push.service`
2. Check the logs with `sudo journalctl -u git-auto-push.service`
3. Ensure your Git credentials are properly configured
4. Try running the scripts manually to see any error messages:
   ```
   ./auto_git_push.sh
   ```

### Common Issues

1. **Dubious ownership error**:
   ```
   git config --global --add safe.directory /home/gobotuser/go/src/mygotelegrambot
   ```

2. **Authentication issues**:
   Set up a GitHub personal access token or SSH key for authentication

3. **Non-fast-forward errors**:
   The script includes logic to handle this by trying to pull first and using --force-with-lease if needed 