#! /usr/bin/env python3
# Step 1 - Package this in a linux container image.  Python runs fine in Linux.

import os
import grpc

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
# Establish grpc channel and make the API client stub
with  grpc.insecure_channel(api_conn_info) as channel:
    mmlogic_api = mmlogic_pb2_grpc.APIStub(channel)

    # Step 3 - Read the profile written to the Backend API.
    # Get profile from redis
    profile_pb2 = mmlogic_api.GetProfile(mmlogic_pb2.Profile(id=os.environ["OM_PROFILE_ID"]))
    profile_dict = json.loads(profile_pb2.properties)

    # Step 4 - Select the player data from Redis that we want for our matchmaking logic.
    player_pool = mmlogic_api.GetPlayerPool()

    # Step 5 - Run custom matchmaking logic to try to find a match
    ###########################################################################
    # This is the exciting part, and where most of your custom code would go! #
    ###########################################################################
    results = customMatchmakerLogic(profile_dict, player_pool)

    # Step 6 - Write the outcome of the matchmaking logic back to state storage.
    # Step 7 - Remove the selected players from consideration by other MMFs.
    # CreateProposal does both of these for you, and some other items as well.
    match_properties = json.dumps(results)
    mo = mmlogic_pb2.MatchObject(id=os.environ["OM_MATCHOBJECT_ID"], properties=match_properties)
    success = CreateProposal(mo)

    # [OPTIONAL] Step 8 - Export stats about this run.
    # TODO 

### DEBUG
pp.pprint(profile)
pp.pprint(this)
print(api_conn_info)
print("yep")
#pp.pprint(cfg)
