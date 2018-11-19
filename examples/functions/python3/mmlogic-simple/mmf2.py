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

import os
import grpc

import customlogic

import mmlogic_pb2
import mmlogic_pb2_grpc

#import simplejson  as json
import ujson  as json
import pprint as pp

cfg = ''

# Load config file
with open("matchmaker_config.json") as f:
    cfg = json.loads(f.read())

api_conn_info = "%s:%d" % (cfg['api']['mmlogic']['hostname'], cfg['api']['mmlogic']['port'])

# Step 2 - Talk to Redis.  This example uses the MM Logic API in OM to read/write to/from redis.
with  grpc.insecure_channel(api_conn_info) as channel:
    mmlogic_api = mmlogic_pb2_grpc.APIStub(channel)

    # Step 3 - Read the profile written to the Backend API.
    profile_dict = json.loads(mmlogic_api.GetProfile(mmlogic_pb2.Profile(id=os.environ["MMF_PROFILE_ID"])))

    # Step 4 - Select the player data from Redis that we want for our matchmaking logic.
    player_pools = dict()       # holds pools returned by the associated filter
    for p in profile_dict['properties']['playerPools']:
        player_pools[p['id']] =mmlogic_api.GetPlayerPool(mmlogic_pb2.JsonFilterSet(id=p['id'],json=json.dumps(p)))

    # Step 5 - Run custom matchmaking logic to try to find a match
    # This is in the file customlogic.py
    match_properties = json.dumps(customlogic.makeMatches(profile_dict, player_pools))

    # Step 6 - Write the outcome of the matchmaking logic back to state storage.
    # Step 7 - Remove the selected players from consideration by other MMFs.
    # CreateProposal does both of these for you, and some other items as well.
    success = mmlogic_api.CreateProposal(mmlogic_pb2.MMFResults(
        id              = os.environ["MMF_PROPOSAL_ID"], 
        matchobject = mmlogic_pb2.MatchObject(
            properties  = match_properties,
            ),
        roster = mmlogic_pb2.Roster(
            id          = os.environ["MMF_ROSTER_ID"],
            player      = match_properties[cfg['jsonKeys']['roster']], 
            ),
        )
    )

    # [OPTIONAL] Step 8 - Export stats about this run.
