export function replaceSelection(text: string, selection: [number, number], replacement: string): string {
  return text.substring(0, selection[0]) + replacement + text.substring(selection[1]);
}
