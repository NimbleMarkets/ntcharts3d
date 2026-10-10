# ntcharts3d gallery demo. Records the Kitty/PNG render path with ntrecord
# (VHS cannot capture Kitty graphics). Build the binary first:
#
#   task build-ex-gallery
#
# Then, from this directory:
#
#   ntrecord demo.tape
#
# Requires ntrecord built after the NO_COLOR fix (internal/record: NO_COLOR is
# filtered from the recorded program's environment). With an older ntrecord,
# run `env -u NO_COLOR ntrecord demo.tape` or the Kitty image records blank.

Output demo.gif

Require bash

Set Shell "bash"
Set Padding 0
Set Framerate 10
Set PlaybackSpeed 1.5
Set FontSize 12
Set Width 900
Set Height 560
Set Theme "Catppuccin Mocha"

Hide
Type "../../bin/ntcharts3d-gallery"
Enter
Wait+Screen@30s /q quit/
Show

Sleep 3s
Type "2"
Sleep 3s
Type "b"
Sleep 2s
Type "b"
Type "3"
Sleep 3s
Type "4"
Sleep 3s
Type "f"
Sleep 2s
Type "5"
Sleep 3s
Type "s"
Sleep 2s

Hide
Type "q"
