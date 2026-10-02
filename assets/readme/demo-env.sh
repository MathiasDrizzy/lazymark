# Isolated demo environment for recording the README GIFs. Use it with `source`
# from the repository root (the tapes do it for you):
#   - builds lazymark into a temporary folder and puts it on PATH;
#   - uses a temporary HOME with a copy of assets/readme/demo-home, so recording
#     never touches real notes or writes inside the repository;
#   - fixes the notes' dates so the dates shown are always the same.
DEMO_TMP="$(mktemp -d)"
go build -o "$DEMO_TMP/lazymark" ./cmd/lazymark || return 1   # before changing HOME: Go uses its cache there
export HOME="$DEMO_TMP/home"
mkdir -p "$HOME"
cp -R assets/readme/demo-home/. "$HOME/"
export PATH="$DEMO_TMP:$PATH"
export LANG=en_US.UTF-8
unset LC_ALL LC_MESSAGES

# fixed date per note (YYYYMMDDhhmm)
set_time() { touch -t "$1" "$HOME/Documents/notes/$2"; }
set_time 202609301800 welcome.md
set_time 202609300930 inbox.md
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
