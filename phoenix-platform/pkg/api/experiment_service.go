package api

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/phoenix/platform/pkg/api/v1"
	"github.com/phoenix/platform/pkg/generator"
	"github.com/phoenix/platform/pkg/models"
	"github.com/phoenix/platform/pkg/store"
	"github.com/phoenix/platform/pkg/utils"
)

type ExperimentService struct {
	pb.UnimplementedExperimentServiceServer
	store     store.ExperimentStore
	generator generator.Service
	logger    *zap.Logger
}

func NewExperimentService(store store.ExperimentStore, generator generator.Service, logger *zap.Logger) *ExperimentService {
	return &ExperimentService{
		store:     store,
		generator: generator,
		logger:    logger,
	}
}

func (s *ExperimentService) CreateExperiment(ctx context.Context, req *pb.CreateExperimentRequest) (*pb.CreateExperimentResponse, error) {
	s.logger.Info("creating experiment", zap.String("name", req.Spec.Name))

	// Validate request
	if err := s.validateExperimentSpec(req.Spec); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid spec: %v", err)
	}

	// Get user from context
	user, ok := ctx.Value("user").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	// Create experiment
	exp := &models.Experiment{
		ID:          utils.GenerateID("exp"),
		Name:        req.Spec.Name,
		Description: req.Spec.Description,
		Owner:       user,
		Spec:        req.Spec,
		Status: &pb.ExperimentStatus{
			Phase:   pb.ExperimentStatus_PHASE_PENDING,
			Message: "Experiment created",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to store
	if err := s.store.CreateExperiment(ctx, exp); err != nil {
		s.logger.Error("failed to create experiment", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create experiment: %v", err)
	}

	// Trigger async generation
	go s.generateArtifacts(exp)

	return &pb.CreateExperimentResponse{
		ExperimentId: exp.ID,
		Status:       exp.Status.Phase.String(),
	}, nil
}

func (s *ExperimentService) GetExperiment(ctx context.Context, req *pb.GetExperimentRequest) (*pb.Experiment, error) {
	exp, err := s.store.GetExperiment(ctx, req.ExperimentId)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Error(codes.NotFound, "experiment not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get experiment: %v", err)
	}

	// Check permissions
	user, _ := ctx.Value("user").(string)
	if exp.Owner != user && !s.isAdmin(ctx) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	return s.modelToProto(exp), nil
}

func (s *ExperimentService) ListExperiments(ctx context.Context, req *pb.ListExperimentsRequest) (*pb.ListExperimentsResponse, error) {
	// Get user from context
	user, _ := ctx.Value("user").(string)
	
	// Build filter
	filter := store.ExperimentFilter{
		Owner:  req.Owner,
		Status: req.Status,
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	}

	// Non-admins can only see their own experiments
	if !s.isAdmin(ctx) {
		filter.Owner = user
	}

	experiments, total, err := s.store.ListExperiments(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list experiments: %v", err)
	}

	// Convert to proto
	pbExperiments := make([]*pb.Experiment, len(experiments))
	for i, exp := range experiments {
		pbExperiments[i] = s.modelToProto(exp)
	}

	return &pb.ListExperimentsResponse{
		Experiments: pbExperiments,
		Total:       int32(total),
	}, nil
}

func (s *ExperimentService) UpdateExperiment(ctx context.Context, req *pb.UpdateExperimentRequest) (*pb.Experiment, error) {
	// Get existing experiment
	exp, err := s.store.GetExperiment(ctx, req.ExperimentId)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Error(codes.NotFound, "experiment not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get experiment: %v", err)
	}

	// Check permissions
	user, _ := ctx.Value("user").(string)
	if exp.Owner != user && !s.isAdmin(ctx) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Check if experiment is running
	if exp.Status.Phase == pb.ExperimentStatus_PHASE_RUNNING {
		return nil, status.Error(codes.FailedPrecondition, "cannot update running experiment")
	}

	// Update fields
	if req.Spec != nil {
		if err := s.validateExperimentSpec(req.Spec); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid spec: %v", err)
		}
		exp.Spec = req.Spec
		exp.UpdatedAt = time.Now()
	}

	// Save to store
	if err := s.store.UpdateExperiment(ctx, exp); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update experiment: %v", err)
	}

	return s.modelToProto(exp), nil
}

func (s *ExperimentService) DeleteExperiment(ctx context.Context, req *pb.DeleteExperimentRequest) (*pb.DeleteExperimentResponse, error) {
	// Get existing experiment
	exp, err := s.store.GetExperiment(ctx, req.ExperimentId)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Error(codes.NotFound, "experiment not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get experiment: %v", err)
	}

	// Check permissions
	user, _ := ctx.Value("user").(string)
	if exp.Owner != user && !s.isAdmin(ctx) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Check if experiment is running
	if exp.Status.Phase == pb.ExperimentStatus_PHASE_RUNNING {
		return nil, status.Error(codes.FailedPrecondition, "cannot delete running experiment")
	}

	// Delete from store
	if err := s.store.DeleteExperiment(ctx, req.ExperimentId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete experiment: %v", err)
	}

	// Cleanup resources
	go s.cleanupExperimentResources(exp)

	return &pb.DeleteExperimentResponse{Success: true}, nil
}

func (s *ExperimentService) GetExperimentStatus(ctx context.Context, req *pb.GetExperimentStatusRequest) (*pb.ExperimentStatus, error) {
	exp, err := s.store.GetExperiment(ctx, req.ExperimentId)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Error(codes.NotFound, "experiment not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get experiment: %v", err)
	}

	// Check permissions
	user, _ := ctx.Value("user").(string)
	if exp.Owner != user && !s.isAdmin(ctx) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	return exp.Status, nil
}

func (s *ExperimentService) StreamExperimentUpdates(req *pb.StreamExperimentUpdatesRequest, stream pb.ExperimentService_StreamExperimentUpdatesServer) error {
	ctx := stream.Context()
	
	// Get experiment to check permissions
	exp, err := s.store.GetExperiment(ctx, req.ExperimentId)
	if err != nil {
		if err == store.ErrNotFound {
			return status.Error(codes.NotFound, "experiment not found")
		}
		return status.Errorf(codes.Internal, "failed to get experiment: %v", err)
	}

	// Check permissions
	user, _ := ctx.Value("user").(string)
	if exp.Owner != user && !s.isAdmin(ctx) {
		return status.Error(codes.PermissionDenied, "access denied")
	}

	// Subscribe to updates
	subscription := s.store.Subscribe(req.ExperimentId)
	defer subscription.Close()

	s.logger.Info("streaming updates for experiment", 
		zap.String("experiment_id", req.ExperimentId),
		zap.String("user", user))

	// Stream updates
	for {
		select {
		case update := <-subscription.Updates():
			if update == nil {
				return nil
			}

			// Convert to proto
			pbUpdate := &pb.ExperimentUpdate{
				ExperimentId: req.ExperimentId,
				Status:       update.Status,
				Metrics:      make(map[string]*pb.MetricValue),
				Timestamp:    timestamppb.Now(),
			}

			for k, v := range update.Metrics {
				pbUpdate.Metrics[k] = &pb.MetricValue{
					Value: v.Value,
					Unit:  v.Unit,
				}
			}

			if err := stream.Send(pbUpdate); err != nil {
				s.logger.Error("failed to send update", zap.Error(err))
				return err
			}

		case <-ctx.Done():
			s.logger.Info("stream closed by client", zap.String("experiment_id", req.ExperimentId))
			return nil
		}
	}
}

func (s *ExperimentService) PromoteVariant(ctx context.Context, req *pb.PromoteVariantRequest) (*pb.PromoteVariantResponse, error) {
	// Get experiment
	exp, err := s.store.GetExperiment(ctx, req.ExperimentId)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Error(codes.NotFound, "experiment not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get experiment: %v", err)
	}

	// Check permissions
	user, _ := ctx.Value("user").(string)
	if exp.Owner != user && !s.isAdmin(ctx) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Check if experiment is completed
	if exp.Status.Phase != pb.ExperimentStatus_PHASE_COMPLETED {
		return nil, status.Error(codes.FailedPrecondition, "experiment must be completed before promotion")
	}

	// Validate variant
	validVariant := false
	for _, v := range exp.Spec.Variants {
		if v.Name == req.Variant {
			validVariant = true
			break
		}
	}
	if !validVariant {
		return nil, status.Errorf(codes.InvalidArgument, "invalid variant: %s", req.Variant)
	}

	// TODO: Implement promotion logic
	// This would typically:
	// 1. Create a PR to update production configuration
	// 2. Update monitoring dashboards
	// 3. Archive the experiment

	s.logger.Info("promoting variant",
		zap.String("experiment_id", req.ExperimentId),
		zap.String("variant", req.Variant),
		zap.String("user", user))

	return &pb.PromoteVariantResponse{
		Success: true,
		Message: fmt.Sprintf("Variant %s promoted successfully", req.Variant),
	}, nil
}

// Helper methods

func (s *ExperimentService) validateExperimentSpec(spec *pb.ExperimentSpec) error {
	if spec == nil {
		return fmt.Errorf("spec is required")
	}

	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(spec.Variants) != 2 {
		return fmt.Errorf("exactly 2 variants required (baseline and candidate)")
	}

	// Validate variants
	hasBaseline := false
	hasCandidate := false
	for _, v := range spec.Variants {
		if v.Name == "baseline" {
			hasBaseline = true
		} else if v.Name == "candidate" {
			hasCandidate = true
		}

		if len(v.Pipeline.Nodes) == 0 {
			return fmt.Errorf("variant %s must have at least one processor node", v.Name)
		}
	}

	if !hasBaseline || !hasCandidate {
		return fmt.Errorf("must have both baseline and candidate variants")
	}

	return nil
}

func (s *ExperimentService) generateArtifacts(exp *models.Experiment) {
	ctx := context.Background()
	
	// Update status
	exp.Status.Phase = pb.ExperimentStatus_PHASE_GENERATING
	exp.Status.Message = "Generating pipeline configurations"
	s.store.UpdateExperiment(ctx, exp)

	// Generate artifacts
	if err := s.generator.GenerateArtifacts(ctx, exp); err != nil {
		s.logger.Error("failed to generate artifacts", 
			zap.String("experiment_id", exp.ID),
			zap.Error(err))
		
		exp.Status.Phase = pb.ExperimentStatus_PHASE_FAILED
		exp.Status.Message = fmt.Sprintf("Generation failed: %v", err)
		s.store.UpdateExperiment(ctx, exp)
		return
	}

	// Update status
	exp.Status.Phase = pb.ExperimentStatus_PHASE_DEPLOYING
	exp.Status.Message = "Deploying pipelines"
	s.store.UpdateExperiment(ctx, exp)

	// TODO: Wait for deployment to complete
	// This would monitor ArgoCD or Kubernetes for readiness
}

func (s *ExperimentService) cleanupExperimentResources(exp *models.Experiment) {
	// TODO: Implement cleanup
	// This would:
	// 1. Delete Kubernetes resources
	// 2. Clean up Git branches
	// 3. Archive metrics data
	s.logger.Info("cleaning up experiment resources", zap.String("experiment_id", exp.ID))
}

func (s *ExperimentService) isAdmin(ctx context.Context) bool {
	claims, ok := ctx.Value("claims").(map[string]interface{})
	if !ok {
		return false
	}

	roles, ok := claims["roles"].([]string)
	if !ok {
		return false
	}

	for _, role := range roles {
		if role == "admin" {
			return true
		}
	}

	return false
}

func (s *ExperimentService) modelToProto(exp *models.Experiment) *pb.Experiment {
	return &pb.Experiment{
		Id:          exp.ID,
		Name:        exp.Name,
		Description: exp.Description,
		Owner:       exp.Owner,
		Spec:        exp.Spec,
		Status:      exp.Status,
		CreatedAt:   timestamppb.New(exp.CreatedAt),
		UpdatedAt:   timestamppb.New(exp.UpdatedAt),
	}
}