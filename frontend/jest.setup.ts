import { configure } from 'enzyme';
import PreactAdapter from 'enzyme-adapter-preact-pure';

configure({ adapter: new PreactAdapter() });
