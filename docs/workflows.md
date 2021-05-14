# Workflows

This document tries to outline what different workflows users can have.  
Hopefully this can be the foundation for actual integration tests later on.

The bot uses role-based access control (RBAC) to manage users and their permissions.  
The idea is to allow this to happen through messages with the bot itself.

## Roles

### Admin

Admins can do basically everything.  
One user is specific at the beginning to be the initial Admin to be able to configure everything else via the bot.

Authorization (RBAC)  
`/rbac roles`  
`/rbac grant @user $role`

### Channel Admin

Channel Admins are users that aren't Admins but have certain privileges within a channel. 

`/alerts` - Shows a list of currently firing alerts.

### Channel Reader

Channel Reader

### Others

These are all other unknown users of Telegram.  
They shouldn't be able to do anything alerting specific.

Commands:  
`/start` - Start a conversation with the bot.  
`/stop` - Start a conversation with the bot.  
`/id` - Returns the users ID to use for initial admins to configure the bot.  
`/help` - Returns the help text. The text should contain the specific commands each Role has access to. 
