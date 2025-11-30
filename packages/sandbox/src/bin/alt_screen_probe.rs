use anyhow::{Context, Result};
use cmux_sandbox::mux::terminal::{TerminalBuffer, TerminalRenderView};

fn render_choice(
    current: Option<TerminalRenderView>,
    previous: Option<TerminalRenderView>,
    force_current: bool,
) -> Option<TerminalRenderView> {
    if force_current {
        return current.or(previous);
    }

    current.or(previous)
}

fn line_as_string(line: &ratatui::text::Line<'_>) -> String {
    line.spans.iter().map(|s| s.content.as_ref()).collect()
}

fn view_contains(view: &TerminalRenderView, needle: &str) -> bool {
    view.lines
        .iter()
        .any(|line| line_as_string(line).contains(needle))
}

fn scenario(
    force_clear: bool,
    start_scrolled: bool,
    alt_has_content: bool,
    shorter_alt_line: bool,
) -> Result<(bool, bool, usize)> {
    // returns (overlay_present, scroll_offset_zero, changed_lines_len)
    let height = 12usize;
    let mut buffer = TerminalBuffer::with_size(height, 40);

    // Seed main screen
    buffer.process(b"MAIN1\nMAIN2\nMAIN3 ->");
    if start_scrolled {
        buffer.scroll_up(2);
    }

    let previous_view = buffer.render_view(height);

    // Enter alternate screen and optionally write content
    let mut sequence = b"\x1b[?1049h\x1b[H\x1b[2J".to_vec();
    if alt_has_content {
        sequence.extend_from_slice(b"ALT1\nALT2\n");
        if shorter_alt_line {
            sequence.extend_from_slice(b"X\n");
        } else {
            sequence.extend_from_slice(b"ALT LONG LINE HERE\n");
        }
    }
    buffer.process(&sequence);

    // In UI we would consume the flag once; mirror that behavior here.
    let force_flag = if buffer.needs_full_clear {
        force_clear
    } else {
        false
    };
    buffer.needs_full_clear = false;

    let current_view = buffer.render_view(height);
    let render_view = render_choice(Some(current_view), Some(previous_view), force_flag)
        .context("missing render view")?;

    let overlay = view_contains(&render_view, "MAIN");
    let offset_zero = buffer.scroll_offset() == 0;
    let changed_len = render_view.changed_lines.len();

    Ok((overlay, offset_zero, changed_len))
}

fn main() -> Result<()> {
    println!("Alt screen probe (simulated UI render)");

    // Hypothesis 1: needs_full_clear not honored causes fallback overlay
    let (overlay1, _, changed1) = scenario(false, false, true, false)?;
    println!(
        "H1 (force_clear=false with alt content): overlay={}, changed_lines={}",
        overlay1, changed1
    );

    // Hypothesis 2: scroll offset not reset on alt entry
    let (overlay2, offset_zero2, _) = scenario(true, true, true, false)?;
    println!(
        "H2 (entered alt while scrolled): overlay={}, scroll_offset_zero={}",
        overlay2, offset_zero2
    );

    // Hypothesis 3: render cache not invalidated on alt toggle
    let (overlay3, _, changed3) = scenario(true, false, true, false)?;
    println!(
        "H3 (force_clear=true): overlay={}, changed_lines={}",
        overlay3, changed3
    );

    // Hypothesis 4: empty alt screen falls back to previous view
    let (overlay4, _, changed4) = scenario(false, false, false, false)?;
    println!(
        "H4 (alt has no content): overlay_from_fallback={}, changed_lines={}",
        overlay4, changed4
    );

    // Hypothesis 5: shorter alt lines leave residue
    let (overlay5, _, _) = scenario(true, false, true, true)?;
    println!(
        "H5 (shorter lines in alt): main_residue_present={}",
        overlay5
    );

    Ok(())
}
