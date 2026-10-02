# The commands without the interface: list the tasks, move one to a Kanban column (the id is stable), and read it back as JSON.
Output assets/readme/feature-cli.gif
Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 16
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set Framerate 25

Hide
Type "source assets/readme/demo-env.sh" Enter
Wait@60s
Type "clear" Enter
Show
Sleep 500ms
Type "lazymark task list --pending | head -5" Enter
Sleep 3s
Type "lazymark task move inbox.md#b6b5aed7 doing" Enter
Sleep 2s
Type "lazymark task list --column doing" Enter
Sleep 3s
Type "lazymark task move inbox.md#b6b5aed7 doing --json" Enter
Sleep 4s
