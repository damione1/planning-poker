const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = path.resolve(__dirname, '../..');

test('saved room permissions initialize after deferred store registration', () => {
  const template = fs.readFileSync(path.join(root, 'web/templates/room.templ'), 'utf8');
  const inline = template.match(/<script type="text\/javascript">([\s\S]*?)<\/script>/)[1];
  const components = fs.readFileSync(path.join(root, 'web/static/js/alpine-components.js'), 'utf8');
  const savedConfig = {
    permissions: {
      allow_all_reveal: false,
      allow_all_reset: false,
      allow_all_new_round: false,
      allow_change_vote_after_reveal: true,
      auto_reveal: true,
    },
  };
  const listeners = new Map();
  const stores = new Map();
  const context = vm.createContext({
    document: {
      addEventListener(name, handler) {
        if (!listeners.has(name)) listeners.set(name, []);
        listeners.get(name).push(handler);
      },
      getElementById(id) {
        assert.equal(id, 'room-config-data');
        return { textContent: JSON.stringify(savedConfig) };
      },
    },
    Alpine: {
      store(name, value) {
        if (value !== undefined) {
          stores.set(name, value);
          // Alpine invokes a registered store's init method without arguments.
          if (typeof value.init === 'function') value.init();
        }
        return stores.get(name);
      },
      data() {},
    },
    console: { log() {} },
  });

  // The room's inline script is parsed before deferred alpine-components.js.
  vm.runInContext(inline, context);
  vm.runInContext(components, context);
  const dispatch = (name) => (listeners.get(name) || []).forEach((handler) => handler());
  dispatch('alpine:init');
  assert.notDeepEqual(JSON.parse(JSON.stringify(stores.get('roomSettings').config)), savedConfig);
  dispatch('alpine:initialized');
  assert.deepEqual(JSON.parse(JSON.stringify(stores.get('roomSettings').config)), savedConfig);
});
