# VHS tape file — run `vhs demo.tape` to generate demo.gif
# https://github.com/charmbracelet/vhs

Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 20

Type "hn search 'local LLMs' -n 5"
Sleep 500ms
Enter
Sleep 3s

Sleep 1s

Type "hn front -n 5"
Sleep 500ms
Enter
Sleep 3s

Sleep 1s

Type "hn read 43163011"
Sleep 500ms
Enter
Sleep 4s

Sleep 2s
