# Demo for Reddit (not part of the README): notes -> tick a task -> Kanban, move a card with colored dates -> set a date with `d` -> search -> back.
# Wide format (1280x720) and a big font so it reads on a phone. Record from the repository root with: vhs assets/social/reddit-demo.tape
Output assets/social/reddit-demo.mp4
Output assets/social/reddit-demo.gif
Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 22
Set Width 1280
Set Height 720
Set Padding 14
Set Theme "Catppuccin Mocha"
Set TypingSpeed 80ms
Set Framerate 25

Hide
Type "source assets/readme/demo-env.sh" Enter
Wait@60s
Type "lazymark" Enter
Wait+Screen@20s /Notes/
Show
Sleep 1.2s

# the Tasks panel, with the due dates in the color of their state
Type "2"
Sleep 1.4s
Down
Sleep 700ms
Space
Sleep 1.6s

# the Kanban board: the cards carry their dates (overdue, today, on time, started, completed)
Type "W"
Sleep 2.2s
Type "L"
Sleep 1.6s
# back to the first column, to a card with no dates yet
Left
Sleep 700ms
Down@500ms 2
Sleep 700ms

# `d` on a card: the dates the way you say them
Type "d"
Sleep 1s
Type "+3d"
Sleep 1s
Tab
Sleep 500ms
Type "friday"
Sleep 1.2s
Enter
Sleep 1.6s
Type "W"
Sleep 1s

# search across all the notes
Type "/"
Sleep 600ms
Type "report"
Sleep 1.6s
Escape
Sleep 800ms
Type "q"
