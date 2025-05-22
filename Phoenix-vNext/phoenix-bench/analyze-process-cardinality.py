#!/usr/bin/env python3
"""
Phoenix-vNext Process-Focused Cardinality Analysis
Analyzes cardinality reduction achieved by process-specific optimization
"""

import requests
import re
import json
from collections import defaultdict
from datetime import datetime

def fetch_metrics(endpoint):
    """Fetch metrics from Prometheus endpoint"""
    try:
        response = requests.get(f"http://localhost:{endpoint}/metrics", timeout=10)
        response.raise_for_status()
        return response.text
    except Exception as e:
        print(f"Error fetching metrics from port {endpoint}: {e}")
        return ""

def analyze_metrics(metrics_text, pipeline_name):
    """Analyze metrics text and return cardinality information"""
    if not metrics_text:
        return {"total_series": 0, "unique_metrics": 0, "processes": set(), "metric_types": set()}
    
    lines = metrics_text.split('\n')
    series_count = 0
    unique_metrics = set()
    processes = set()
    metric_types = set()
    
    for line in lines:
        # Skip comments and empty lines
        if line.startswith('#') or not line.strip():
            continue
            
        # Count actual metric series
        if re.match(r'^phoenix_.*?_process_', line):
            series_count += 1
            
            # Extract metric name (before the labels)
            metric_match = re.match(r'^(phoenix_.*?_process_[^{\s]+)', line)
            if metric_match:
                metric_name = metric_match.group(1)
                unique_metrics.add(metric_name)
                
                # Extract base metric type
                base_type = metric_name.split('_')[-1] if '_' in metric_name else metric_name
                metric_types.add(base_type)
            
            # Extract process information from labels
            process_match = re.search(r'process_executable_name="([^"]+)"', line)
            if process_match:
                processes.add(process_match.group(1))
    
    return {
        "total_series": series_count,
        "unique_metrics": len(unique_metrics),
        "processes": processes,
        "metric_types": metric_types,
        "pipeline": pipeline_name
    }

def calculate_reduction(baseline, optimized):
    """Calculate cardinality reduction percentage"""
    if baseline == 0:
        return 0
    return round(((baseline - optimized) / baseline) * 100, 2)

def main():
    print("🔍 Phoenix-vNext Process-Focused Cardinality Analysis")
    print("=" * 60)
    
    # Define pipeline endpoints
    pipelines = {
        "Full (Baseline)": 8888,
        "Optimized": 8889, 
        "Ultra (Rollup)": 8890
    }
    
    results = {}
    
    # Analyze each pipeline
    for name, port in pipelines.items():
        print(f"\n📊 Analyzing {name} pipeline (port {port})...")
        metrics_text = fetch_metrics(port)
        analysis = analyze_metrics(metrics_text, name)
        results[name] = analysis
        
        print(f"   Total Time Series: {analysis['total_series']}")
        print(f"   Unique Metrics: {analysis['unique_metrics']}")
        print(f"   Processes Detected: {len(analysis['processes'])}")
        print(f"   Metric Types: {len(analysis['metric_types'])}")
    
    # Calculate reductions
    print("\n📈 Cardinality Reduction Analysis")
    print("-" * 40)
    
    baseline = results["Full (Baseline)"]["total_series"]
    optimized = results["Optimized"]["total_series"] 
    ultra = results["Ultra (Rollup)"]["total_series"]
    
    opt_reduction = calculate_reduction(baseline, optimized)
    ultra_reduction = calculate_reduction(baseline, ultra)
    
    print(f"Full Pipeline (Baseline): {baseline:,} time series")
    print(f"Optimized Pipeline: {optimized:,} time series ({opt_reduction}% reduction)")
    print(f"Ultra Pipeline (Rollup): {ultra:,} time series ({ultra_reduction}% reduction)")
    
    # Gap Analysis Assessment
    print("\n🎯 Gap Analysis Phase 1 Assessment")
    print("-" * 40)
    
    target_reduction = 60  # Target from gap analysis
    
    if ultra_reduction >= target_reduction:
        status = "✅ ACHIEVED"
    elif ultra_reduction >= target_reduction * 0.8:
        status = "⚠️  CLOSE (80%+ of target)"
    else:
        status = "❌ BELOW TARGET"
        
    print(f"Target Cardinality Reduction: {target_reduction}%")
    print(f"Achieved Reduction: {ultra_reduction}%")
    print(f"Status: {status}")
    
    # Process-Specific Analysis
    print("\n🔧 Process-Specific Optimization Evidence")
    print("-" * 40)
    
    all_processes = set()
    for result in results.values():
        all_processes.update(result['processes'])
    
    print(f"Total Processes Monitored: {len(all_processes)}")
    print(f"Process Focus: {'✅ YES' if len(all_processes) > 0 else '❌ NO'}")
    
    if len(all_processes) > 0:
        print("Sample Processes:")
        for i, process in enumerate(sorted(list(all_processes))[:5]):
            print(f"  • {process}")
        if len(all_processes) > 5:
            print(f"  • ... and {len(all_processes) - 5} more")
    
    # Save detailed report
    report = {
        "timestamp": datetime.now().isoformat(),
        "analysis_type": "process-focused-cardinality",
        "pipelines": results,
        "reductions": {
            "optimized_reduction_pct": opt_reduction,
            "ultra_reduction_pct": ultra_reduction,
            "target_reduction_pct": target_reduction,
            "target_achieved": ultra_reduction >= target_reduction
        },
        "processes": list(all_processes),
        "summary": {
            "baseline_series": baseline,
            "optimized_series": optimized,
            "ultra_series": ultra,
            "total_processes": len(all_processes)
        }
    }
    
    with open("process-cardinality-analysis.json", "w") as f:
        json.dump(report, f, indent=2, default=str)
    
    print(f"\n📄 Detailed report saved to: process-cardinality-analysis.json")
    
    # Final Assessment
    print("\n🏁 Final Assessment")
    print("=" * 60)
    
    if ultra_reduction >= target_reduction:
        print("✅ SUCCESS: Process-focused optimization achieves target cardinality reduction")
        print("✅ Phase 1 of gap analysis implementation is COMPLETE")
    else:
        print("⚠️  PARTIAL: Process focus implemented but target reduction not fully achieved")
        print("💡 Recommendation: Proceed with Phase 2 optimizations")

if __name__ == "__main__":
    main()