#!/bin/bash
# Comprehensive TUI cursor reproduction test
# Run this inside the terminal emulator and observe where text appears

set -e

# Colors for visibility
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== TUI Cursor Bug Reproduction ===${NC}"
echo ""

# Test 1: Simple alt screen
echo -e "${GREEN}Test 1: Basic alternate screen${NC}"
echo "A1"
echo "A2"
echo -n "A3->"
sleep 0.2
printf '\e[?1049h\e[H\e[2J'
echo "IN ALT SCREEN"
sleep 0.5
printf '\e[?1049l'
echo "<-A3end"
echo "A4"
echo ""

# Test 2: vim-style smcup/rmcup (save cursor, alt screen, restore cursor)
echo -e "${GREEN}Test 2: vim-style (ESC7 + alt + ESC8)${NC}"
echo "B1"
echo "B2"
echo -n "B3->"
printf '\e7'  # Save cursor
sleep 0.2
printf '\e[?1049h\e[H\e[2J'
echo "VIM MODE"
sleep 0.5
printf '\e[?1049l'
printf '\e8'  # Restore cursor
echo "<-B3end"
echo "B4"
echo ""

# Test 3: With scroll region manipulation in alt screen
echo -e "${GREEN}Test 3: Scroll region in alt screen${NC}"
echo "C1"
echo "C2"
echo -n "C3->"
sleep 0.2
printf '\e[?1049h\e[H\e[2J'
printf '\e[5;15r'  # Set scroll region
printf '\e[10;1H'  # Move inside region
echo "SCROLL REGION"
sleep 0.5
printf '\e[?1049l'
echo "<-C3end"
echo "C4"
echo ""

# Test 4: Origin mode in alt screen
echo -e "${GREEN}Test 4: Origin mode in alt screen${NC}"
echo "D1"
echo "D2"
echo -n "D3->"
sleep 0.2
printf '\e[?1049h\e[H\e[2J'
printf '\e[5;15r'  # Set scroll region
printf '\e[?6h'    # Enable origin mode
printf '\e[1;1H'   # This goes to top of scroll region, not screen
echo "ORIGIN MODE"
sleep 0.5
printf '\e[?1049l'
echo "<-D3end"
echo "D4"
echo ""

# Test 5: Auto-wrap test
echo -e "${GREEN}Test 5: Auto-wrap in alt screen${NC}"
echo "E1"
echo "E2"
echo -n "E3->"
sleep 0.2
printf '\e[?1049h\e[H\e[2J'
printf '\e[?7l'    # Disable auto-wrap
echo "AUTOWRAP OFF"
sleep 0.5
printf '\e[?1049l'
echo "<-E3end"
echo "E4"
echo ""

# Test 6: Multiple rapid switches
echo -e "${GREEN}Test 6: Multiple rapid switches${NC}"
echo "F1"
echo "F2"
echo -n "F3->"
for i in 1 2 3; do
    printf '\e[?1049h\e[H\e[2JALT%d' "$i"
    sleep 0.1
    printf '\e[?1049l'
done
echo "<-F3end"
echo "F4"
echo ""

# Test 7: Cursor visibility toggle in alt screen
echo -e "${GREEN}Test 7: Cursor visibility in alt screen${NC}"
echo "G1"
echo "G2"
echo -n "G3->"
sleep 0.2
printf '\e[?1049h\e[H\e[2J'
printf '\e[?25l'  # Hide cursor
echo "CURSOR HIDDEN"
sleep 0.5
printf '\e[?1049l'
echo "<-G3end"
echo "G4"
echo ""

# Test 8: htop-style (clear screen, no cursor save)
echo -e "${GREEN}Test 8: htop-style (no DECSC/DECRC)${NC}"
echo "H1"
echo "H2"
echo -n "H3->"
sleep 0.2
printf '\e[?1049h\e[H\e[2J'
echo "HTOP"
sleep 0.5
printf '\e[?1049l'
echo "<-H3end"
echo "H4"
echo ""

# Test 9: Now run a simple program after all the TUI switches
echo -e "${GREEN}Test 9: Simple output after TUI operations${NC}"
echo "I1"
echo "I2"
printf 'I3->'
sleep 0.2
printf '\e[?1049h\e[H\e[2JALT'
sleep 0.3
printf '\e[?1049l'
# Now run a simple command
echo "<-done, running ls:"
ls -la 2>/dev/null | head -3 || echo "(ls not available)"
echo ""

echo -e "${YELLOW}=== Tests Complete ===${NC}"
echo ""
echo "Check each test:"
echo "- Text '<-XNend' should appear on same line as 'XN->'"
echo "- If '<-XNend' appears on a different line, that test failed"
echo "- Report which test numbers show incorrect cursor positioning"
