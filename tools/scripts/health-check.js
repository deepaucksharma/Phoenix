#!/usr/bin/env node
const http = require('http');

const endpoints = [
  { name: 'Main Collector', url: 'http://localhost:13133/health' },
  { name: 'Observer Collector', url: 'http://localhost:13134/health' }
];

function check({ name, url }) {
  return new Promise(resolve => {
    const req = http.get(url, res => {
      if (res.statusCode === 200) {
        console.log(`✓ ${name} healthy`);
        resolve(true);
      } else {
        console.error(`✗ ${name} status ${res.statusCode}`);
        resolve(false);
      }
    });
    req.on('error', () => {
      console.error(`✗ ${name} unreachable`);
      resolve(false);
    });
    req.setTimeout(5000, () => {
      console.error(`✗ ${name} timeout`);
      req.destroy();
      resolve(false);
    });
  });
}

(async () => {
  let ok = true;
  for (const ep of endpoints) {
    if (!(await check(ep))) ok = false;
  }
  if (!ok) process.exitCode = 1;
})();
