/*
package apisrv provides an implementation of the gRPC server defined in ../../../api/protobuf-spec/pb.proto.

Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

// Package apisrv provides the frontendapi service for Open Match.
package apisrv

import (
	"context"
	"errors"

	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/GoogleCloudPlatform/open-match/internal/expbo"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/serving"
	redishelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/ignorelist"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/playerindices"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/redispb"

	"github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"

	"github.com/gomodule/redigo/redis"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// frontendAPI implements pb.ApiServer, the server generated by compiling
// the protobuf, by fulfilling the pb.APIClient interface.
type frontendAPI struct {
	cfg    config.View
	pool   *redis.Pool
	logger *logrus.Entry
}

// Bind binds the gRPC endpoint to OpenMatchServer
func Bind(omSrv *serving.OpenMatchServer) {
	handler := &frontendAPI{
		cfg:    omSrv.Config,
		pool:   omSrv.RedisPool,
		logger: omSrv.Logger,
	}
	omSrv.GrpcServer.AddService(func(server *grpc.Server) {
		pb.RegisterFrontendServer(server, handler)
	})
	omSrv.GrpcServer.AddProxy(pb.RegisterFrontendHandler)
}

// CreatePlayer is this service's implementation of the CreatePlayer gRPC method defined in frontend.proto
func (s *frontendAPI) CreatePlayer(ctx context.Context, req *pb.CreatePlayerRequest) (*pb.CreatePlayerResponse, error) {
	group := req.Player
	// Write group
	err := redispb.MarshalToRedis(ctx, s.pool, group, s.cfg.GetInt("redis.expirations.player"))
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Error("State storage error")

		return nil, status.Error(codes.Unknown, err.Error())
	}

	// Index group
	err = playerindices.Create(ctx, s.pool, s.cfg, *group)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Error("State storage error")

		return nil, status.Error(codes.Unknown, err.Error())
	}

	// Return success.
	return &pb.CreatePlayerResponse{}, nil
}

// DeletePlayer is this service's implementation of the DeletePlayer gRPC method defined in frontend.proto
func (s *frontendAPI) DeletePlayer(ctx context.Context, req *pb.DeletePlayerRequest) (*pb.DeletePlayerResponse, error) {
	group := req.Player
	// Deindex this player; at that point they don't show up in MMFs anymore.  We can then delete
	// their actual player object from Redis later.
	err := playerindices.Delete(ctx, s.pool, s.cfg, group.Id)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Error("State storage error")

		return nil, status.Error(codes.Unknown, err.Error())
	}
	// Kick off delete but don't wait for it to complete.
	go s.deletePlayer(group.Id)

	return &pb.DeletePlayerResponse{}, nil
}

// deletePlayer is a 'lazy' player delete
// It should always be called as a goroutine and should only be called after
// confirmation that a player has been deindexed (and therefore MMF's can't
// find the player to read them anyway)
// As a final action, it also kicks off a lazy delete of the player's metadata
func (s *frontendAPI) deletePlayer(id string) {
	err := redishelpers.Delete(context.Background(), s.pool, id)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Warn("Error deleting player from state storage, this could leak state storage memory but is usually not a fatal error")
	}

	// Delete player from all ignorelists
	go func() {
		redisConn := s.pool.Get()
		defer redisConn.Close()

		redisConn.Send("MULTI")
		for il := range s.cfg.GetStringMap("ignoreLists") {
			ignorelist.SendRemove(redisConn, il, []string{id})
		}
		_, err := redisConn.Do("EXEC")
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"error":     err.Error(),
				"component": "statestorage",
			}).Error("Error de-indexing player from ignorelists")
		}
	}()

	go playerindices.DeleteMeta(context.Background(), s.pool, id)
}

// GetUpdates is this service's implementation of the GetUpdates gRPC method defined in frontend.proto
func (s *frontendAPI) GetUpdates(req *pb.GetUpdatesRequest, assignmentStream pb.Frontend_GetUpdatesServer) error {
	p := req.Player
	// Get cancellable context
	ctx, cancel := context.WithCancel(assignmentStream.Context())
	defer cancel()

	watcherBO := backoff.NewExponentialBackOff()
	if err := expbo.UnmarshalExponentialBackOff(s.cfg.GetString("api.pb.backoff"), watcherBO); err != nil {
		s.logger.WithError(err).Warn("Could not parse backoff string, using default backoff parameters for Player watcher")
	}

	// We have to stop Watcher manually because in a normal case client closes channel before the timeout
	watcherCtx, stopWatcher := context.WithCancel(context.Background())
	defer stopWatcher()
	watcherBOCtx := backoff.WithContext(watcherBO, watcherCtx)

	// get and return connection string
	watchChan := redispb.PlayerWatcher(watcherBOCtx, s.pool, *p) // watcher() runs the appropriate Redis commands.

	for {
		select {
		case <-ctx.Done():
			// Context canceled
			s.logger.WithField("playerid", p.Id).Info("client closed connection successfully")
			return nil

		case a, ok := <-watchChan:
			if !ok {
				// Timeout reached without client closing connection
				err := errors.New("server timeout reached without client closing connection")
				s.logger.WithFields(logrus.Fields{
					"error":     err.Error(),
					"component": "statestorage",
					"playerid":  p.Id,
				}).Error("State storage error")

				//TODO: we could generate a frontend.player message with an error
				//field and stream it to the client before throwing the error here
				//if we wanted to send more useful client retry information
				return status.Error(codes.DeadlineExceeded, err.Error())
			}

			s.logger.WithFields(logrus.Fields{
				"assignment": a.Assignment,
				"playerid":   a.Id,
				"status":     a.Status,
				"error":      a.Error,
			}).Info("updating client")
			assignmentStream.Send(&pb.GetUpdatesResponse{
				Player: &a,
			})
		}
	}
}
