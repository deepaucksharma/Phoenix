const endpoints = [
  { name: 'Main collector', url: 'http://localhost:13133/health' },
  { name: 'Control actuator', url: 'http://localhost:8081/metrics' },
  { name: 'Anomaly detector', url: 'http://localhost:8082/health' },
  { name: 'Benchmark controller', url: 'http://localhost:8083/health' },
];

(async () => {
  let allHealthy = true;
  for (const { name, url } of endpoints) {
    try {
      const res = await fetch(url);
      if (res.ok) {
        console.log(`${name}: OK`);
      } else {
        console.log(`${name}: FAIL (status ${res.status})`);
        allHealthy = false;
      }
    } catch (err) {
      console.log(`${name}: DOWN`);
      allHealthy = false;
    }
  }
  if (!allHealthy) {
    process.exitCode = 1;
  }
})();
