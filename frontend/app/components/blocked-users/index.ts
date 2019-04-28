import withTheme from '../../components/with-theme';
import BlockedUsers from './blocked-users';

export default withTheme(BlockedUsers);

require('./blocked-users.scss');

require('./__action/blocked-users__action.scss');
require('./__list/blocked-users__list.scss');

require('./__list-item/blocked-users__list-item.scss');
require('./__list-item/_view/_invisible/blocked-users__list-item_view_invisible.scss');

require('./__username/blocked-users__username.scss');

require('./_theme/_dark/blocked-users_theme_dark.scss');
require('./_theme/_light/blocked-users_theme_light.scss');
