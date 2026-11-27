/**
 * The seconds left until a moment, as text such as `42s`, counting down once a second. When it
 * reaches zero it stops and calls onTimePassed, once per moment it was given.
 *
 * @param {() => void} onTimePassed
 */
export function countdown(onTimePassed) {
  const text = document.createTextNode('');
  /** @type {number | undefined} */
  let timestamp;
  /** @type {ReturnType<typeof setInterval> | undefined} */
  let timer;

  function tick() {
    const left = Math.ceil(Math.max(0, ((timestamp ?? 0) - Date.now()) / 1000));
    text.data = `${left}s`;
    if (left === 0) {
      stop();
      onTimePassed();
    }
  }

  function stop() {
    clearInterval(timer);
    timer = undefined;
  }

  return {
    text,
    /**
     * Counts down to this moment, starting again when it differs from the one before.
     *
     * @param {number} moment  milliseconds since the epoch
     */
    countTo(moment) {
      if (moment === timestamp) {
        return;
      }
      stop();
      timestamp = moment;
      timer = setInterval(tick, 1000);
      tick();
    },
    /** stops counting; the next countTo starts afresh, even to the same moment */
    stop() {
      stop();
      timestamp = undefined;
    },
  };
}
