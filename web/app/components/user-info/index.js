import { userComments, isLoadingUserComments } from './user-info.reducers';

export const userInfoReducers = { userComments, isLoadingUserComments };

export { default } from './user-info';

require('./user-info.scss');
require('./__id/user-info__id.scss');
require('./__preloader/user-info__preloader.scss');
require('./__title/user-info__title.scss');
require('./__avatar/user-info__avatar.scss');
