# Keyboard Shortcut Button — Valid Keys and Combos

This document lists which keys and combinations are valid for the _Keyboard Shortcut_ button type. During capture, the
app blocks OS actions (e.g., `Win`+`Tab`) so you can record it without executing it. The Windows key might still open
the Start Menu.

## Supported modifiers

- `Ctrl`
- `Alt`
- `Shift`
- `Win` (Windows key)

Notes:

- Left/right variants (e.g., Left `Ctrl`/Right `Ctrl`) are normalized to a generic modifier.

## Supported main keys (exhaustive)

### Letters

| `A` | `B` | `C` | `D` | `E` | `F` | `G` | `H` | `I` | `J` | `K` | `L` |
|:----|:----|:----|:----|:----|:----|:----|:----|:----|:----|:----|:----|
| `M` | `N` | `O` | `P` | `Q` | `R` | `S` | `T` | `U` | `V` | `W` | `X` |
| `Y` | `Z` |

### Digits (top row)

| `0` | `1` | `2` | `3` | `4` | `5` | `6` | `7` | `8` | `9` |
|:----|:----|:----|:----|:----|:----|:----|:----|:----|:----|

### Function keys

| `F1`  | `F2`  | `F3`  | `F4`  | `F5`  | `F6`  | `F7`  | `F8`  |
|:-----:|:-----:|:-----:|:-----:|:-----:|:-----:|:-----:|:-----:|
| `F9`  | `F10` | `F11` | `F12` | `F13` | `F14` | `F15` | `F16` |
| `F17` | `F18` | `F19` | `F20` | `F21` | `F22` | `F23` | `F24` |

### Arrow keys

| `Up` | `Down` | `Left` | `Right` |
|:-----|:-------|:-------|:--------|

### Special keys

|  `Tab`   | `CapsLock` | `Space` | `Enter`  | `Backspace` |   `Delete`    |
|:--------:|:----------:|:-------:|:--------:|:-----------:|:-------------:|
| `Insert` |   `Home`   |  `End`  | `PageUp` | `PageDown`  | `PrintScreen` |

### Numpad

| `Num0`    | `Num1`   | `Num2`     | `Num3`     | `Num4` | `Num5`    | `Num6` | `Num7` | `Num8` | `Num9` |
|:----------|:---------|:-----------|:-----------|:-------|:----------|:-------|:-------|:-------|:-------|
| `NumLock` | `Divide` | `Multiply` | `Subtract` | `Add`  | `Decimal` |

### Media keys

| `VolumeMute` | `VolumeDown` | `VolumeUp` | `MediaNext` | `MediaPrevious` | `MediaStop` | `MediaPlayPause` |
|:-------------|:-------------|:-----------|:------------|:----------------|:------------|:-----------------|

### Symbols / punctuation (may be region-dependent)

|  `;` (Semicolon)   |    `=` (Equal)    |   `,` (Comma)   |    `-` (Minus)     | `.` (Period) | `/` (Slash) |
|:------------------:|:-----------------:|:---------------:|:------------------:|:------------:|:-----------:|
| `` ` `` (Backtick) | `[` (BracketOpen) | `\` (Backslash) | `]` (BracketClose) | `'` (Quote)  |

## Examples of valid shortcuts

- `Win`+`Tab`
- `Ctrl`+`C`
- `Alt`+`F4`
- `Win`+`D`
- `Shift`+`F12`
- `Ctrl`+`Alt`+`T`

## Limitations

- Layout-dependent OEM/symbol keys (e.g., some `<`, `>`, `^`, `~`, `|`) may not be accepted as the main key on all
  layouts.
- Some OS-secure sequences like `Ctrl`+`Alt`+`Del` cannot be captured or executed by user-mode apps.
- Combinations with `Esc` are not accepted. Pressing `Esc` alone closes the dialog.
- If a key is invalid, the dialog will show a message and let you try again immediately.
