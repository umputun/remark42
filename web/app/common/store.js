let _instance;

class Store {
  constructor() {
    if (_instance) return _instance;

    this.data = {};

    _instance = this;

    return _instance;
  }

  set(obj) {
    this.data = {
      ...this.data,
      ...obj,
    };
  }

  get(key) {
    return this.data[key];
  }
}

export default new Store();
