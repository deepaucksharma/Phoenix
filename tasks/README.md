# Phoenix Project Tasks

This directory contains task definitions for the Phoenix project, organized by component and status.

## Directory Structure

- **components/** - Tasks organized by component
  - **pid/** - PID Controller related tasks
  - **processor/** - Processor component tasks
  - **extension/** - Extension component tasks
  - **util/** - Utility component tasks

- **status/** - Tasks organized by current status
  - **open/** - Tasks that haven't been started
  - **in_progress/** - Tasks currently being worked on
  - **completed/** - Tasks that have been completed

## Task Format

Tasks are defined in YAML files with the following format:

```yaml
task_id: "COMPONENT-NUMBER"
title: "Task Title"
description: |
  Detailed description of the task
assignee: "username"
status: "open|in_progress|completed"
priority: "high|medium|low"
tags:
  - "tag1"
  - "tag2"
dependencies:
  - "DEPENDENT-TASK-ID"
acceptance_criteria:
  - "Criterion 1"
  - "Criterion 2"
estimated_time: "XhYd"
created_at: "YYYY-MM-DD"
updated_at: "YYYY-MM-DD"
```

## Adding New Tasks

1. Create a new YAML file in the appropriate component directory
2. Update the status directory with a copy or symlink
3. Ensure the task ID follows the format: `COMPONENT-NUMBER` (e.g., `PID-004`)

## Tracking Progress

Task status should be kept updated by:

1. Moving the task file to the appropriate status directory
2. Updating the `status` field in the task YAML
3. Adding an `implementation_details` section to completed tasks

## Current Project Focus

The current focus is on implementing and stabilizing the PID control system, with the following key tasks:

- PID-001: Implement PID Controller for Adaptive Processing (Completed)
- PID-002: Implement Comprehensive Test Suite for PID Controller (In Progress)
- PID-003: Implement Policy-In-Code (PIC) Control Extension (In Progress)