package pic_control_ext

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

// MockUpdateableProcessor is a mock implementation of interfaces.UpdateableProcessor
type MockUpdateableProcessor struct {
	mock.Mock
}

func (m *MockUpdateableProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	args := m.Called(ctx, patch)
	return args.Error(0)
}

func (m *MockUpdateableProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(interfaces.ConfigStatus), args.Error(1)
}

func (m *MockUpdateableProcessor) GetName() string {
	args := m.Called()
	return args.String(0)
}

// MockSafetyMonitor is a mock implementation of interfaces.SafetyMonitor
type MockSafetyMonitor struct {
	mock.Mock
}

func (m *MockSafetyMonitor) IsInSafeMode() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSafetyMonitor) GetSafeModeEnterTime() *time.Time {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	t := args.Get(0).(time.Time)
	return &t
}

func (m *MockSafetyMonitor) GetCurrentCPUThreshold() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

// MockTargetID implements interfaces.TargetID for testing
type MockTargetID struct {
	name string
	typ  string
}

func (m MockTargetID) String() string {
	return m.typ + "/" + m.name
}

func (m MockTargetID) Type() string {
	return m.typ
}

func (m MockTargetID) Name() string {
	return m.name
}

func newMockTargetID(typ, name string) MockTargetID {
	return MockTargetID{
		typ:  typ,
		name: name,
	}
}

// TestPicControlExtensionCreation tests that the extension can be created
func TestPicControlExtensionCreation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	factory := pic_control_ext.NewFactory()
	cfg := factory.CreateDefaultConfig()

	ext, err := factory.CreateExtension(context.Background(), componenttest.NewNopExtensionCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)
}

// TestRegisterUpdateableProcessor tests registering a processor
func TestRegisterUpdateableProcessor(t *testing.T) {
	// Create the extension
	logger := zaptest.NewLogger(t)
	factory := pic_control_ext.NewFactory()
	cfg := factory.CreateDefaultConfig()

	ext, err := factory.CreateExtension(context.Background(), componenttest.NewNopExtensionCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)

	// Create a mock processor
	mockProcessor := new(MockUpdateableProcessor)
	mockProcessor.On("GetName").Return("test_processor")

	// Cast to PicControl interface
	picControl, ok := ext.(interfaces.PicControl)
	require.True(t, ok, "extension does not implement PicControl interface")

	// Register the processor
	err = picControl.RegisterUpdateableProcessor(mockProcessor)
	require.NoError(t, err)

	// Verify
	mockProcessor.AssertCalled(t, "GetName")
}

// TestApplyConfigPatch tests applying a config patch
func TestApplyConfigPatch(t *testing.T) {
	// Create the extension
	logger := zaptest.NewLogger(t)
	factory := pic_control_ext.NewFactory()
	cfg := factory.CreateDefaultConfig()

	ext, err := factory.CreateExtension(context.Background(), componenttest.NewNopExtensionCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)

	// Create a mock processor
	mockProcessor := new(MockUpdateableProcessor)
	mockProcessor.On("GetName").Return("test_processor")
	mockProcessor.On("OnConfigPatch", mock.Anything, mock.Anything).Return(nil)

	// Cast to PicControl interface
	picControl, ok := ext.(interfaces.PicControl)
	require.True(t, ok, "extension does not implement PicControl interface")

	// Register the processor
	err = picControl.RegisterUpdateableProcessor(mockProcessor)
	require.NoError(t, err)

	// Create a config patch
	patch := interfaces.ConfigPatch{
		TargetProcessorName: newMockTargetID("processor", "test_processor"),
		ParameterPath:       "enabled",
		NewValue:            true,
		Reason:              "Unit test",
	}

	// Apply the patch
	err = picControl.ApplyConfigPatch(context.Background(), patch)
	require.NoError(t, err)

	// Verify
	mockProcessor.AssertCalled(t, "OnConfigPatch", mock.Anything, mock.Anything)
}

// TestSafeModeRejection tests that patches are rejected in safe mode
func TestSafeModeRejection(t *testing.T) {
	// Create the extension
	logger := zaptest.NewLogger(t)
	factory := pic_control_ext.NewFactory()
	cfg := factory.CreateDefaultConfig()

	ext, err := factory.CreateExtension(context.Background(), componenttest.NewNopExtensionCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)

	// Create a mock processor
	mockProcessor := new(MockUpdateableProcessor)
	mockProcessor.On("GetName").Return("test_processor")

	// Cast to PicControl interface
	picControl, ok := ext.(interfaces.PicControl)
	require.True(t, ok, "extension does not implement PicControl interface")

	// Register the processor
	err = picControl.RegisterUpdateableProcessor(mockProcessor)
	require.NoError(t, err)

	// Set safe mode
	picControl.SetSafeMode(true)

	// Create a config patch
	patch := interfaces.ConfigPatch{
		TargetProcessorName: newMockTargetID("processor", "test_processor"),
		ParameterPath:       "enabled",
		NewValue:            true,
		Reason:              "Unit test",
		SafetyOverride:      false,
	}

	// Apply the patch - should be rejected
	err = picControl.ApplyConfigPatch(context.Background(), patch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "safe mode")

	// Create a patch with safety override
	overridePatch := interfaces.ConfigPatch{
		TargetProcessorName: newMockTargetID("processor", "test_processor"),
		ParameterPath:       "enabled",
		NewValue:            true,
		Reason:              "Unit test with override",
		SafetyOverride:      true,
	}

	// Mock the processor to accept the patch
	mockProcessor.On("OnConfigPatch", mock.Anything, mock.Anything).Return(nil)

	// Apply the patch with override - should be accepted
	err = picControl.ApplyConfigPatch(context.Background(), overridePatch)
	require.NoError(t, err)
}
