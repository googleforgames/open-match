#! /usr/bin/env python3
import random

def makeMatches(profile_dict, player_pools):
    ###########################################################################
    # This is the exciting part, and where most of your custom code would go! #
    ###########################################################################

    # for example, if your JSON format includes 'red' and 'blue' teams at the 'teams' key:
    for roster in profile_dict['properties']['rosters']:
        for player in roster['players']:
            if 'pool' in player:
                player['id']  = random.choice(list(player_pools[player['pool']]))
                print("Selected player %s from pool %s (strategy: RANDOM)" % (player['id'], player['pool']))
            else:
                print(player)
    return profile_dict
