# Settings: the theme changes live (and now paints the whole screen), and so does the screen background.
Output assets/readme/feature-settings.gif
Source assets/readme/tapes/lib/common.tape

# the Tasks panel behind the popup shows the dates; the new date rows below change their colors live
Type "2"
Sleep 600ms
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
# the rows added for the dates (the list scrolls): "Date colors" off and on, then how many days ahead counts as "due soon"
Down@100ms 7
Sleep 800ms
Right
Sleep 1.5s
Right
Sleep 1.2s
Down
Sleep 600ms
Right@500ms 3
Sleep 1.2s
Escape
Sleep 800ms

Type "q"
