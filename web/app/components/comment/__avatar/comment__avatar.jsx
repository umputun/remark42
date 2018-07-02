import { h, Component } from 'preact';


export default class Avatar extends Component {
    render() {
        const { picture } = this.props;
        return (
            <img
                className={b('comment__avatar', {}, { default: !picture })}
                src={picture || require('./comment__avatar.svg')}
                alt=""
            />
        )
    }
}
