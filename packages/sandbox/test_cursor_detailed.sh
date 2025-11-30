#!/bin/bash
# Detailed cursor position test - outputs escape sequences in hex for debugging

# Function to show cursor position
show_pos() {
    # Request cursor position and capture response
    printf '\e[6n' >/dev/tty
    read -r -d R -t 1 CURPOS </dev/tty 2>/dev/null
    CURPOS=${CURPOS#*[}
    echo "Cursor at: $CURPOS (row;col)"
}

echo "=== Detailed Cursor Test ==="
echo ""

echo "Step 1: Initial state"
echo "Line 1"
echo "Line 2"
echo "Line 3"
show_pos

echo ""
echo "Step 2: About to enter alternate screen..."
printf '\e[?1049h'
printf '\e[2J\e[H'
echo "Now in ALTERNATE screen"
echo "Line 2 in alt"
show_pos
sleep 0.3

echo ""
echo "Step 3: Moving cursor in alt screen to row 5, col 10"
printf '\e[5;10H'
show_pos
sleep 0.3

echo ""
echo "Step 4: Exiting alternate screen..."
printf '\e[?1049l'
echo "Back in MAIN screen"
show_pos

echo ""
echo "Step 5: Printing more lines"
echo "After line 1"
echo "After line 2"
show_pos

echo ""
echo "=== Test Complete ==="
