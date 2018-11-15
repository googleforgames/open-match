#! /usr/bin/env python3
# Step 1 - Package this in a linux container image.  Python runs fine in Linux.

import os
import grpc
import sys

import mmf 

import api.protobuf_spec.messages_pb2 as mmlogic
import api.protobuf_spec.mmlogic_pb2_grpc as mmlogic_pb2_grpc
from google.protobuf.json_format import Parse

import ujson  as json
import pprint as pp

from timeit import default_timer as timer

cfg = ''

# Load config file
with open("matchmaker_config.json") as f:
    cfg = json.loads(f.read())

api_conn_info = "%s:%d" % (cfg['api']['mmlogic']['hostname'], cfg['api']['mmlogic']['port'])

# Step 2 - Talk to Redis.  This example uses the MM Logic API in OM to read/write to/from redis.
# Establish grpc channel and make the API client stub
with  grpc.insecure_channel(api_conn_info) as channel:
    mmlogic_api = mmlogic_pb2_grpc.MmLogicStub(channel)

    # Step 3 - Read the profile written to the Backend API.
    # Get profile from redis
    profile_pb = mmlogic_api.GetProfile(mmlogic.MatchObject(id=os.environ["MMF_PROFILE_ID"]))
    pp.pprint(profile_pb)
    profile_dict = json.loads(profile_pb.properties)

    # Step 4 - Select the player data from Redis that we want for our matchmaking logic.
    # Embedded in this profile are JSON representations of the filters for each player pool.
    # JsonFilterSet() is able to read those directly.  No need to marhal that
    # JSON into the protobuf message format.
    player_pools = dict()       # holds dictionary version of the pools. 
    print("profile_pb", profile_pb)
    for empty_pool in profile_pb.pools:
        # Dict to hold value-sorted field dictionary for easy retreival of players by value
        player_pools[empty_pool.name] = dict() 
        print("Retrieving pool '%s'" % empty_pool.name, end='')
        start = timer()
        for partial_results in mmlogic_api.GetPlayerPool(empty_pool):
            empty_pool.stats.count = partial_results.stats.count
            empty_pool.stats.elapsed = partial_results.stats.elapsed
            print(".", end='')
            try:
                for player in partial_results.roster.players:
                    if not player.id in player_pools[empty_pool.name]:
                        player_pools[empty_pool.name][player.id] = dict()
                    for prop in player.properties:
                        player_pools[empty_pool.name][player.id][prop.name] = prop.value
            except Exception:
                raise
        print("\n'%s': count %06d | elapsed %0.3f" % (empty_pool.name, len(player_pools[empty_pool.name]),timer() - start))

    # Step 5 - Run custom matchmaking logic to try to find a match
    # This is in the file mmf.py
    pp.pprint(profile_dict)
    results = mmf.makeMatches(profile_dict, player_pools)
    match_properties = json.dumps(results)
    pp.pprint(match_properties) # DEBUG
    #mo = Parse(match_properties, mmlogic.MatchObject(), ignore_unknown_fields=True) 


#    # Step 6 - Write the outcome of the matchmaking logic back to state storage.
#    # Step 7 - Remove the selected players from consideration by other MMFs.
#    # CreateProposal does both of these for you, and some other items as well.
    mo = mmlogic.MatchObject(
        id = os.environ["MMF_PROPOSAL_ID"], 
        properties = match_properties,
        )
    # These look odd but it's how you assign to repeated protobuf fields.
    # https://developers.google.com/protocol-buffers/docs/reference/python-generated#repeated-fields
    mo.pools.extend(profile_pb.pools[:])
    for roster in results['properties']['rosters']:
        mo.rosters.extend([Parse(json.dumps(roster), mmlogic.Roster(), ignore_unknown_fields=True)])
    pp.pprint(mo)
    success = mmlogic_api.CreateProposal(mo)
    pp.pprint(success)
#
#    # [OPTIONAL] Step 8 - Export stats about this run.
#    # TODO 
#
