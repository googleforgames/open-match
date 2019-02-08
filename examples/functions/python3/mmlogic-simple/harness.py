#! /usr/bin/env python3
#Copyright 2018 Google LLC
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    https://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.
# Note:
# This harness exits with a success code even in cases of error as  
# kubernetes jobs will re-run until it sees a successful exit code.
# Errors are populated through the backend API back to the backend API client.
import os
import grpc
import mmf 
import api.protobuf_spec.messages_pb2 as mmlogic
import api.protobuf_spec.mmlogic_pb2_grpc as mmlogic_pb2_grpc
from google.protobuf.json_format import Parse
import ujson  as json
import pprint as pp

from timeit import default_timer as timer

# Step 2 - Talk to Redis.  This example uses the MM Logic API in OM to read/write to/from redis.
# Establish grpc channel and make the API client stub
api_conn_info = "%s:%d" % (os.environ["OM_MMLOGICAPI_SERVICE_HOST"],os.environ["OM_MMLOGICAPI_SERVICE_PORT"])
with  grpc.insecure_channel(api_conn_info) as channel:
    mmlogic_api = mmlogic_pb2_grpc.MmLogicStub(channel)

    # Step 3 - Read the profile written to the Backend API.
    # Get profile from redis
    profile_pb = mmlogic_api.GetProfile(mmlogic.MatchObject(id=os.environ["MMF_PROFILE_ID"]))
    pp.pprint(profile_pb) #DEBUG
    profile_dict = json.loads(profile_pb.properties)

    # Step 4 - Select the player data from Redis that we want for our matchmaking logic.
    # Embedded in this profile are JSON representations of the filters for each player pool.
    # JsonFilterSet() is able to read those directly.  No need to marhal that
    # JSON into the protobuf message format.
    player_pools = dict()       # holds dictionary version of the pools. 
    for empty_pool in profile_pb.pools:
        # Dict to hold value-sorted field dictionary for easy retreival of players by value
        player_pools[empty_pool.name] = dict() 
        print("Retrieving pool '%s'" % empty_pool.name, end='')

        # DEBUG: Print how long the filtering takes
        if os.environ["DEBUG"]:
            start = timer()

        # Pool filter results are streamed in chunks as they can be too large to send
        # in one grpc message.  Loop to get them all.
        for partial_results in mmlogic_api.GetPlayerPool(empty_pool):
            empty_pool.stats.count = partial_results.stats.count
            empty_pool.stats.elapsed = partial_results.stats.elapsed
            print(".", end='')
            try:
                for player in partial_results.roster.players:
                    if not player.id in player_pools[empty_pool.name]:
                        player_pools[empty_pool.name][player.id] = dict()
                    for attr in player.attributes:
                        player_pools[empty_pool.name][player.id][attr.name] = attr.value
            except Exception as err:
                print("Error encountered: %s" % err) 
        if os.environ["DEBUG"]:
            print("\n'%s': count %06d | elapsed %0.3f" % (empty_pool.name, len(player_pools[empty_pool.name]),timer() - start))

    num_players_total = 0
    for _, pool in player_pools.items():
        num_players_total += len(pool)

    #################################################################
    # Step 5 - Run custom matchmaking logic to try to find a match
    # This is in the file mmf.py
    results = mmf.makeMatches(profile_dict, player_pools)
    #################################################################

    # DEBUG
    if os.environ["DEBUG"]:
        print("======== match_properties")
        pp.pprint(results) 

    # Generate a MatchObject message to write to state storage with the results in it.
    # This looks odd but it's how you assign to repeated protobuf fields.
    # https://developers.google.com/protocol-buffers/docs/reference/python-generated#repeated-fields
    match_properties = json.dumps(results)
    mo = mmlogic.MatchObject(id = os.environ["MMF_PROPOSAL_ID"], properties = match_properties)
    mo.pools.extend(profile_pb.pools[:])

    # Access the rosters in dict form within the properties json.
    # It is stored at the key specified in the config file.
    rosters_dict = results
    for partial_key in os.environ["JSONKEYS_ROSTERS"].split('.'):
        rosters_dict = rosters_dict.get(partial_key, {})

    # Unmarshal the rosters into the MatchObject 
    for roster in rosters_dict:
        mo.rosters.extend([Parse(json.dumps(roster), mmlogic.Roster(), ignore_unknown_fields=True)])

    #DEBUG: writing to error key prevents evalutor run 
    if os.environ["DEBUG"]:
        #mo.id = os.environ["MMF_ERROR_ID"] 
        #mo.error = "skip evaluator"
        print("======== MMF results:") 
        pp.pprint(mo)

    # Return error when there are no players in the pools
    if num_players_total == 0:
        if cfg['debug']:
            print("All player pools are empty, writing to error to skip the evaluator")
        mo.id = os.environ["MMF_ERROR_ID"]
        mo.error = "insufficient players"

    # Step 6 - Write the outcome of the matchmaking logic back to state storage.    
    # Step 7 - Remove the selected players from consideration by other MMFs.
    # CreateProposal does both of these for you, and some other items as well.
    success = mmlogic_api.CreateProposal(mo)
    print("======== MMF write to state storage:  %s" % success) 

    # [OPTIONAL] Step 8 - Export stats about this run.
    # TODO 
