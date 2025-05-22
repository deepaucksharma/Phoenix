#!/usr/bin/env python3
"""
Phoenix-vNext Dynamic Cardinality Observer
Monitors pipeline cardinality and generates dynamic control signals
"""

import time
import requests
import yaml
import json
from datetime import datetime, timezone
import threading
import argparse
import os

class PhoenixCardinalityObserver:
    def __init__(self, 
                 main_collector_url="http://localhost:8888/metrics",
                 control_file_path="/Users/deepaksharma/Desktop/src_main/Phoenix-vNext/phoenix-bench/configs/control_signals/opt_mode.yaml",
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

def main():
    parser = argparse.ArgumentParser(description="Phoenix-vNext Cardinality Observer")
    parser.add_argument("--collector-url", default="http://localhost:8888/metrics",
                       help="Main collector metrics URL")
    parser.add_argument("--control-file", 
                       default="/Users/deepaksharma/Desktop/src_main/Phoenix-vNext/phoenix-bench/configs/control_signals/opt_mode.yaml",
                       help="Path to control signal file")
    parser.add_argument("--interval", type=int, default=30,
                       help="Check interval in seconds")
    parser.add_argument("--moderate-threshold", type=float, default=300.0,
                       help="Moderate optimization threshold")
    parser.add_argument("--adaptive-threshold", type=float, default=375.0,
                       help="Adaptive optimization threshold") 
    parser.add_argument("--ultra-threshold", type=float, default=450.0,
                       help="Ultra optimization threshold")
    
    args = parser.parse_args()
    
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
    
    try:
        observer.run_observer(args.interval)
    except KeyboardInterrupt:
        print("\n🛑 Stopping observer...")
        observer.stop()

if __name__ == "__main__":
    main()