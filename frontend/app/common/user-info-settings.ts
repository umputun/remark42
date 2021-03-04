import { parseQuery } from 'utils/parseQuery';

import type { UserInfo } from './types';

export const userInfo: UserInfo = parseQuery();

export const id = userInfo.id;
export const name = userInfo.name;
export const picture = userInfo.picture;
