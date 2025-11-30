#!/bin/bash
# Simple non-interactive test - just prints escape sequences and text
# Run this and observe where the output appears

echo "=== Simple TUI cursor test ==="

# Print 3 lines
echo "Line A"
echo "Line B"
echo -n "Line C ->"

# Enter alternate screen
printf '\e[?1049h'
# Clear and home
printf '\e[2J\e[H'
# Print in alternate screen
printf 'ALT SCREEN LINE 1\n'
printf 'ALT SCREEN LINE 2\n'
# Move cursor around
printf '\e[10;20H'
printf 'At row 10 col 20'
# Small delay to see it
sleep 0.5

# Exit alternate screen
printf '\e[?1049l'

# This should appear right after "Line C ->"
echo "<- After exit (same line as C)"
echo "Line D"
echo "Line E"

echo ""
echo "=== Test with smcup/rmcup style (like less/vim) ==="

echo "Before 1"
echo "Before 2"
echo -n "Before 3 ->"

# smcup: save cursor + enter alt screen + clear
printf '\e7\e[?1049h\e[2J\e[H'
printf 'In alternate\n'
sleep 0.3

# rmcup: exit alt screen + restore cursor
printf '\e[?1049l\e8'

echo "<- Should be same line"
echo "After line"

echo ""
echo "=== Test scroll region in alt screen ==="

echo "Pre-scroll 1"
echo "Pre-scroll 2"
echo -n "Pre-scroll 3 ->"

printf '\e[?1049h\e[2J\e[H'
# Set scroll region to rows 5-15
printf '\e[5;15r'
# Move to row 10
printf '\e[10;1H'
printf 'In scroll region'
sleep 0.3
printf '\e[?1049l'

# Scroll region should be reset to full screen
echo "<- After alt (scroll region should be restored)"
echo "More text"
echo "Even more"

echo ""
echo "Done!"
