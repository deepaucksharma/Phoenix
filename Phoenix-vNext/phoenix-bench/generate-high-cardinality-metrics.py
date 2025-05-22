#!/usr/bin/env python3
"""
High-Cardinality Metrics Generator for Phoenix-vNext Benchmarking
Generates realistic high-cardinality metrics to test optimization strategies
"""

import time
import random
import requests
import json
from datetime import datetime
import threading
import argparse

class PhoenixMetricsGenerator:
    def __init__(self, collector_url="http://localhost:4318/v1/metrics"):
        self.collector_url = collector_url
        self.running = False
        
        # Simulate realistic high-cardinality dimensions
        self.services = [f"service-{i}" for i in range(1, 21)]  # 20 services
        self.environments = ["prod", "staging", "dev", "test"]
        self.regions = ["us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"]
        self.instance_types = ["t3.micro", "t3.small", "t3.medium", "m5.large", "c5.xlarge"]
        self.endpoints = [f"/api/v{v}/endpoint-{i}" for v in [1,2,3] for i in range(1, 26)]  # 75 endpoints
        self.users = [f"user-{i:04d}" for i in range(1, 501)]  # 500 users
        self.request_types = ["GET", "POST", "PUT", "DELETE", "PATCH"]
        self.status_codes = ["200", "201", "400", "401", "403", "404", "500", "502", "503"]
        
    def generate_phoenix_metrics(self):
        """Generate Phoenix-branded high-cardinality metrics"""
        timestamp = int(time.time() * 1000)
        
        metrics = []
        
        # 1. Phoenix Request Metrics (High Cardinality)
        for _ in range(50):  # Generate 50 request metrics per batch
            service = random.choice(self.services)
            endpoint = random.choice(self.endpoints)
            method = random.choice(self.request_types)
            status = random.choice(self.status_codes)
            env = random.choice(self.environments)
            region = random.choice(self.regions)
            
            # Request count metric
            metrics.append({
                "name": "phoenix_http_requests_total",
                "type": "counter",
                "value": random.randint(1, 100),
                "timestamp": timestamp,
                "labels": {
                    "service": service,
                    "endpoint": endpoint,
                    "method": method,
                    "status_code": status,
                    "environment": env,
                    "region": region,
                    "benchmark_id": "phoenix-vnext"
                }
            })
            
            # Request duration metric
            metrics.append({
                "name": "phoenix_http_request_duration_seconds",
                "type": "histogram",
                "value": random.uniform(0.001, 2.0),
                "timestamp": timestamp,
                "labels": {
                    "service": service,
                    "endpoint": endpoint,
                    "method": method,
                    "environment": env,
                    "region": region,
                    "benchmark_id": "phoenix-vnext"
                }
            })
        
        # 2. Phoenix User Activity Metrics (Very High Cardinality)
        for _ in range(30):  # Generate 30 user activity metrics
            user = random.choice(self.users)
            service = random.choice(self.services)
            action = random.choice(["login", "logout", "purchase", "view", "search", "click"])
            
            metrics.append({
                "name": "phoenix_user_activity_total",
                "type": "counter", 
                "value": random.randint(1, 10),
                "timestamp": timestamp,
                "labels": {
                    "user_id": user,
                    "service": service,
                    "action": action,
                    "environment": random.choice(self.environments),
                    "benchmark_id": "phoenix-vnext"
                }
            })
        
        # 3. Phoenix System Metrics (Medium Cardinality)
        for _ in range(25):  # Generate 25 system metrics
            instance_type = random.choice(self.instance_types)
            service = random.choice(self.services)
            region = random.choice(self.regions)
            
            # CPU usage
            metrics.append({
                "name": "phoenix_system_cpu_usage_percent",
                "type": "gauge",
                "value": random.uniform(0, 100),
                "timestamp": timestamp,
                "labels": {
                    "instance_type": instance_type,
                    "service": service,
                    "region": region,
                    "environment": random.choice(self.environments),
                    "benchmark_id": "phoenix-vnext"
                }
            })
            
            # Memory usage
            metrics.append({
                "name": "phoenix_system_memory_usage_bytes",
                "type": "gauge",
                "value": random.randint(1000000, 8000000000),
                "timestamp": timestamp,
                "labels": {
                    "instance_type": instance_type,
                    "service": service,
                    "region": region,
                    "environment": random.choice(self.environments),
                    "benchmark_id": "phoenix-vnext"
                }
            })
        
        # 4. Phoenix Business Metrics (Custom High Cardinality)
        for _ in range(20):  # Generate 20 business metrics
            service = random.choice(self.services)
            feature = random.choice(["checkout", "search", "recommendation", "payment", "notification"])
            
            metrics.append({
                "name": "phoenix_business_feature_usage_total",
                "type": "counter",
                "value": random.randint(1, 1000),
                "timestamp": timestamp,
                "labels": {
                    "service": service,
                    "feature": feature,
                    "environment": random.choice(self.environments),
                    "region": random.choice(self.regions),
                    "benchmark_id": "phoenix-vnext"
                }
            })
        
        return metrics
    
    def send_metrics_to_collector(self, metrics):
        """Send metrics to OpenTelemetry collector via OTLP HTTP"""
        # Convert to OTLP format
        otlp_payload = {
            "resourceMetrics": [{
                "resource": {
                    "attributes": [{
                        "key": "service.name",
                        "value": {"stringValue": "phoenix-metrics-generator"}
                    }, {
                        "key": "service.version", 
                        "value": {"stringValue": "1.0.0"}
                    }, {
                        "key": "telemetry.sdk.name",
                        "value": {"stringValue": "phoenix-vnext"}
                    }]
                },
                "scopeMetrics": [{
                    "scope": {
                        "name": "phoenix.benchmark",
                        "version": "1.0.0"
                    },
                    "metrics": []
                }]
            }]
        }
        
        # Convert metrics to OTLP format
        for metric in metrics:
            otlp_metric = {
                "name": metric["name"],
                "description": f"Phoenix benchmark metric: {metric['name']}",
                "unit": "1" if metric["type"] == "counter" else ""
            }
            
            # Add data points based on metric type
            if metric["type"] == "counter":
                otlp_metric["sum"] = {
                    "dataPoints": [{
                        "timeUnixNano": metric["timestamp"] * 1000000,
                        "asInt": int(metric["value"]),
                        "attributes": [
                            {"key": k, "value": {"stringValue": str(v)}} 
                            for k, v in metric["labels"].items()
                        ]
                    }],
                    "aggregationTemporality": 2,  # CUMULATIVE
                    "isMonotonic": True
                }
            elif metric["type"] == "gauge":
                otlp_metric["gauge"] = {
                    "dataPoints": [{
                        "timeUnixNano": metric["timestamp"] * 1000000,
                        "asDouble": float(metric["value"]),
                        "attributes": [
                            {"key": k, "value": {"stringValue": str(v)}} 
                            for k, v in metric["labels"].items()
                        ]
                    }]
                }
            
            otlp_payload["resourceMetrics"][0]["scopeMetrics"][0]["metrics"].append(otlp_metric)
        
        try:
            response = requests.post(
                self.collector_url,
                json=otlp_payload,
                headers={"Content-Type": "application/json"},
                timeout=5
            )
            
            if response.status_code == 200:
                print(f"✅ Sent {len(metrics)} metrics to collector")
            else:
                print(f"❌ Failed to send metrics: {response.status_code} - {response.text}")
                
        except Exception as e:
            print(f"❌ Error sending metrics: {e}")
    
    def run_generator(self, interval=10, duration=None):
        """Run the metrics generator"""
        self.running = True
        start_time = time.time()
        
        print(f"🚀 Starting Phoenix high-cardinality metrics generator")
        print(f"📊 Target: {self.collector_url}")
        print(f"⏱️  Interval: {interval} seconds")
        if duration:
            print(f"⏳ Duration: {duration} seconds")
        print("="*50)
        
        iteration = 0
        while self.running:
            if duration and (time.time() - start_time) >= duration:
                break
                
            iteration += 1
            print(f"🔄 Iteration {iteration} - {datetime.now().strftime('%H:%M:%S')}")
            
            # Generate high-cardinality metrics
            metrics = self.generate_phoenix_metrics()
            
            # Calculate expected cardinality
            unique_series = len(set(
                f"{m['name']}_{hash(tuple(sorted(m['labels'].items())))}" 
                for m in metrics
            ))
            print(f"📈 Generated {len(metrics)} metrics with ~{unique_series} unique time series")
            
            # Send to collector
            self.send_metrics_to_collector(metrics)
            
            time.sleep(interval)
        
        print("🛑 Metrics generator stopped")
    
    def stop(self):
        """Stop the metrics generator"""
        self.running = False

def main():
    parser = argparse.ArgumentParser(description="Phoenix-vNext High-Cardinality Metrics Generator")
    parser.add_argument("--url", default="http://localhost:4318/v1/metrics", 
                       help="OTLP HTTP endpoint URL")
    parser.add_argument("--interval", type=int, default=10, 
                       help="Metrics generation interval in seconds")
    parser.add_argument("--duration", type=int, 
                       help="Total duration to run (seconds)")
    
    args = parser.parse_args()
    
    generator = PhoenixMetricsGenerator(args.url)
    
    try:
        generator.run_generator(args.interval, args.duration)
    except KeyboardInterrupt:
        print("\n🛑 Stopping generator...")
        generator.stop()

if __name__ == "__main__":
    main()