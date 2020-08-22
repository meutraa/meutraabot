# Meutbot

Command (identifiers) are regex matchers on messages

Command (templates) are golang templates

## Builtin Command Usage

Usage|User|Mod|Channel Owner|Operator|Description
-----|----|---|-------------|--------|-----------
__!leave__|✓|✓|✓|✓|Leave the caller's chat
__!ban USERNAME__| | | |✓|Ban a user from all chats the bot is in
__!join__|✓|✓|✓|✓|Join the caller's chat
__!get COMMAND__|✓|✓|✓|✓|Show the command template for an identifier
__!set COMMAND__| |✓|✓|✓|Set a command template for an identifier
__!unset COMMAND__| |✓|✓|✓|Unset a command by identifier
__!list__|✓|✓|✓|✓|List all channel commands
__!gget COMMAND__|✓|✓|✓|✓|Show the global command template for an identifier
__!gset COMMAND__| | | |✓|Set a global command template for an identifier
__!gunset COMMAND__| | | |✓|Unset a global command by identifier
__!glist__|✓|✓|✓|✓|List all global commands
__!functions__|✓|✓|✓|✓|List functions for use in command templates
__!data__|✓|✓|✓|✓|List data for use in command templates
__!test TEMPLATE__| |✓|✓|✓|Test a template without creating a command
__!builtins__|✓|✓|✓|✓|List builtin commands
