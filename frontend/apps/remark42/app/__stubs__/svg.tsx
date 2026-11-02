// the real build resolves an svg import through webpack's public path, which is absolute at
// runtime. a relative stub hides a component that prefixes the instance origin onto an already
// absolute asset url: the concatenation looks right here and ships a doubled url
const SvgrUrl = 'http://localhost:8080/web/image.svg';

export default SvgrUrl;
