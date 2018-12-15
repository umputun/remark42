import withTheme from 'components/with-theme';
import AuthPanel from './auth-panel';

export default withTheme(AuthPanel);

require('./auth-panel.scss');

require('./__column/auth-panel__column.scss');
require('./__pseudo-link/auth-panel__pseudo-link.scss');
require('./__select/auth-panel__select.scss');
require('./__select-label/auth-panel__select-label.scss');
require('./__sort/auth-panel__sort.scss');

require('./__user-id/auth-panel__user-id.scss');
require('./__sign-out/auth-panel__sign-out.scss');

require('./_logged-in/auth-panel_logged-in.scss');
