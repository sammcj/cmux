#!/bin/bash
# Test script to reproduce TUI cursor positioning issues

echo "=== Test 1: Basic alternate screen enter/exit ==="
echo "Line 1 before TUI"
echo "Line 2 before TUI"
echo "Line 3 before TUI"
echo -n "Cursor should be here -> "

# Enter alternate screen (mode 1049)
printf '\e[?1049h'
# Clear alternate screen
printf '\e[2J'
# Move to top
printf '\e[H'
echo "Inside alternate screen"
echo "Press enter to exit..."
read -r

# Exit alternate screen (mode 1049)
printf '\e[?1049l'

echo "<- Line after TUI (should be on same line as arrow)"
echo "Line 2 after TUI"
echo ""

echo "=== Test 2: TUI with origin mode ==="
echo "Line before TUI"
echo -n "Cursor here -> "

# Enter alternate screen
printf '\e[?1049h'
printf '\e[2J\e[H'
# Set scroll region
printf '\e[5;20r'
# Enable origin mode
printf '\e[?6h'
echo "Inside alternate screen with origin mode"
echo "Press enter to exit..."
read -r

# Exit alternate screen (should restore origin mode to OFF)
printf '\e[?1049l'

echo "<- After TUI (should be same line)"
echo ""

echo "=== Test 3: TUI with cursor save/restore ==="
echo "Line 1"
echo "Line 2"
echo -n "Line 3 cursor -> "
# Save cursor (DECSC)
printf '\e7'
echo ""
echo "Line 4"
echo "Line 5"

# Enter alternate screen
printf '\e[?1049h'
printf '\e[2J\e[H'
# Save a different cursor in alternate screen
printf '\e[10;10H'
printf '\e7'
echo "Saved cursor at 10,10 in alt screen"
echo "Press enter to exit..."
read -r

# Exit alternate screen
printf '\e[?1049l'

# Restore cursor (should go back to Line 3 position, not 10,10)
printf '\e8'
echo "<- Should be on Line 3"
echo ""

echo "=== Test 4: Check actual cursor position ==="
echo "Line 1"
echo "Line 2"
echo -n "Line 3 -> "

# Query cursor position before
printf '\e[6n'
read -r -d 'R' pos
echo "Position before: $pos"

# Enter alternate screen
printf '\e[?1049h'
printf '\e[2J\e[H'
printf '\e[15;40H'
echo "Moved to row 15, col 40 in alt screen"
read -r

# Exit alternate screen
printf '\e[?1049l'

# Query cursor position after
printf '\e[6n'
read -r -d 'R' pos
echo "Position after: $pos"
echo "Next line after query"

echo ""
echo "=== All tests complete ==="
