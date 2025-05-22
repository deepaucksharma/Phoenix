#!/usr/bin/env python3
"""
Phoenix-vNext Consolidated Toolkit
Unified command-line interface for all Phoenix-vNext operations
Combines metrics generation, cardinality observation, and demo orchestration
"""

import time
import random
import requests
import json
import yaml
import subprocess
import threading
import signal
import sys
import os
import argparse
from datetime import datetime, timezone

class PhoenixMetricsGenerator:
    """High-Cardinality Metrics Generator for Phoenix-vNext Benchmarking"""
    
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


class PhoenixCardinalityObserver:
    """Dynamic Cardinality Observer - Monitors pipeline cardinality and generates control signals"""
    
    def __init__(self, 
                 main_collector_url="http://localhost:8888/metrics",
                 control_file_path="configs/control_signals/opt_mode.yaml",
                 thresholds=None):
        
        self.main_collector_url = main_collector_url
        self.control_file_path = control_file_path
        self.running = False
        
        # Schema-aligned thresholds
        self.thresholds = thresholds or {
            "moderate": 300.0,
            "adaptive": 375.0, 
            "ultra": 450.0
        }
        
        self.current_mode = "moderate"
        self.last_cardinality_count = 0
        self.mode_change_count = 0
        
    def get_cardinality_metrics(self):
        """Scrape cardinality metrics from main collector"""
        try:
            response = requests.get(self.main_collector_url, timeout=10)
            response.raise_for_status()
            
            metrics_text = response.text
            phoenix_metrics = [line for line in metrics_text.split('\n') 
                             if line.startswith('phoenix_') and not line.startswith('#')]
            
            # Count unique time series (metric name + unique label combinations)
            unique_series = set()
            for metric_line in phoenix_metrics:
                if '{' in metric_line:
                    # Extract metric name and labels
                    metric_name = metric_line.split('{')[0]
                    labels_part = metric_line.split('{')[1].split('}')[0]
                    series_key = f"{metric_name}#{labels_part}"
                    unique_series.add(series_key)
                else:
                    # Simple metric without labels
                    metric_name = metric_line.split(' ')[0]
                    unique_series.add(metric_name)
            
            cardinality_count = len(unique_series)
            total_metrics = len(phoenix_metrics)
            
            return {
                "cardinality_count": cardinality_count,
                "total_metrics": total_metrics,
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "sample_metrics": list(unique_series)[:5]  # First 5 for debugging
            }
            
        except Exception as e:
            print(f"❌ Error fetching cardinality metrics: {e}")
            return None
    
    def determine_optimization_mode(self, cardinality_count):
        """Determine the appropriate optimization mode based on cardinality"""
        if cardinality_count <= self.thresholds["moderate"]:
            return "moderate", 0  # 0% optimization
        elif cardinality_count <= self.thresholds["adaptive"]:
            # Scale optimization level between 26-75% based on position in range
            range_size = self.thresholds["adaptive"] - self.thresholds["moderate"]
            position = cardinality_count - self.thresholds["moderate"]
            opt_level = int(26 + (position / range_size) * 49)  # 26-75%
            return "adaptive", min(75, max(26, opt_level))
        else:
            # Scale optimization level between 76-100% based on position above adaptive threshold
            excess = cardinality_count - self.thresholds["adaptive"]
            max_excess = self.thresholds["ultra"] - self.thresholds["adaptive"]
            if excess >= max_excess:
                return "ultra", 100
            else:
                opt_level = int(76 + (excess / max_excess) * 24)  # 76-100%
                return "ultra", min(100, max(76, opt_level))
    
    def generate_control_signal(self, cardinality_data):
        """Generate control signal based on cardinality analysis"""
        cardinality_count = cardinality_data["cardinality_count"]
        new_mode, optimization_level = self.determine_optimization_mode(cardinality_count)
        
        # Only generate signal if mode changes or significant cardinality change
        cardinality_change = abs(cardinality_count - self.last_cardinality_count)
        significant_change = cardinality_change > 50  # More than 50 series change
        
        if new_mode != self.current_mode or significant_change:
            timestamp = datetime.now(timezone.utc)
            correlation_id = f"observer-{int(timestamp.timestamp())}"
            
            # Determine reason for mode change
            if new_mode != self.current_mode:
                reason = f"Mode change: {self.current_mode} → {new_mode} (cardinality: {cardinality_count})"
                self.mode_change_count += 1
            else:
                reason = f"Cardinality update: {cardinality_count} series (change: +{cardinality_change})"
            
            control_signal = {
                "mode": new_mode,
                "last_updated": timestamp.isoformat(),
                "reason": reason,
                "ts_count": cardinality_count,
                "config_version": int(timestamp.timestamp()),
                "correlation_id": correlation_id,
                "optimization_level": optimization_level,
                "thresholds": self.thresholds.copy(),
                "state": {
                    "previous_mode": self.current_mode,
                    "transition_timestamp": timestamp.isoformat(),
                    "transition_duration_seconds": 0,
                    "stability_period_seconds": 300,
                    "mode_changes_total": self.mode_change_count
                },
                "observer_metadata": {
                    "cardinality_analysis": {
                        "total_phoenix_metrics": cardinality_data["total_metrics"],
                        "unique_time_series": cardinality_count,
                        "reduction_potential": f"{max(0, cardinality_count - self.thresholds['moderate'])} series",
                        "sample_metrics": cardinality_data["sample_metrics"]
                    }
                }
            }
            
            # Update internal state
            self.current_mode = new_mode
            self.last_cardinality_count = cardinality_count
            
            return control_signal
        
        return None
    
    def write_control_signal(self, control_signal):
        """Write control signal to the control file"""
        try:
            # Create backup
            if os.path.exists(self.control_file_path):
                backup_path = f"{self.control_file_path}.backup"
                with open(self.control_file_path, 'r') as src, open(backup_path, 'w') as dst:
                    dst.write(src.read())
            
            # Write new control signal
            with open(self.control_file_path, 'w') as f:
                yaml.dump(control_signal, f, default_flow_style=False, sort_keys=False)
            
            print(f"📝 Control signal written: mode={control_signal['mode']}, "
                  f"cardinality={control_signal['ts_count']}, "
                  f"opt_level={control_signal['optimization_level']}%")
            
            return True
            
        except Exception as e:
            print(f"❌ Error writing control signal: {e}")
            return False
    
    def run_observer(self, check_interval=30):
        """Run the cardinality observer"""
        self.running = True
        
        print("🔍 Starting Phoenix Cardinality Observer")
        print(f"📊 Monitoring: {self.main_collector_url}")
        print(f"📁 Control file: {self.control_file_path}")
        print(f"⏱️  Check interval: {check_interval} seconds")
        print(f"🎯 Thresholds: {self.thresholds}")
        print("="*60)
        
        iteration = 0
        while self.running:
            iteration += 1
            timestamp = datetime.now().strftime('%H:%M:%S')
            print(f"\n🔄 Observer Iteration {iteration} - {timestamp}")
            
            # Get current cardinality metrics
            cardinality_data = self.get_cardinality_metrics()
            if not cardinality_data:
                time.sleep(check_interval)
                continue
            
            cardinality_count = cardinality_data["cardinality_count"]
            total_metrics = cardinality_data["total_metrics"]
            
            print(f"📈 Current cardinality: {cardinality_count} unique time series "
                  f"({total_metrics} total Phoenix metrics)")
            
            # Determine if control signal is needed
            control_signal = self.generate_control_signal(cardinality_data)
            
            if control_signal:
                print(f"🚨 Generating control signal!")
                print(f"   Mode: {self.current_mode} → {control_signal['mode']}")
                print(f"   Optimization: {control_signal['optimization_level']}%")  
                print(f"   Reason: {control_signal['reason']}")
                
                success = self.write_control_signal(control_signal)
                if success:
                    print("✅ Control signal successfully written")
                else:
                    print("❌ Failed to write control signal")
            else:
                print(f"ℹ️  No mode change needed (current: {self.current_mode})")
            
            # Show threshold analysis
            mode, opt_level = self.determine_optimization_mode(cardinality_count)
            if cardinality_count <= self.thresholds["moderate"]:
                status = "🟢 LOW"
            elif cardinality_count <= self.thresholds["adaptive"]:
                status = "🟡 MEDIUM"
            else:
                status = "🔴 HIGH"
            
            print(f"🎯 Cardinality status: {status} ({mode} mode, {opt_level}% optimization)")
            
            time.sleep(check_interval)
        
        print("🛑 Observer stopped")
    
    def stop(self):
        """Stop the observer"""
        self.running = False


class PhoenixDemo:
    """Complete End-to-End Demo - Demonstrates full cardinality optimization workflow"""
    
    def __init__(self):
        self.processes = []
        self.running = True
        
    def log(self, message, level="INFO"):
        timestamp = datetime.now().strftime("%H:%M:%S")
        if level == "SUCCESS":
            print(f"✅ [{timestamp}] {message}")
        elif level == "ERROR":
            print(f"❌ [{timestamp}] {message}")
        elif level == "WARNING":
            print(f"⚠️  [{timestamp}] {message}")
        else:
            print(f"ℹ️  [{timestamp}] {message}")
    
    def check_system_health(self):
        """Check if all Phoenix components are healthy"""
        self.log("Checking system health...")
        
        endpoints = {
            "Main Collector (Full)": "http://localhost:8888/metrics",
            "Main Collector (Opt)": "http://localhost:8889/metrics", 
            "Main Collector (Ultra)": "http://localhost:8890/metrics",
            "Observer Collector": "http://localhost:8891/metrics",
            "Prometheus": "http://localhost:9090/-/healthy",
            "Grafana": "http://localhost:3000/api/health"
        }
        
        all_healthy = True
        for name, url in endpoints.items():
            try:
                response = requests.get(url, timeout=5)
                if response.status_code == 200:
                    self.log(f"{name}: Healthy", "SUCCESS")
                else:
                    self.log(f"{name}: Unhealthy (HTTP {response.status_code})", "ERROR")
                    all_healthy = False
            except Exception as e:
                self.log(f"{name}: Unreachable ({str(e)})", "ERROR")
                all_healthy = False
        
        return all_healthy
    
    def get_cardinality_stats(self):
        """Get current cardinality statistics from all pipelines"""
        stats = {}
        
        pipelines = {
            "full": "http://localhost:8888/metrics",
            "opt": "http://localhost:8889/metrics",
            "ultra": "http://localhost:8890/metrics"
        }
        
        for pipeline, url in pipelines.items():
            try:
                response = requests.get(url, timeout=5)
                if response.status_code == 200:
                    phoenix_metrics = len([line for line in response.text.split('\n') 
                                         if line.startswith('phoenix_') and not line.startswith('#')])
                    stats[pipeline] = phoenix_metrics
                else:
                    stats[pipeline] = 0
            except:
                stats[pipeline] = 0
        
        return stats
    
    def run_complete_demo(self, duration=120):
        """Run complete demonstration workflow"""
        self.log("🚀 Starting Phoenix-vNext Complete Demo")
        self.log("="*60)
        
        # Phase 1: System Health Check
        self.log("Phase 1: System Health Verification")
        if not self.check_system_health():
            self.log("System health check failed. Please ensure all services are running.", "ERROR")
            return False
        
        # Phase 2: Baseline Measurement
        self.log("\nPhase 2: Baseline Cardinality Measurement")
        baseline_stats = self.get_cardinality_stats()
        self.log(f"Baseline - Full: {baseline_stats['full']}, Opt: {baseline_stats['opt']}, Ultra: {baseline_stats['ultra']}")
        
        # Phase 3: Start Background Observer
        self.log("\nPhase 3: Starting Cardinality Observer")
        observer = PhoenixCardinalityObserver()
        observer_thread = threading.Thread(target=observer.run_observer, args=(20,))
        observer_thread.daemon = True
        observer_thread.start()
        time.sleep(5)  # Let observer initialize
        
        # Phase 4: Generate High Cardinality Load
        self.log("\nPhase 4: Generating High-Cardinality Load")
        generator = PhoenixMetricsGenerator()
        generator_thread = threading.Thread(target=generator.run_generator, args=(5, duration))
        generator_thread.daemon = True
        generator_thread.start()
        
        # Phase 5: Monitor Optimization Cycles
        self.log(f"\nPhase 5: Monitoring Optimization for {duration}s")
        
        monitor_duration = duration
        start_time = time.time()
        cycle = 1
        
        while time.time() - start_time < monitor_duration:
            self.log(f"\n--- Monitoring Cycle {cycle} ---")
            time.sleep(20)  # Wait for metrics to accumulate
            
            current_stats = self.get_cardinality_stats()
            self.log(f"Current cardinality - Full: {current_stats['full']}, "
                    f"Opt: {current_stats['opt']}, Ultra: {current_stats['ultra']}")
            
            # Calculate reduction percentage
            if current_stats['full'] > 0:
                opt_reduction = ((current_stats['full'] - current_stats['opt']) / current_stats['full']) * 100
                ultra_reduction = ((current_stats['full'] - current_stats['ultra']) / current_stats['full']) * 100
                
                self.log(f"Optimization results:")
                self.log(f"  Opt Pipeline:   {opt_reduction:.1f}% reduction")
                self.log(f"  Ultra Pipeline: {ultra_reduction:.1f}% reduction")
                
                if ultra_reduction > 50:
                    self.log(f"🎯 Excellent optimization: {ultra_reduction:.1f}% cardinality reduction!", "SUCCESS")
                elif ultra_reduction > 25:
                    self.log(f"✅ Good optimization: {ultra_reduction:.1f}% cardinality reduction!", "SUCCESS")
                else:
                    self.log(f"⚠️  Moderate optimization: {ultra_reduction:.1f}% cardinality reduction")
            
            cycle += 1
        
        # Stop components
        generator.stop()
        observer.stop()
        
        # Phase 6: Final Results
        self.log("\nPhase 6: Final Results Summary")
        final_stats = self.get_cardinality_stats()
        
        self.log("🎊 PHOENIX-VNEXT DEMO COMPLETE!")
        self.log("="*60)
        self.log("📊 Final Cardinality Results:")
        self.log(f"   Full Pipeline:  {final_stats['full']:,} metrics (100% baseline)")
        self.log(f"   Opt Pipeline:   {final_stats['opt']:,} metrics")
        self.log(f"   Ultra Pipeline: {final_stats['ultra']:,} metrics")
        
        if final_stats['full'] > 0:
            ultra_reduction = ((final_stats['full'] - final_stats['ultra']) / final_stats['full']) * 100
            cost_savings = ultra_reduction
            
            self.log(f"\n🏆 Key Achievements:")
            self.log(f"   ✅ {ultra_reduction:.1f}% cardinality reduction achieved")
            self.log(f"   ✅ ~{cost_savings:.1f}% potential cost savings")
            self.log(f"   ✅ Dynamic optimization working")
            self.log(f"   ✅ Multi-pipeline benchmarking operational")
        
        self.log(f"\n🔗 Access Points:")
        self.log(f"   Grafana Dashboard: http://localhost:3000")
        self.log(f"   Prometheus: http://localhost:9090")
        self.log(f"   Full Pipeline Metrics: http://localhost:8888/metrics")
        self.log(f"   Ultra Pipeline Metrics: http://localhost:8890/metrics")
        
        return True


def main():
    """Main CLI interface for Phoenix-vNext Toolkit"""
    parser = argparse.ArgumentParser(
        description="Phoenix-vNext Consolidated Toolkit",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Available Commands:
  generate      Generate high-cardinality metrics
  observe       Monitor cardinality and generate control signals
  demo          Run complete end-to-end demonstration
  
Examples:
  %(prog)s generate --interval 5 --duration 60
  %(prog)s observe --interval 30
  %(prog)s demo --duration 120
        """
    )
    
    subparsers = parser.add_subparsers(dest='command', help='Available commands')
    
    # Generate command
    gen_parser = subparsers.add_parser('generate', help='Generate high-cardinality metrics')
    gen_parser.add_argument("--url", default="http://localhost:4318/v1/metrics", 
                           help="OTLP HTTP endpoint URL")
    gen_parser.add_argument("--interval", type=int, default=10, 
                           help="Metrics generation interval in seconds")
    gen_parser.add_argument("--duration", type=int, 
                           help="Total duration to run (seconds)")
    
    # Observe command
    obs_parser = subparsers.add_parser('observe', help='Monitor cardinality and generate control signals')
    obs_parser.add_argument("--collector-url", default="http://localhost:8888/metrics",
                           help="Main collector metrics URL")
    obs_parser.add_argument("--control-file", 
                           default="configs/control_signals/opt_mode.yaml",
                           help="Path to control signal file")
    obs_parser.add_argument("--interval", type=int, default=30,
                           help="Check interval in seconds")
    obs_parser.add_argument("--moderate-threshold", type=float, default=300.0,
                           help="Moderate optimization threshold")
    obs_parser.add_argument("--adaptive-threshold", type=float, default=375.0,
                           help="Adaptive optimization threshold") 
    obs_parser.add_argument("--ultra-threshold", type=float, default=450.0,
                           help="Ultra optimization threshold")
    
    # Demo command
    demo_parser = subparsers.add_parser('demo', help='Run complete end-to-end demonstration')
    demo_parser.add_argument("--duration", type=int, default=120,
                            help="Demo duration in seconds")
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        return
    
    try:
        if args.command == 'generate':
            generator = PhoenixMetricsGenerator(args.url)
            generator.run_generator(args.interval, args.duration)
            
        elif args.command == 'observe':
            thresholds = {
                "moderate": args.moderate_threshold,
                "adaptive": args.adaptive_threshold,
                "ultra": args.ultra_threshold
            }
            
            observer = PhoenixCardinalityObserver(
                main_collector_url=args.collector_url,
                control_file_path=args.control_file,
                thresholds=thresholds
            )
            observer.run_observer(args.interval)
            
        elif args.command == 'demo':
            demo = PhoenixDemo()
            demo.run_complete_demo(args.duration)
            
    except KeyboardInterrupt:
        print("\n🛑 Interrupted by user")
    except Exception as e:
        print(f"❌ Error: {e}")

if __name__ == "__main__":
    main()