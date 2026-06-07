const modifierKeys = [
  ["ctrlKey", "Ctrl"],
  ["altKey", "Alt"],
  ["shiftKey", "Shift"],
  ["metaKey", "Win"],
] as const;

export function keyComboFromEvent(event: KeyboardEvent): string | null {
  const key = normalizedHotkeyKey(event);
  const modifiers = modifierKeys.filter(([property]) => event[property]).map(([, label]) => label);
  if (!key || !modifiers.length) {
    return null;
  }
  return [...modifiers, key].join("+");
}

export function normalizedHotkeyKey(event: KeyboardEvent): string | null {
  if (["Control", "Alt", "Shift", "Meta"].includes(event.key)) {
    return null;
  }
  if (event.code === "Space" || event.key === " ") {
    return "Space";
  }
  if (event.key === "Enter") {
    return "Enter";
  }
  if (/^F(?:[1-9]|1[0-2])$/.test(event.key)) {
    return event.key;
  }
  if (/^[a-z]$/i.test(event.key)) {
    return event.key.toUpperCase();
  }
  if (/^[0-9]$/.test(event.key)) {
    return event.key;
  }
  return null;
}
