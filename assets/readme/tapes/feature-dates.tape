# Dates: press d on a task and type a date the way you say it; no emoji to type.
Output assets/readme/feature-dates.gif
Source assets/readme/tapes/lib/common.tape

# the Tasks panel shows the due dates in the color of their state (overdue, today, on time); five steps down is a task with no dates yet
Type "2"
Sleep 1.5s
Down@300ms 5
Sleep 600ms
Type "d"
Sleep 1.2s
Type "+3d"
Sleep 1.2s
Tab
Sleep 500ms
Type "friday"
Sleep 1.8s
Enter
Sleep 1.5s
Type "W"
Sleep 2.5s
Type "W"
Sleep 800ms

Type "q"
