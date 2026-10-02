# Settings: the theme changes live (and now paints the whole screen), and so does the screen background.
Output assets/readme/feature-settings.gif
Source assets/readme/tapes/lib/common.tape

Type ","
Sleep 1.2s
# Left from the first theme wraps to the last one: Solarized Light, a light theme
Left
Sleep 1.8s
Right
Sleep 1s
# seven steps to the right: Dracula
Right@250ms 7
Sleep 1.8s
# "Screen background": theme (the whole screen) or terminal (yours shows through)
Down@100ms 6
Sleep 800ms
Right
Sleep 1.8s
Right
Sleep 1.5s
Escape
Sleep 800ms

Type "q"
