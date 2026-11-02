import type { Config } from './types';

interface StaticStoreType {
  config: Config;
  /** how far the client clock is ahead of the server, in milliseconds. set by the fetcher,
   * read wherever a server timestamp meets the local clock, such as the comment edit deadline */
  serverClientTimeDiffMs?: number;
}

/**
 * Represent store of values that and will not change, or doesn't need reactivity
 *
 * Initialized once at webpack's entry points (i.e remark.tsx)
 */
export const StaticStore: StaticStoreType = {
  config: {
    version: '',
    edit_duration: 5000,
    admin_edit: false,
    max_comment_size: 5000,
    admins: [],
    admin_email: '',
    auth_providers: [],
    critical_score: 0,
    low_score: 0,
    positive_score: false,
    readonly_age: 0,
    max_image_size: 0,
    simple_view: false,
    anon_vote: false,
    email_notifications: false,
    telegram_notifications: false,
    emoji_enabled: false,
  },
};
