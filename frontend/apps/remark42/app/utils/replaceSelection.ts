export function replaceSelection(text: string, selection: [number, number], replacement: string): string {
  return text.substr(0, selection[0]) + replacement + text.substr(selection[1]);
}
