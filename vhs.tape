# VHS Demo for lifx
# Run with: vhs vhs.tape

Output vhs.gif
Output vhs.mp4
Output vhs.webm

Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 50ms
Set Shell "bash"

# Start the app
Type "./lifx --backend lan"
Enter
Sleep 2500ms

# Navigate to a group/room (adjust counts for your setup)
Down
Sleep 150ms
Down
Sleep 150ms
Down
Sleep 400ms

# Toggle side panel
Tab
Sleep 400ms
Tab
Sleep 400ms

# Move to a light under the group
Down
Sleep 200ms

# Toggle light off/on
Space
Sleep 600ms
Space
Sleep 600ms

# Brightness up/down
Type "l"
Sleep 150ms
Type "l"
Sleep 150ms
Type "h"
Sleep 600ms

# Color temperature warm/cool
Type "w"
Sleep 400ms
Type "c"
Sleep 600ms

# Cycle color presets
Type "p"
Sleep 400ms
Type "p"
Sleep 600ms

# Go back to group header
Up
Sleep 200ms
Up
Sleep 400ms

# Group on/off
Type "a"
Sleep 600ms
Type "x"
Sleep 600ms

# Group brightness
Type "l"
Sleep 150ms
Type "l"
Sleep 600ms

# Open scenes
Type "s"
Sleep 800ms
Down
Sleep 250ms
Enter
Sleep 1200ms
Escape
Sleep 400ms

# Search/filter
Type "/"
Sleep 300ms
Type "desk"
Sleep 600ms
Enter
Sleep 500ms

# Set brightness to 50%
Type "5"
Sleep 800ms

# Clear search and quit
Escape
Sleep 300ms
Type "q"
