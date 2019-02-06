export default function debounce(func, wait) {
  let timeout;
  return function() {
    const laterCall = () => func.apply(this, arguments);
    clearTimeout(timeout);
    timeout = setTimeout(laterCall, wait);
  };
}
