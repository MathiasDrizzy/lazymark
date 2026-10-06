# Demo for the MCP post (not part of the README): the same operations an agent calls through the MCP tools, from the shell:
# list the pending tasks (dates in the color of their state on a terminal), move one to the doing column, read the board back.
# The id of a task is a hash of its text (no dates, no board tag), so it does not change from one recording to the next.
# Record from the repository root with: vhs assets/social/mcp-demo.tape
Output assets/social/mcp-demo.gif
Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 22
Set Width 1600
Set Height 540
Set Padding 18
Set Theme "Catppuccin Mocha"
Set TypingSpeed 22ms
Set Framerate 20

Hide
Type "source assets/readme/demo-env.sh" Enter
Wait@60s
Type "PS1='$ '" Enter
Type "clear" Enter
Show
Sleep 800ms

Type "# what an agent does through the MCP tools"
Sleep 400ms
Enter
Type "lazymark task list --pending --note inbox.md"
Sleep 500ms
Enter
Sleep 2.2s

Type "lazymark task move inbox.md#da1a5f75 doing"
Sleep 500ms
Enter
Sleep 1.4s

Type "lazymark task list --column doing"
Sleep 500ms
Enter
Sleep 2.2s
