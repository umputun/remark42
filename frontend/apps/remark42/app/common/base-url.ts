/**
 * Validating the host an integrator configured the widget with.
 *
 * Kept apart from constants.config.ts so the rules can be tested against arbitrary input: that module
 * reads window.remark_config and window.location at import time, and the jest suite has to run in a
 * jsdom pinned to an https url just to reach one of the branches.
 *
 * This value comes from the host page, so it is attacker-controlled wherever the page is. A host of
 * `javascript:...` would otherwise be loaded as the widget's own origin, which is why the protocol
 * is checked instead of only parsed.
 */
export function resolveBaseUrl(host: string | undefined, pageProtocol: string): string {
  if (!host) {
    throw new Error(`Remark42: remark_config.host wasn't configured.`);
  }

  try {
    const { protocol } = new URL(host);

    // an http widget on an https page is blocked by the browser as mixed content, and an https
    // widget on an http page works. neither is this function's to refuse, so it says so and
    // returns: refusing here would take the widget down over an arrangement that may be fine
    if (protocol !== pageProtocol) {
      console.error('Remark42: Protocol mismatch.');
    }

    // compared exactly, not by prefix: `startsWith('http')` admits any scheme whose name begins
    // with those four letters, and whatever survives this becomes the widget's own origin
    if (protocol !== 'http:' && protocol !== 'https:') {
      console.error('Remark42: Wrong protocol in host URL.');
      throw new Error();
    }
  } catch (e) {
    throw new Error('Remark42: Invalid host URL.');
  }

  return host;
}
