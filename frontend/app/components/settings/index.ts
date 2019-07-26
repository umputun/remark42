import './settings.scss';

import './__action/settings__action.scss';
import './__section/settings__section.scss';
import './__list/settings__list.scss';
import './__invisible/settings__invisible.scss';
import './__dimmed/settings__dimmed.scss';
import './__username/settings__username.scss';
import './__user-id/settings__user-id.scss';
import './_theme/_dark/settings_theme_dark.scss';
import './_theme/_light/settings_theme_light.scss';

import withTheme from '../../components/with-theme';
import Settings from './settings';

export default withTheme(Settings);
