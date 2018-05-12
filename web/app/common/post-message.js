export default msg => window.parent.postMessage(JSON.stringify(msg), '*');
