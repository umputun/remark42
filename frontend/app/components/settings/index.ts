import withTheme from '../../components/with-theme';
import Settings from './settings';

export default withTheme(Settings);

require('./settings.scss');

require('./__action/settings__action.scss');
require('./__section/settings__section.scss');
require('./__list/settings__list.scss');
require('./__invisible/settings__invisible.scss');
require('./__dimmed/settings__dimmed.scss');
require('./__username/settings__username.scss');
require('./__user-id/settings__user-id.scss');
require('./_theme/_dark/settings_theme_dark.scss');
require('./_theme/_light/settings_theme_light.scss');
