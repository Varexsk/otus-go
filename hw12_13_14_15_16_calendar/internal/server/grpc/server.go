package internalgrpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/app"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/server/grpc/pb"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrInvalidArgument = errors.New("invalid argument")

type Application interface {
	CreateEvent(ctx context.Context, event storage.Event) (uuid.UUID, error)
	UpdateEvent(ctx context.Context, id uuid.UUID, event storage.Event) error
	DeleteEvent(ctx context.Context, id uuid.UUID) error
	ListDay(ctx context.Context, date time.Time) ([]storage.Event, error)
	ListWeek(ctx context.Context, start time.Time) ([]storage.Event, error)
	ListMonth(ctx context.Context, month time.Time) ([]storage.Event, error)
}

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

var _ pb.EventServiceServer = (*Server)(nil)

type Server struct {
	pb.UnimplementedEventServiceServer
	app    Application
	logg   Logger
	addr   string
	server *grpc.Server
}

func New(logg Logger, app Application, addr string) *Server {
	s := &Server{logg: logg, app: app, addr: addr}

	s.server = grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor(logg)),
	)
	pb.RegisterEventServiceServer(s.server, s)
	return s
}

func (s *Server) Start(_ context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	s.logg.Info("starting grpc server", "addr", s.addr)
	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logg.Info("stopping grpc server")

	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.logg.Info("grpc server stopped")
		return nil
	case <-ctx.Done():
		s.server.Stop()
		return fmt.Errorf("graceful stop: %w", ctx.Err())
	}
}

func (s *Server) CreateEvent(ctx context.Context, req *pb.CreateEventRequest) (*pb.CreateEventResponse, error) {
	event, err := toEvent(req.GetEvent())
	if err != nil {
		return nil, s.toStatusError(err)
	}

	id, err := s.app.CreateEvent(ctx, event)
	if err != nil {
		return nil, s.toStatusError(err)
	}

	return &pb.CreateEventResponse{
		Id: id.String(),
	}, nil
}

func (s *Server) UpdateEvent(ctx context.Context, req *pb.UpdateEventRequest) (*pb.UpdateEventResponse, error) {
	id := req.GetId()
	uid, err := parseID(id)
	if err != nil {
		return nil, s.toStatusError(err)
	}

	event, err := toEvent(req.GetEvent())
	if err != nil {
		return nil, s.toStatusError(err)
	}

	if err := s.app.UpdateEvent(ctx, uid, event); err != nil {
		return nil, s.toStatusError(err)
	}

	return &pb.UpdateEventResponse{}, nil
}

func (s *Server) DeleteEvent(ctx context.Context, req *pb.DeleteEventRequest) (*pb.DeleteEventResponse, error) {
	id := req.GetId()
	uid, err := parseID(id)
	if err != nil {
		return nil, s.toStatusError(err)
	}

	if err := s.app.DeleteEvent(ctx, uid); err != nil {
		return nil, s.toStatusError(err)
	}

	return &pb.DeleteEventResponse{}, nil
}

func (s *Server) ListDay(ctx context.Context, req *pb.ListDayRequest) (*pb.ListDayResponse, error) {
	events, err := s.listEvents(ctx, req.GetDate(), s.app.ListDay)
	if err != nil {
		return nil, err
	}
	return &pb.ListDayResponse{Events: events}, nil
}

func (s *Server) ListWeek(ctx context.Context, req *pb.ListWeekRequest) (*pb.ListWeekResponse, error) {
	events, err := s.listEvents(ctx, req.GetStart(), s.app.ListWeek)
	if err != nil {
		return nil, err
	}
	return &pb.ListWeekResponse{Events: events}, nil
}

func (s *Server) ListMonth(ctx context.Context, req *pb.ListMonthRequest) (*pb.ListMonthResponse, error) {
	events, err := s.listEvents(ctx, req.GetMonth(), s.app.ListMonth)
	if err != nil {
		return nil, err
	}
	return &pb.ListMonthResponse{Events: events}, nil
}

type listFunc func(ctx context.Context, date time.Time) ([]storage.Event, error)

func (s *Server) listEvents(ctx context.Context, ts *timestamppb.Timestamp, list listFunc) ([]*pb.Event, error) {
	if err := ts.CheckValid(); err != nil {
		return nil, s.toStatusError(fmt.Errorf("%w: date: %w", ErrInvalidArgument, err))
	}

	events, err := list(ctx, ts.AsTime())
	if err != nil {
		return nil, s.toStatusError(err)
	}

	return toPBEvents(events), nil
}

func parseID(id string) (uuid.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: id: %w", ErrInvalidArgument, err)
	}
	return uid, nil
}

func toPBEvents(events []storage.Event) []*pb.Event {
	pbEvents := make([]*pb.Event, len(events))
	for i, e := range events {
		pbEvents[i] = &pb.Event{
			Id:           e.ID.String(),
			Title:        e.Title,
			Description:  e.Description,
			UserId:       e.UserID,
			StartAt:      timestamppb.New(e.StartAt),
			EndAt:        timestamppb.New(e.EndAt),
			NotifyBefore: durationpb.New(e.NotifyBefore),
		}
	}

	return pbEvents
}

func toEvent(pbEvent *pb.Event) (storage.Event, error) {
	startAt := pbEvent.GetStartAt()
	if err := startAt.CheckValid(); err != nil {
		return storage.Event{}, fmt.Errorf("%w: start_at: %w", ErrInvalidArgument, err)
	}
	endAt := pbEvent.GetEndAt()
	if err := endAt.CheckValid(); err != nil {
		return storage.Event{}, fmt.Errorf("%w: end_at: %w", ErrInvalidArgument, err)
	}
	return storage.Event{
		Title:        pbEvent.GetTitle(),
		Description:  pbEvent.GetDescription(),
		UserID:       pbEvent.GetUserId(),
		StartAt:      startAt.AsTime(),
		EndAt:        endAt.AsTime(),
		NotifyBefore: pbEvent.GetNotifyBefore().AsDuration(),
	}, nil
}

func (s *Server) toStatusError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, storage.ErrEventNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, storage.ErrEventAlreadyExist):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, storage.ErrDateBusy):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, app.ErrInvalidEvent):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		s.logg.Error("internal error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
