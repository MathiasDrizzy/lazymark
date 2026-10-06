# Isolated demo environment for recording the README GIFs. Use it with `source`
# from the repository root (the tapes do it for you):
#   - builds lazymark into a temporary folder and puts it on PATH;
#   - uses a temporary HOME with a copy of assets/readme/demo-home, so recording
#     never touches real notes or writes inside the repository;
#   - fakes the clipboard and installs the micro plugin in that HOME;
#   - fixes the notes' dates so the dates shown are always the same.
# a fixed, neutral folder (not a random one in the system temp dir): some editors show the full path of the note
DEMO_TMP=/tmp/lazymark-demo
rm -rf "${DEMO_TMP:?}"
mkdir -p "$DEMO_TMP"
go build -o "$DEMO_TMP/lazymark" ./cmd/lazymark || return 1   # before changing HOME: Go uses its cache there
export HOME="$DEMO_TMP/home"
mkdir -p "$HOME"
cp -R assets/readme/demo-home/. "$HOME/"
# The dates of the demo tasks are written for 2026-10-06; they are moved by as many days as separate that day from today, so that the recorded colors
# (overdue, due today, on time, started, not started, completed) always look the same whichever day you record.
python3 - "$HOME/Documents/notes" <<'PYDEMO'
import datetime, pathlib, re, sys
delta = (datetime.date.today() - datetime.date(2026, 10, 6)).days
field = re.compile(r"(\[(?:start|due|completion|scheduled|created):: )(\d{4}-\d{2}-\d{2})(\])")
for p in pathlib.Path(sys.argv[1]).rglob("*.md"):
    s = p.read_text()
    t = field.sub(lambda m: m.group(1) + (datetime.date.fromisoformat(m.group(2)) + datetime.timedelta(days=delta)).isoformat() + m.group(3), s)
    if t != s:
        p.write_text(t)
PYDEMO
# A fake clipboard (never the real one): `pngpaste`, `wl-paste` and `osascript` that "have copied"
# the demo architecture diagram, so `lazymark paste` can be recorded without touching yours.
mkdir -p "$DEMO_TMP/clip"
printf '#!/bin/sh\ncp "$HOME/Documents/notes/assets/architecture.png" "$1"\n' > "$DEMO_TMP/clip/pngpaste"
printf '#!/bin/sh\ncase "$*" in *image/png*) cat "$HOME/Documents/notes/assets/architecture.png" ;; *) exit 1 ;; esac\n' > "$DEMO_TMP/clip/wl-paste"
printf '#!/bin/sh\nexit 1\n' > "$DEMO_TMP/clip/osascript"
chmod +x "$DEMO_TMP/clip/"*
export PATH="$DEMO_TMP/clip:$DEMO_TMP:$PATH"
lazymark editor-plugins install micro >/dev/null   # into the temporary HOME
export LANG=en_US.UTF-8
unset LC_ALL LC_MESSAGES

# fixed date per note (YYYYMMDDhhmm)
set_time() { touch -t "$1" "$HOME/Documents/notes/$2"; }
set_time 202609301800 welcome.md
set_time 202610050900 inbox.md  # the newest note: its dated tasks come first in the Tasks panel (the recordings show the date colors there)
set_time 202609291100 reading-list.md
set_time 202609281500 meeting-notes.md
set_time 202609271000 architecture.md
set_time 202609261700 groceries.md
set_time 202609301400 projects/website-redesign.md
set_time 202609291600 projects/mobile-app.md
set_time 202609302200 journal/2026-09-30.md
set_time 202609292200 journal/2026-09-29.md
unset -f set_time
cd "$HOME" || return 1
clear
