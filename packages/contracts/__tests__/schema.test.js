const schema = require('../schemas/control_signal.json');

test('schema has mode enum', () => {
  expect(schema.properties).toHaveProperty('mode');
  expect(schema.properties.mode.enum).toContain('balanced');
});
