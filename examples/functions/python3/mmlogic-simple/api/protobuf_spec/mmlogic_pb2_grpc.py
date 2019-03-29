# Generated by the gRPC Python protocol compiler plugin. DO NOT EDIT!
import grpc

from api.protobuf_spec import messages_pb2 as api_dot_protobuf__spec_dot_messages__pb2


class MmLogicStub(object):
  """The MMLogic API provides utility functions for common MMF functionality, such
  as retreiving profiles and players from state storage, writing results to state storage,
  and exposing metrics and statistics.
  Profile and match object functions
  """

  def __init__(self, channel):
    """Constructor.

    Args:
      channel: A grpc.Channel.
    """
    self.GetProfile = channel.unary_unary(
        '/api.MmLogic/GetProfile',
        request_serializer=api_dot_protobuf__spec_dot_messages__pb2.MatchObject.SerializeToString,
        response_deserializer=api_dot_protobuf__spec_dot_messages__pb2.MatchObject.FromString,
        )
    self.CreateProposal = channel.unary_unary(
        '/api.MmLogic/CreateProposal',
        request_serializer=api_dot_protobuf__spec_dot_messages__pb2.MatchObject.SerializeToString,
        response_deserializer=api_dot_protobuf__spec_dot_messages__pb2.Result.FromString,
        )
    self.GetPlayerPool = channel.unary_stream(
        '/api.MmLogic/GetPlayerPool',
        request_serializer=api_dot_protobuf__spec_dot_messages__pb2.PlayerPool.SerializeToString,
        response_deserializer=api_dot_protobuf__spec_dot_messages__pb2.PlayerPool.FromString,
        )
    self.GetAllIgnoredPlayers = channel.unary_unary(
        '/api.MmLogic/GetAllIgnoredPlayers',
        request_serializer=api_dot_protobuf__spec_dot_messages__pb2.IlInput.SerializeToString,
        response_deserializer=api_dot_protobuf__spec_dot_messages__pb2.Roster.FromString,
        )
    self.ListIgnoredPlayers = channel.unary_unary(
        '/api.MmLogic/ListIgnoredPlayers',
        request_serializer=api_dot_protobuf__spec_dot_messages__pb2.IlInput.SerializeToString,
        response_deserializer=api_dot_protobuf__spec_dot_messages__pb2.Roster.FromString,
        )


class MmLogicServicer(object):
  """The MMLogic API provides utility functions for common MMF functionality, such
  as retreiving profiles and players from state storage, writing results to state storage,
  and exposing metrics and statistics.
  Profile and match object functions
  """

  def GetProfile(self, request, context):
    """Send GetProfile a match object with the ID field populated, it will return a
    'filled' one.
    Note: filters are assumed to have been checked for validity by the
    backendapi  when accepting a profile
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def CreateProposal(self, request, context):
    """CreateProposal is called by MMFs that wish to write their results to
    a proposed MatchObject, that can be sent out the Backend API once it has
    been approved (by default, by the evaluator process).
    - adds all players in all Rosters to the proposed player ignore list
    - writes the proposed match to the provided key
    - adds that key to the list of proposals to be considered
    INPUT: 
    * TO RETURN A MATCHOBJECT AFTER A SUCCESSFUL MMF RUN
    To create a match MatchObject message with these fields populated:
    - id, set to the value of the MMF_PROPOSAL_ID env var
    - properties
    - error.  You must explicitly set this to an empty string if your MMF
    - roster, with the playerIDs filled in the 'players' repeated field. 
    - [optional] pools, set to the output from the 'GetPlayerPools' call,
    will populate the pools with stats about how many players the filters
    matched and how long the filters took to run, which will be sent out
    the backend api along with your match results.
    was successful.
    * TO RETURN AN ERROR 
    To report a failure or error, send a MatchObject message with these
    these fields populated:
    - id, set to the value of the MMF_ERROR_ID env var. 
    - error, set to a string value describing the error your MMF encountered.
    - [optional] properties, anything you put here is returned to the
    backend along with your error.
    - [optional] rosters, anything you put here is returned to the
    backend along with your error.
    - [optional] pools, set to the output from the 'GetPlayerPools' call,
    will populate the pools with stats about how many players the filters
    matched and how long the filters took to run, which will be sent out
    the backend api along with your match results.
    OUTPUT: a Result message with a boolean success value and an error string
    if an error was encountered
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def GetPlayerPool(self, request, context):
    """Player listing and filtering functions

    RetrievePlayerPool gets the list of players that match every Filter in the
    PlayerPool, .excluding players in any configured ignore lists.  It
    combines the results, and returns the resulting player pool.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def GetAllIgnoredPlayers(self, request, context):
    """Ignore List functions

    IlInput is an empty message reserved for future use.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def ListIgnoredPlayers(self, request, context):
    """ListIgnoredPlayers retrieves players from the ignore list specified in the
    config file under 'ignoreLists.proposed.name'.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')


def add_MmLogicServicer_to_server(servicer, server):
  rpc_method_handlers = {
      'GetProfile': grpc.unary_unary_rpc_method_handler(
          servicer.GetProfile,
          request_deserializer=api_dot_protobuf__spec_dot_messages__pb2.MatchObject.FromString,
          response_serializer=api_dot_protobuf__spec_dot_messages__pb2.MatchObject.SerializeToString,
      ),
      'CreateProposal': grpc.unary_unary_rpc_method_handler(
          servicer.CreateProposal,
          request_deserializer=api_dot_protobuf__spec_dot_messages__pb2.MatchObject.FromString,
          response_serializer=api_dot_protobuf__spec_dot_messages__pb2.Result.SerializeToString,
      ),
      'GetPlayerPool': grpc.unary_stream_rpc_method_handler(
          servicer.GetPlayerPool,
          request_deserializer=api_dot_protobuf__spec_dot_messages__pb2.PlayerPool.FromString,
          response_serializer=api_dot_protobuf__spec_dot_messages__pb2.PlayerPool.SerializeToString,
      ),
      'GetAllIgnoredPlayers': grpc.unary_unary_rpc_method_handler(
          servicer.GetAllIgnoredPlayers,
          request_deserializer=api_dot_protobuf__spec_dot_messages__pb2.IlInput.FromString,
          response_serializer=api_dot_protobuf__spec_dot_messages__pb2.Roster.SerializeToString,
      ),
      'ListIgnoredPlayers': grpc.unary_unary_rpc_method_handler(
          servicer.ListIgnoredPlayers,
          request_deserializer=api_dot_protobuf__spec_dot_messages__pb2.IlInput.FromString,
          response_serializer=api_dot_protobuf__spec_dot_messages__pb2.Roster.SerializeToString,
      ),
  }
  generic_handler = grpc.method_handlers_generic_handler(
      'api.MmLogic', rpc_method_handlers)
  server.add_generic_rpc_handlers((generic_handler,))
