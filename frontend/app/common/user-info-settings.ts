import parseQuery from 'utils/parseQuery';

import type { UserInfo } from './types';

export const userInfo: UserInfo = parseQuery();

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const isDefaultPicture = ((userInfo.isDefaultPicture as any) as string) !== '1';
export const id = userInfo.id;
export const name = userInfo.name;
export const picture = userInfo.picture;
