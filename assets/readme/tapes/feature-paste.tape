# Paste an image from micro: copy an image, open the note in micro, run pasteimage (Alt-i). The fake
# clipboard of demo-env.sh holds the demo diagram, so nothing of yours is read.
Output assets/readme/feature-paste.gif
Source assets/readme/tapes/lib/common.tape

# go to the last note and open it in micro (the editor lazymark finds)
Type "G"
Sleep 800ms
Type "e"
Wait+Screen@10s /Ctrl-e|\(1,1\)|micro/
Sleep 1.5s
Down
Down
Sleep 500ms
Enter
Up
Sleep 700ms
# the recorder cannot send Alt-i to micro (it types an "i"), so the command is used: Alt-i does the same
Ctrl+E
Sleep 600ms
Type "pasteimage"
Sleep 600ms
Enter
Sleep 2.5s
Ctrl+S
Sleep 1s
Ctrl+Q
Sleep 2s

Type "q"
