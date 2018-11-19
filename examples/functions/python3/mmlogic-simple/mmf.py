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

import random

def makeMatches(profile_dict, player_pools):
    ###########################################################################
    # This is the exciting part, and where most of your custom code would go! #
    ###########################################################################

    # The python3 MMF harness passed this function filtered players and their
    # filtered attributes in the player_pools dictionary.  If we wanted to evaluate
    # other player attributes, we could connect to redis directly and query the
    # players by their ID to get the entire 'properties' player JSON passed in
    # to the frontend API when they entered matchmaking.

    # This basic example just pulls players at random from the specified pools in the 
    # profile.  This just serves to show how the dictionaries are accessed and you 
    # should write your own rigourous logic here.
    for roster in profile_dict['properties']['rosters']:
        for player in roster['players']:
            if 'pool' in player:
                player['id']  = random.choice(list(player_pools[player['pool']]))
                print("Selected player %s from pool %s (strategy: RANDOM)" % (player['id'], player['pool']))
            else:
                print(player)
    return profile_dict
