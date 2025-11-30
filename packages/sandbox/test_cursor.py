#!/usr/bin/env python3
"""
Precise cursor position test for debugging TUI cursor issues.
This script carefully tracks cursor positions before/after alternate screen switches.
"""

import sys
import os
import time
import termios
import tty
import select

def write(s):
    """Write string to stdout and flush."""
    sys.stdout.write(s)
    sys.stdout.flush()

def read_response(timeout=1.0):
    """Read response from terminal with timeout."""
    fd = sys.stdin.fileno()
    old_settings = termios.tcgetattr(fd)
    try:
        tty.setraw(fd)
        response = ""
        end_time = time.time() + timeout
        while time.time() < end_time:
            if select.select([sys.stdin], [], [], 0.1)[0]:
                char = sys.stdin.read(1)
                response += char
                if char == 'R':  # End of cursor position response
                    break
        return response
    finally:
        termios.tcsetattr(fd, termios.TCSADRAIN, old_settings)

def get_cursor_position():
    """Query and return current cursor position as (row, col)."""
    write('\x1b[6n')  # DSR - Device Status Report
    response = read_response()
    # Response format: ESC [ row ; col R
    try:
        # Strip ESC [ and R
        response = response.lstrip('\x1b[').rstrip('R')
        row, col = response.split(';')
        return int(row), int(col)
    except:
        return None, None

def main():
    write("\x1b[2J\x1b[H")  # Clear screen and home cursor
    write("=== Cursor Position Debug Test ===\n\n")

    # Test 1: Basic position tracking
    write("Test 1: Basic alternate screen\n")
    write("Line 1\n")
    write("Line 2\n")
    write("Line 3\n")

    pos_before = get_cursor_position()
    write(f"Position BEFORE alt screen: row={pos_before[0]}, col={pos_before[1]}\n")

    # Enter alternate screen
    write('\x1b[?1049h')  # DECSET 1049
    write('\x1b[2J\x1b[H')  # Clear and home
    write("IN ALTERNATE SCREEN\n")
    write("Moving to row 10, col 20...\n")
    write('\x1b[10;20H')
    write("X")
    time.sleep(0.3)

    # Exit alternate screen
    write('\x1b[?1049l')  # DECRST 1049

    pos_after = get_cursor_position()
    write(f"Position AFTER alt screen: row={pos_after[0]}, col={pos_after[1]}\n")
    write(f"Expected: row={pos_before[0]}, col={pos_before[1]}\n")

    if pos_before == pos_after:
        write("✓ PASS: Cursor position restored correctly\n")
    else:
        write(f"✗ FAIL: Cursor position NOT restored! Diff: row={pos_after[0]-pos_before[0]}, col={pos_after[1]-pos_before[1]}\n")

    write("\n")

    # Test 2: With origin mode
    write("Test 2: Origin mode preservation\n")
    write("Setting up...\n")

    pos_before2 = get_cursor_position()
    write(f"Position before: {pos_before2}\n")

    # Enter alternate screen
    write('\x1b[?1049h')
    write('\x1b[2J\x1b[H')
    # Set scroll region
    write('\x1b[5;20r')
    # Enable origin mode
    write('\x1b[?6h')
    write("Origin mode ON, scroll region 5-20\n")
    time.sleep(0.3)

    # Exit alternate screen
    write('\x1b[?1049l')

    pos_after2 = get_cursor_position()
    write(f"Position after: {pos_after2}\n")

    # Try to move to absolute position to test if origin mode is still on
    write('\x1b[1;1H')  # Should go to row 1, col 1 if origin mode is OFF
    pos_test = get_cursor_position()
    write(f"After CUP(1,1): {pos_test}\n")

    if pos_test[0] == 1:
        write("✓ PASS: Origin mode correctly restored to OFF\n")
    else:
        write(f"✗ FAIL: Origin mode still ON! CUP(1,1) went to row {pos_test[0]}\n")

    write("\n")

    # Test 3: Multiple enter/exit cycles
    write("Test 3: Multiple cycles\n")
    for i in range(3):
        write(f"Cycle {i+1}: ")
        pos_b = get_cursor_position()

        write('\x1b[?1049h')
        write('\x1b[2J\x1b[H')
        write(f"Alt screen cycle {i+1}")
        time.sleep(0.1)
        write('\x1b[?1049l')

        pos_a = get_cursor_position()
        if pos_b[0] == pos_a[0] and pos_b[1] == pos_a[1]:
            write("OK\n")
        else:
            write(f"FAIL (before={pos_b}, after={pos_a})\n")

    write("\n=== Tests Complete ===\n")
    write("Press Enter to exit...")
    input()

if __name__ == "__main__":
    main()
