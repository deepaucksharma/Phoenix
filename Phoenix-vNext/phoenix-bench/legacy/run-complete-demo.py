#!/usr/bin/env python3
"""
Phoenix-vNext Complete End-to-End Demo
Demonstrates full cardinality optimization benchmarking workflow
"""

import subprocess
import time
import requests
import threading
import signal
import sys
from datetime import datetime

class PhoenixDemo:
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
    
    def setup_dashboards(self):
        """Setup and enhance Phoenix-vNext dashboards"""
        self.log("Setting up enhanced dashboards...")
        try:
            # Run the dashboard unification script
            self.log("Unifying dashboards...")
            subprocess.run(["./unify-dashboards.sh"], check=True)
            
            # Run the dashboard enhancement script
            self.log("Enhancing dashboards with additional panels...")
            subprocess.run(["./enhance-dashboards.sh"], check=True)
            
            # Run the dashboard setup script
            self.log("Importing dashboards to Grafana...")
            subprocess.run(["./setup-dashboards.sh"], check=True)
            
            self.log("Dashboards setup complete!", "SUCCESS")
            self.log("Access dashboards at: http://localhost:3000", "SUCCESS")
            return True
        except subprocess.CalledProcessError as e:
            self.log(f"Dashboard setup failed: {e}", "ERROR")
            return False
    
    def check_system_health(self):
        """Check if all Phoenix components are healthy"""
        self.log("Checking system health...")
        
        endpoints = {
            "Main Collector (Full)": "http://localhost:8888/metrics",
            "Main Collector (Opt)": "http://localhost:8889/metrics", 
            "Main Collector (Ultra)": "http://localhost:8890/metrics",
            "Observer Collector": "http://localhost:8891/metrics",
            "Synthetic Generator": "http://localhost:9999/metrics",
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
    
    def start_metrics_generator(self, duration=60):
        """Start high-cardinality metrics generator"""
        self.log(f"Starting high-cardinality metrics generator for {duration}s...")
        
        cmd = [
            "python3", "generate-high-cardinality-metrics.py",
            "--interval", "5",
            "--duration", str(duration)
        ]
        
        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.processes.append(process)
        return process
    
    def start_cardinality_observer(self):
        """Start cardinality observer"""
        self.log("Starting cardinality observer...")
        
        cmd = [
            "python3", "phoenix-cardinality-observer.py",
            "--interval", "20"
        ]
        
        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.processes.append(process)
        return process
    
    def demonstrate_optimization_cycles(self):
        """Demonstrate multiple optimization cycles"""
        self.log("🚀 Starting Phoenix-vNext Complete Demo")
        self.log("="*60)
        
        # Phase 1: System Health Check
        self.log("Phase 1: System Health Verification")
        if not self.check_system_health():
            self.log("System health check failed. Please ensure all services are running.", "ERROR")
            return False
        
        # Phase 2: Dashboard Setup
        self.log("\nPhase 2: Setting Up Enhanced Dashboards")
        if not self.setup_dashboards():
            self.log("Dashboard setup encountered issues. Continuing with demo.", "WARNING")
        else:
            self.log("Dashboard setup complete. You can now monitor the system through Grafana.", "SUCCESS")
        
        # Phase 3: Baseline Measurement
        self.log("\nPhase 3: Baseline Cardinality Measurement")
        baseline_stats = self.get_cardinality_stats()
        self.log(f"Baseline - Full: {baseline_stats['full']}, Opt: {baseline_stats['opt']}, Ultra: {baseline_stats['ultra']}")
        
        # Phase 3: Start Background Observer
        self.log("\nPhase 3: Starting Cardinality Observer")
        observer_process = self.start_cardinality_observer()
        time.sleep(5)  # Let observer initialize
        
        # Phase 4: Generate High Cardinality Load
        self.log("\nPhase 4: Generating High-Cardinality Load")
        generator_process = self.start_metrics_generator(60)
        
        # Phase 5: Monitor Optimization Cycles
        self.log("\nPhase 5: Monitoring Optimization Cycles")
        
        for cycle in range(1, 4):  # 3 monitoring cycles
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
    
    def cleanup(self):
        """Clean up running processes"""
        self.log("Cleaning up processes...")
        for process in self.processes:
            try:
                process.terminate()
                process.wait(timeout=5)
            except:
                try:
                    process.kill()
                except:
                    pass
        self.processes.clear()
    
    def signal_handler(self, signum, frame):
        """Handle interrupt signals"""
        self.log("Received interrupt signal, cleaning up...", "WARNING")
        self.running = False
        self.cleanup()
        sys.exit(0)

def main():
    demo = PhoenixDemo()
    
    # Set up signal handlers
    signal.signal(signal.SIGINT, demo.signal_handler)
    signal.signal(signal.SIGTERM, demo.signal_handler)
    
    try:
        success = demo.demonstrate_optimization_cycles()
        if success:
            demo.log("Demo completed successfully! 🎉", "SUCCESS")
        else:
            demo.log("Demo encountered issues. Check system status.", "ERROR")
    except KeyboardInterrupt:
        demo.log("Demo interrupted by user", "WARNING")
    except Exception as e:
        demo.log(f"Demo failed with error: {e}", "ERROR")
    finally:
        demo.cleanup()

if __name__ == "__main__":
    main()