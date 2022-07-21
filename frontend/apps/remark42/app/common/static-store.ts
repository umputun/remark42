import { Config } from './types';

interface StaticStoreType {
  config: Config;
  /** used in fetcher, fer example to set comment edit timeout */
  serverClientTimeDiff?: number;
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
    telegram_bot_username: '',
    emoji_enabled: false,
  },
};
