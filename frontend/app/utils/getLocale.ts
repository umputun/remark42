import { LastCommentsConfig } from '@app/common/config-types';
export function getLocale(params: { [key: string]: string } | LastCommentsConfig): string {
  return params.locale || 'en';
}
