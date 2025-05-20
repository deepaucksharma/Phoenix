package pic_control_ext

// RegisterSafetyMonitor registers a safety monitor with the extension
func (e *Extension) RegisterSafetyMonitor(monitor safetyMonitor) {
	e.lock.Lock()
	defer e.lock.Unlock()
	
	e.safetyMonitor = monitor
	
	// Update safe mode status from monitor
	if monitor != nil {
		e.safeMode = monitor.IsInSafeMode()
	}
}

// SetSafeMode manually sets the safe mode status for testing
func (e *Extension) SetSafeMode(safeMode bool) {
	e.lock.Lock()
	defer e.lock.Unlock()
	
	e.safeMode = safeMode
}