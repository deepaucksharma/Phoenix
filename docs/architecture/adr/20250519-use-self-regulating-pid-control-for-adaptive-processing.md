# ADR-002: Use Self-Regulating PID Control for Adaptive Processing

Date: 2025-05-19

## Context

The SA-OMF (Phoenix) system needs an adaptive mechanism to automatically adjust processor parameters based on observed metrics and KPIs. We need a robust and proven control mechanism that can:

1. Effectively adapt processor parameters without manual intervention
2. Ensure stability and prevent oscillation or thrashing
3. Provide predictable behavior in diverse operating conditions
4. Handle various types of KPIs (coverage scores, resource utilization, cardinality, etc.)
5. Operate efficiently without excessive overhead

We need to select a control algorithm that is well-understood, provides stability guarantees, and can be tuned for different scenarios.

## Decision

We will implement a comprehensive PID (Proportional-Integral-Derivative) controller with enhanced safety mechanisms as the primary adaptation mechanism for Phoenix:

1. **Core PID Controller Implementation**:
   - Standard P, I, D terms for balanced and configurable control
   - Thread-safe implementation to support concurrent metrics collection and parameter adjustment
   - Comprehensive metrics emission for monitoring and debugging

2. **Enhanced Stability Mechanisms**:
   - **Anti-windup protection** to prevent integral term saturation when output is limited 
   - **Low-pass derivative filtering** to reduce noise sensitivity in the derivative term
   - **Hysteresis** to prevent parameter oscillation around the target value
   - **Oscillation detection and circuit breaking** to detect and mitigate unstable behavior
   
3. **Fallback Optimization Support**:
   - For complex parameter spaces or when PID control stalls, fallback to Bayesian optimization
   - Gaussian Process (GP) regression model with anisotropic RBF kernel
   - Latin Hypercube Sampling for efficient parameter space exploration
   - Expected Improvement acquisition function balancing exploration/exploitation

4. **Centralized Control in adaptive_pid processor**:
   - Monitor KPIs from system metrics
   - Apply PID control algorithms to determine parameter adjustments
   - Generate configuration updates to be applied to target processors

## Consequences

### Positive

- **Industrial-Grade Control Theory**: PID controllers are well-understood and widely used in control systems across industries.
- **Stability Guarantees**: Properly tuned PID controllers provide mathematical guarantees about stability.
- **Self-healing Capability**: The system can automatically correct for parameter drift and react to changing conditions.
- **Multi-layered Safety**: Enhanced safety mechanisms like circuit breakers, anti-windup, and derivative filtering provide robustness.
- **Adaptive Optimization**: The Bayesian optimization fallback handles complex, non-linear parameter spaces when PID is insufficient.
- **Observable Behavior**: All control actions are exposed as metrics for monitoring and debugging.

### Negative

- **Tuning Complexity**: PID controllers require careful tuning of kp, ki, kd parameters, which can be challenging.
- **Single-parameter Focus**: Each PID controller typically manages one parameter, making multi-parameter optimization more complex.
- **Learning Curve**: Understanding and configuring PID control may be unfamiliar to some users.
- **Computational Overhead**: The Bayesian optimization fallback involves more complex calculations than basic PID.

### Mitigations

- **Default Tunings**: Provide reasonable default PID parameters for common scenarios.
- **Automatic Tuning Helpers**: Implement utilities to suggest PID parameters based on observed system behavior.
- **Circuit Breaker Mechanism**: Automatically detect and prevent unstable oscillations.
- **Clear Documentation**: Provide comprehensive guides for PID tuning and adaptive parameter configuration.
- **Configuration Templates**: Offer pre-configured settings for different workloads and environments.
- **Gradual Integration**: Start with basic PID control and only activate advanced features as needed.

## Implementation Details

1. **Controller Structure**:
   ```go
   type Controller struct {
       // PID constants
       kp float64 // Proportional gain
       ki float64 // Integral gain
       kd float64 // Derivative gain

       // State
       setpoint      float64   // Target value
       lastError     float64   // Last error value
       integral      float64   // Accumulated error
       
       // Safety features
       integralLimit float64           // Maximum absolute value for integral term
       outputMin     float64           // Minimum output value
       outputMax     float64           // Maximum output value
       antiWindupEnabled bool          // Whether anti-windup protection is enabled
       derivativeFilterCoeff float64   // Coefficient for derivative filtering
       circuitBreaker        *OscillationDetector // Detects and prevents oscillations
   }
   ```

2. **Oscillation Detection**:
   ```go
   type OscillationDetector struct {
       // Configuration
       sampleWindow                int       // Number of samples to track
       oscillationThresholdPercent float64   // Percentage of zero crossings required
       minSignalMagnitude          float64   // Minimum magnitude for significance
       minDuration                 time.Duration // Min duration before tripping
       resetDuration               time.Duration // Auto-reset duration
       
       // State
       signalHistory      []float64    // History of signal values
       isTripped          bool         // Whether circuit breaker is active
   }
   ```

3. **Bayesian Optimization**:
   ```go
   type Optimizer struct {
       gp         *GaussianProcess
       bounds     [][2]float64
       candidates int
       explorationWeight float64    // Weight for exploration vs. exploitation
       lenScales         []float64  // Length scales for each dimension
   }
   ```

## Alternative Approaches Considered

1. **Simple Threshold-based Adaptation**: Rejected due to inability to handle complex dynamics and tendency to oscillate.

2. **Machine Learning Models**: Neural networks or other ML models for parameter prediction. Rejected due to:
   - High complexity and resource requirements
   - Need for training data
   - Less explainable behavior
   - Potential for unexpected behavior in edge cases

3. **Direct Bayesian Optimization**: Using Bayesian optimization as the primary control method. Rejected as the main approach because:
   - Higher computational overhead
   - Requires more exploration samples to converge
   - Less effective for simple, well-behaved parameters
   - Better used as a fallback for complex parameter spaces

4. **Fixed Rules and Heuristics**: Using static rules for adaptation. Rejected due to inflexibility and inability to self-tune.

## References

- PID Controller Theory: https://en.wikipedia.org/wiki/PID_controller
- Anti-Windup Techniques: https://en.wikipedia.org/wiki/Integral_windup
- Bayesian Optimization: https://arxiv.org/abs/1807.02811
- Oscillation Detection in Control Systems: https://www.sciencedirect.com/science/article/pii/S2405896318304567