export default function capitalizeFirstLetter(str: string): string {
  return `${str.charAt(0).toLocaleUpperCase()}${str.slice(1)}`;
}
