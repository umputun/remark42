export function getLocale(params: { locale?: string; [key: string]: unknown }): string {
  return params.locale || 'en';
}
