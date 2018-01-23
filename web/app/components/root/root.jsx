import { Component } from 'preact';
import fetcher from 'common/fetcher';

export default class Root extends Component {
  componentDidMount() {
    // TODO: add preloader
    // TODO: all of these settings must be optional params
    fetcher
      .get('/find?url=https://radio-t.com/p/2017/12/16/podcast-576/&sort=time&format=tree')
      .then(this.loadData.bind(this));
  }

  loadData(data) {
    this.setState({ data: JSON.stringify(data) })
  }

  render() {
    const { data } = this.state;

    return data;
  }
}
