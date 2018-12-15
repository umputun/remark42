import withTheme from 'components/with-theme';
import Button from './button';

export default withTheme(Button);

require('./button.scss');

require('./_kind/_link/button_kind_link.scss');
require('./_kind/_text/button_kind_text.scss');

require('./_focused/button_focused.scss');
