# Meutbot

Command (identifiers) are regex matchers on messages
Command (templates) are golang templates

## Builtin Command Usage

	### !leave

    Leave the caller's chat

	### !ban USERNAME (owner only)

    Ban a user from all chats the bot is in

	### !join

    Join the caller's chat

	### !get COMMAND

    Show the command template for an identifier

	### !set COMMAND (mod only)

    Set a command template for an identifier

	### !unset COMMAND (mod only)

    Unset a command by identifier

	### !list

    List all channel commands

	### !gget COMMAND

    Show the global command template for an identifier

	### !gset COMMAND (owner only)

    Set a global command template for an identifier

	### !gunset COMMAND (owner only)

    Unset a global command by identifier

	### !glist

    List all global commands

	### !functions

    List functions for use in command templates

	### !data

    List data for use in command templates

	### !test TEMPLATE

    Test a template without creating a command

	### !builtins

    List builtin commands
