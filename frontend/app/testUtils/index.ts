import 'jest-extended';
import 'jest-enzyme';
import { StaticStore } from '@app/common/static_store';
import { configure } from 'enzyme';
import PreactAdapter from 'enzyme-adapter-preact-pure';

configure({ adapter: new PreactAdapter() });

require('document-register-element/pony')(window);

beforeEach(() => {
  StaticStore.config = {
    admin_email: 'admin@remark42.com',
    admins: ['admin'],
    auth_providers: ['dev', 'google'],
    critical_score: -15,
    low_score: -5,
    edit_duration: 300,
    max_comment_size: 3000,
    max_image_size: 5000,
    positive_score: false,
    readonly_age: 100,
    version: 'jest-test',
    simple_view: false,
    anon_vote: false,
  };
});
