# Shared preamble of every README GIF: same theme, font, size and speed, and the
# isolated demo environment (assets/readme/demo-env.sh).
# Use it with: Source assets/readme/tapes/lib/common.tape
Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 16
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms
Set Framerate 25

Hide
Type "source assets/readme/demo-env.sh" Enter
Wait@60s
Type "lazymark" Enter
Wait+Screen@20s /Notes/
Show
Sleep 1s
