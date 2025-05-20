# Phoenix Dashboards

This directory contains Grafana dashboard definitions for monitoring the Phoenix (SA-OMF) system.

## Available Dashboards

| Dashboard | Description | Tags |
|-----------|-------------|------|
| [autonomy-pulse.json](./autonomy-pulse.json) | Monitor autonomy level and self-adaptive behavior | sa-omf, phoenix |

## Usage

These dashboards can be imported into Grafana using the JSON file or by referencing the file URL.

### Local Development

1. Start Grafana locally using the provided docker-compose file:
   ```
   docker-compose -f deploy/compose/full/docker-compose.yaml up -d
   ```

2. Access Grafana at http://localhost:3000
   - Default credentials: admin/admin

3. Import the dashboard(s) from the JSON files in this directory

### Production Deployment

When deploying to production, consider using the Grafana provisioning feature to automatically 
load these dashboards. See the [Grafana documentation](https://grafana.com/docs/grafana/latest/administration/provisioning/#dashboards) for details.

