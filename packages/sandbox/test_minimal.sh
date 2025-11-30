#!/bin/bash
# Minimal reproduction test - run this and observe where output appears

# Test case 1: Simple alt screen with content before
printf 'Line1\nLine2\nLine3\n'
printf 'Before->'
printf '\e[?1049h'  # Enter alt screen
printf '\e[H\e[2J'  # Clear and home
printf 'ALT'
sleep 0.2
printf '\e[?1049l'  # Exit alt screen
printf '<-After\n'
printf 'NextLine\n'
echo ""

# Test case 2: With scrolling before entering alt screen
printf '=== Test 2 ===\n'
for i in {1..5}; do printf "Scroll$i\n"; done
printf 'BeforeAlt->'
printf '\e[?1049h\e[H\e[2JALT'
sleep 0.2
printf '\e[?1049l'
printf '<-AfterAlt\n'
echo ""

# Test case 3: Vim-like sequence (smcup/rmcup)
printf '=== Test 3: vim-like ===\n'
printf 'Pre1\nPre2\nPre3\n'
printf 'VimStart->'
# vim's smcup: ESC7 (save cursor) + enter alt + clear
printf '\e7\e[?1049h\e[H\e[2J'
printf 'EDITING\n'
sleep 0.2
# vim's rmcup: exit alt + ESC8 (restore cursor)
printf '\e[?1049l\e8'
printf '<-VimEnd\n'
printf 'PostVim\n'
echo ""

# Test case 4: htop-like (no DECSC/DECRC)
printf '=== Test 4: htop-like ===\n'
printf 'H1\nH2\nH3\n'
printf 'Htop->'
printf '\e[?1049h\e[H\e[2JHTOP_UI'
sleep 0.2
printf '\e[?1049l'
printf '<-HtopDone\n'
echo ""

# Test case 5: With scroll region in alt screen
printf '=== Test 5: scroll region ===\n'
printf 'S1\nS2\nS3\n'
printf 'ScrollRgn->'
printf '\e[?1049h\e[H\e[2J'
printf '\e[5;15r'  # Set scroll region
printf '\e[10;1H'  # Move inside region
printf 'IN_REGION'
sleep 0.2
printf '\e[?1049l'
printf '<-SRDone\n'
echo ""

# Test case 6: With origin mode
printf '=== Test 6: origin mode ===\n'
printf 'O1\nO2\nO3\n'
printf 'Origin->'
printf '\e[?1049h\e[H\e[2J'
printf '\e[?6h'  # Enable origin mode
printf '\e[5;1H'  # This position is now relative to scroll region
printf 'ORIGIN_ON'
sleep 0.2
printf '\e[?1049l'
printf '<-OriginDone\n'
echo ""

printf '=== ALL TESTS DONE ===\n'
