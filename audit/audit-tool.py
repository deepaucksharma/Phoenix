#!/usr/bin/env python3
"""
Phoenix Audit Tracking Tool

This tool helps manage the component-by-component audit for the Phoenix project.
It allows:
- Initializing the audit directory structure
- Updating audit status
- Generating reports
- Visualizing audit progress
"""

import os
import sys
import yaml
import glob
import argparse
import datetime
from pathlib import Path
from typing import List, Dict, Any, Optional

# YAML utilities
def load_yaml(file_path: str) -> Dict:
    """Load a YAML file."""
    try:
        with open(file_path, 'r') as f:
            return yaml.safe_load(f) or {}
    except Exception as e:
        print(f"Error loading {file_path}: {e}")
        return {}

def save_yaml(file_path: str, data: Dict) -> None:
    """Save data to a YAML file."""
    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    with open(file_path, 'w') as f:
        yaml.dump(data, f, default_flow_style=False, sort_keys=False)

# Audit Management Functions
def initialize_audit_structure(base_dir: str) -> None:
    """Create the audit directory structure and initial files."""
    # Create main directories
    directories = [
        "components/processors",
        "components/extensions", 
        "components/connectors",
        "interfaces",
        "algorithms",
        "configurations"
    ]
    
    for directory in directories:
        os.makedirs(os.path.join(base_dir, directory), exist_ok=True)
    
    # Create summary file
    summary = {
        "last_updated": datetime.datetime.now().isoformat(),
        "status": {
            "not_started": 0,
            "in_progress": 0,
            "completed": 0,
            "total": 0
        },
        "priority_issues": [],
        "audit_progress": 0.0
    }
    save_yaml(os.path.join(base_dir, "summary.yaml"), summary)
    
    print(f"Initialized audit structure in {base_dir}")

def discover_components(project_root: str) -> List[Dict[str, str]]:
    """Discover components in the project structure."""
    components = []
    
    # Processors
    for path in glob.glob(f"{project_root}/internal/processor/*/"):
        name = os.path.basename(os.path.dirname(path))
        components.append({
            "name": name,
            "type": "processors",
            "path": os.path.relpath(path, project_root)
        })
    
    # Extensions
    for path in glob.glob(f"{project_root}/internal/extension/*/"):
        name = os.path.basename(os.path.dirname(path))
        components.append({
            "name": name,
            "type": "extensions",
            "path": os.path.relpath(path, project_root)
        })
    
    # Connectors
    for path in glob.glob(f"{project_root}/internal/connector/*/"):
        name = os.path.basename(os.path.dirname(path))
        components.append({
            "name": name,
            "type": "connectors",
            "path": os.path.relpath(path, project_root)
        })
    
    return components

def create_component_audit_files(audit_dir: str, components: List[Dict[str, str]]) -> None:
    """Create audit files for each component."""
    for comp in components:
        audit_file = os.path.join(audit_dir, "components", comp["type"], f"{comp['name']}.yaml")
        
        # Skip if file already exists
        if os.path.exists(audit_file):
            continue
            
        data = {
            "component": {
                "name": comp["name"],
                "type": comp["type"],
                "path": comp["path"],
            },
            "audit_status": {
                "state": "Not Started",
                "owner": "",
                "start_date": None,
                "completion_date": None,
            },
            "quality_metrics": {
                "test_coverage": None,
                "cyclomatic_complexity": None,
                "linting_issues": None,
                "security_score": None,
            },
            "compliance": {
                "updateable_processor": None,
                "error_handling": None,
                "thread_safety": None,
                "documentation": None,
            },
            "performance": {
                "memory_usage": None,
                "cpu_usage": None,
                "scalability": None,
                "bottlenecks": None,
            },
            "findings": {
                "issues": [],
                "recommendations": [],
            }
        }
        
        save_yaml(audit_file, data)
    
    print(f"Created audit files for {len(components)} components")

def update_component_status(audit_dir: str, component_name: str, 
                           status: str, owner: Optional[str] = None) -> None:
    """Update the status of a component audit."""
    # Find component file
    component_files = []
    for path in glob.glob(f"{audit_dir}/components/*/{component_name}.yaml"):
        component_files.append(path)
    
    if not component_files:
        print(f"Component '{component_name}' not found in audit files")
        return
        
    if len(component_files) > 1:
        print(f"Warning: Multiple audit files found for '{component_name}', updating all")
    
    for file_path in component_files:
        data = load_yaml(file_path)
        
        # Update status
        data["audit_status"]["state"] = status
        
        # Set owner if provided
        if owner:
            data["audit_status"]["owner"] = owner
            
        # Set dates
        if status == "In Progress" and not data["audit_status"]["start_date"]:
            data["audit_status"]["start_date"] = datetime.datetime.now().isoformat()
        elif status == "Completed" and not data["audit_status"]["completion_date"]:
            data["audit_status"]["completion_date"] = datetime.datetime.now().isoformat()
            
        save_yaml(file_path, data)
    
    print(f"Updated status of '{component_name}' to '{status}'")
    update_summary(audit_dir)

def add_finding(audit_dir: str, component_name: str, severity: str, 
               description: str, location: Optional[str] = None,
               remediation: Optional[str] = None) -> None:
    """Add a finding to a component audit."""
    # Find component file
    component_files = []
    for path in glob.glob(f"{audit_dir}/components/*/{component_name}.yaml"):
        component_files.append(path)
    
    if not component_files:
        print(f"Component '{component_name}' not found in audit files")
        return
        
    if len(component_files) > 1:
        print(f"Warning: Multiple audit files found for '{component_name}', updating first one")
        
    file_path = component_files[0]
    data = load_yaml(file_path)
    
    # Create finding
    finding = {
        "severity": severity,
        "description": description,
    }
    
    if location:
        finding["location"] = location
        
    if remediation:
        finding["remediation"] = remediation
        
    # Add finding
    if "issues" not in data["findings"]:
        data["findings"]["issues"] = []
        
    data["findings"]["issues"].append(finding)
    save_yaml(file_path, data)
    
    print(f"Added {severity} finding to '{component_name}'")
    
    # Update summary for high/critical findings
    if severity.lower() in ["high", "critical"]:
        update_summary(audit_dir)

def add_recommendation(audit_dir: str, component_name: str, recommendation: str) -> None:
    """Add a recommendation to a component audit."""
    # Find component file
    component_files = []
    for path in glob.glob(f"{audit_dir}/components/*/{component_name}.yaml"):
        component_files.append(path)
    
    if not component_files:
        print(f"Component '{component_name}' not found in audit files")
        return
        
    if len(component_files) > 1:
        print(f"Warning: Multiple audit files found for '{component_name}', updating first one")
        
    file_path = component_files[0]
    data = load_yaml(file_path)
    
    # Add recommendation
    if "recommendations" not in data["findings"]:
        data["findings"]["recommendations"] = []
        
    data["findings"]["recommendations"].append(recommendation)
    save_yaml(file_path, data)
    
    print(f"Added recommendation to '{component_name}'")

def update_summary(audit_dir: str) -> None:
    """Update the audit summary file."""
    summary_path = os.path.join(audit_dir, "summary.yaml")
    summary = load_yaml(summary_path)
    
    # Count status
    status_count = {
        "not_started": 0,
        "in_progress": 0,
        "completed": 0,
        "total": 0
    }
    
    # Collect high/critical findings
    priority_issues = []
    
    # Process all component files
    for file_path in glob.glob(f"{audit_dir}/components/*/*/*.yaml"):
        data = load_yaml(file_path)
        component_name = data["component"]["name"]
        state = data["audit_status"]["state"].lower().replace(" ", "_")
        status_count[state] = status_count.get(state, 0) + 1
        status_count["total"] += 1
        
        # Check for priority issues
        if "issues" in data["findings"]:
            for issue in data["findings"]["issues"]:
                if issue["severity"].lower() in ["high", "critical"]:
                    priority_issues.append({
                        "component": component_name,
                        "severity": issue["severity"],
                        "description": issue["description"]
                    })
    
    # Update summary
    summary["status"] = status_count
    if status_count["total"] > 0:
        summary["audit_progress"] = (status_count["completed"] / status_count["total"]) * 100
    else:
        summary["audit_progress"] = 0
        
    summary["priority_issues"] = priority_issues
    summary["last_updated"] = datetime.datetime.now().isoformat()
    
    save_yaml(summary_path, summary)
    print("Updated audit summary")

def print_report(audit_dir: str) -> None:
    """Print a report of the audit status."""
    summary = load_yaml(os.path.join(audit_dir, "summary.yaml"))
    
    print("\n=== Phoenix Audit Status Report ===")
    print(f"Last Updated: {summary['last_updated']}")
    print(f"Progress: {summary['audit_progress']:.1f}%")
    print("\nComponent Status:")
    print(f"  Completed:   {summary['status']['completed']}/{summary['status']['total']}")
    print(f"  In Progress: {summary['status']['in_progress']}/{summary['status']['total']}")
    print(f"  Not Started: {summary['status']['not_started']}/{summary['status']['total']}")
    
    print("\nPriority Issues:")
    if summary["priority_issues"]:
        for issue in summary["priority_issues"]:
            print(f"  [{issue['severity']}] {issue['component']}: {issue['description']}")
    else:
        print("  No priority issues found")
    
    print("\nComponent Details:")
    for component_type in ["processors", "extensions", "connectors"]:
        print(f"\n{component_type.title()}:")
        
        for file_path in glob.glob(f"{audit_dir}/components/{component_type}/*.yaml"):
            data = load_yaml(file_path)
            status = data["audit_status"]["state"]
            owner = data["audit_status"]["owner"] or "Unassigned"
            issue_count = len(data["findings"]["issues"]) if "issues" in data["findings"] else 0
            
            status_symbol = {
                "Completed": "✅",
                "In Progress": "🔄",
                "Not Started": "⏱️"
            }.get(status, "❓")
            
            print(f"  {status_symbol} {data['component']['name']} - {status} - Owner: {owner}, Issues: {issue_count}")

def export_html_report(audit_dir: str, output_file: str) -> None:
    """Export an HTML report of the audit status."""
    summary = load_yaml(os.path.join(audit_dir, "summary.yaml"))
    
    # Collect component data
    components = []
    for component_type in ["processors", "extensions", "connectors"]:
        for file_path in glob.glob(f"{audit_dir}/components/{component_type}/*.yaml"):
            components.append(load_yaml(file_path))
    
    # Generate HTML
    html = f"""
    <!DOCTYPE html>
    <html>
    <head>
        <title>Phoenix Audit Report</title>
        <style>
            body {{ font-family: Arial, sans-serif; margin: 20px; }}
            h1, h2 {{ color: #333; }}
            .progress-bar {{ 
                width: 100%; 
                background-color: #e0e0e0; 
                border-radius: 5px; 
                margin: 10px 0; 
            }}
            .progress {{ 
                height: 20px; 
                background-color: #4CAF50; 
                border-radius: 5px; 
                width: {summary['audit_progress']}%; 
            }}
            table {{ border-collapse: collapse; width: 100%; margin-top: 20px; }}
            th, td {{ border: 1px solid #ddd; padding: 8px; text-align: left; }}
            th {{ background-color: #f2f2f2; }}
            tr:nth-child(even) {{ background-color: #f9f9f9; }}
            .status-not-started {{ color: gray; }}
            .status-in-progress {{ color: blue; }}
            .status-completed {{ color: green; }}
            .severity-critical {{ color: darkred; font-weight: bold; }}
            .severity-high {{ color: red; }}
            .severity-medium {{ color: orange; }}
            .severity-low {{ color: green; }}
        </style>
    </head>
    <body>
        <h1>Phoenix Audit Status Report</h1>
        <p>Last Updated: {summary['last_updated']}</p>
        
        <h2>Overall Progress: {summary['audit_progress']:.1f}%</h2>
        <div class="progress-bar">
            <div class="progress"></div>
        </div>
        
        <p>
            <strong>Completed:</strong> {summary['status']['completed']}/{summary['status']['total']} |
            <strong>In Progress:</strong> {summary['status']['in_progress']}/{summary['status']['total']} |
            <strong>Not Started:</strong> {summary['status']['not_started']}/{summary['status']['total']}
        </p>
        
        <h2>Priority Issues</h2>
    """
    
    if summary["priority_issues"]:
        html += """
        <table>
            <tr>
                <th>Severity</th>
                <th>Component</th>
                <th>Description</th>
            </tr>
        """
        
        for issue in summary["priority_issues"]:
            severity_class = f"severity-{issue['severity'].lower()}"
            html += f"""
            <tr>
                <td class="{severity_class}">{issue['severity']}</td>
                <td>{issue['component']}</td>
                <td>{issue['description']}</td>
            </tr>
            """
            
        html += "</table>"
    else:
        html += "<p>No priority issues found</p>"
    
    html += """
        <h2>Component Status</h2>
        <table>
            <tr>
                <th>Component</th>
                <th>Type</th>
                <th>Status</th>
                <th>Owner</th>
                <th>Issues</th>
                <th>Test Coverage</th>
            </tr>
    """
    
    for data in sorted(components, key=lambda x: x["component"]["name"]):
        component_name = data["component"]["name"]
        component_type = data["component"]["type"]
        status = data["audit_status"]["state"]
        owner = data["audit_status"]["owner"] or "Unassigned"
        issue_count = len(data["findings"]["issues"]) if "issues" in data["findings"] else 0
        test_coverage = data["quality_metrics"]["test_coverage"] or "N/A"
        
        status_class = f"status-{status.lower().replace(' ', '-')}"
        
        html += f"""
        <tr>
            <td>{component_name}</td>
            <td>{component_type}</td>
            <td class="{status_class}">{status}</td>
            <td>{owner}</td>
            <td>{issue_count}</td>
            <td>{test_coverage}</td>
        </tr>
        """
    
    html += """
        </table>
    </body>
    </html>
    """
    
    with open(output_file, 'w') as f:
        f.write(html)
        
    print(f"Generated HTML report at {output_file}")

def main():
    parser = argparse.ArgumentParser(description='Phoenix Audit Tracking Tool')
    subparsers = parser.add_subparsers(dest='command', help='Audit commands')
    
    # Initialize command
    init_parser = subparsers.add_parser('init', help='Initialize audit structure')
    init_parser.add_argument('--project-root', default='.', help='Project root directory')
    init_parser.add_argument('--audit-dir', default='audit', help='Audit directory')
    
    # Update status command
    status_parser = subparsers.add_parser('status', help='Update component status')
    status_parser.add_argument('component', help='Component name')
    status_parser.add_argument('state', choices=['Not Started', 'In Progress', 'Completed'], 
                             help='Audit status')
    status_parser.add_argument('--owner', help='Audit owner')
    status_parser.add_argument('--audit-dir', default='audit', help='Audit directory')
    
    # Add issue command
    issue_parser = subparsers.add_parser('issue', help='Add an issue finding')
    issue_parser.add_argument('component', help='Component name')
    issue_parser.add_argument('severity', choices=['Critical', 'High', 'Medium', 'Low'], 
                            help='Issue severity')
    issue_parser.add_argument('description', help='Issue description')
    issue_parser.add_argument('--location', help='Code location (file:line)')
    issue_parser.add_argument('--remediation', help='Remediation steps')
    issue_parser.add_argument('--audit-dir', default='audit', help='Audit directory')
    
    # Add recommendation command
    rec_parser = subparsers.add_parser('recommend', help='Add a recommendation')
    rec_parser.add_argument('component', help='Component name')
    rec_parser.add_argument('recommendation', help='Recommendation text')
    rec_parser.add_argument('--audit-dir', default='audit', help='Audit directory')
    
    # Report command
    report_parser = subparsers.add_parser('report', help='Generate audit report')
    report_parser.add_argument('--audit-dir', default='audit', help='Audit directory')
    report_parser.add_argument('--format', choices=['text', 'html'], default='text',
                              help='Report format')
    report_parser.add_argument('--output', default='audit-report.html',
                              help='Output file for HTML report')
    
    args = parser.parse_args()
    
    # Handle commands
    if args.command == 'init':
        init_dir = os.path.join(args.project_root, args.audit_dir)
        initialize_audit_structure(init_dir)
        components = discover_components(args.project_root)
        create_component_audit_files(init_dir, components)
        update_summary(init_dir)
        
    elif args.command == 'status':
        update_component_status(args.audit_dir, args.component, args.state, args.owner)
        
    elif args.command == 'issue':
        add_finding(args.audit_dir, args.component, args.severity, 
                   args.description, args.location, args.remediation)
        
    elif args.command == 'recommend':
        add_recommendation(args.audit_dir, args.component, args.recommendation)
        
    elif args.command == 'report':
        if args.format == 'text':
            print_report(args.audit_dir)
        else:  # html
            export_html_report(args.audit_dir, args.output)
    
    else:
        parser.print_help()

if __name__ == "__main__":
    main()