import { UserInfo, Theme } from './types';

export interface CounterConfig {
  site_id: string;
  url?: string;
}

export type UserInfoConfig = UserInfo;

export interface CommentsConfig {
  site_id: string;
  url?: string;
  max_shown_comments?: number;
  theme?: Theme;
  page_title?: string;
}

export interface LastCommentsConfig {
  site_id: string;
  max_last_comments: number;
}
