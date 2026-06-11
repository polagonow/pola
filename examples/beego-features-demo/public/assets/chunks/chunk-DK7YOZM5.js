// node_modules/@pola/actions/src/bridge.ts
function createAction(name) {
  return new Proxy({}, {
    get(_, method) {
      const bridge = typeof __DEPENDENCY_INJECTION__ !== "undefined" ? __DEPENDENCY_INJECTION__ : null;
      const key = `${name}.${method}`;
      const fn = bridge && bridge[key];
      if (typeof fn === "function")
        return (...args) => fn(...args);
      return () => Promise.reject(new Error(`pola/actions: ${key} not registered`));
    }
  });
}

// node_modules/@pola/actions/src/generated.ts
var Auth = createAction("Auth");

export {
  Auth
};
